// webrtc.js – WebRTC, signaling & screen-share logic for Bro Media

(function () {
    "use strict";

    var config = window.BRO_CONFIG;

    var localStream = null;
    var screenStream = null;
    var ws = null;
    var currentRoom = null;
    var iceServers = [];
    var screenSharing = false;

    // Map of peerId -> { pc: RTCPeerConnection, stream: MediaStream }
    var peers = {};

    // ---- Public API ----

    window.BroRTC = {
        init: init,
        joinRoom: joinRoom,
        leaveRoom: leaveRoom,
        toggleAudio: toggleAudio,
        toggleVideo: toggleVideo,
        toggleScreenShare: toggleScreenShare,
        sendChat: sendChat,
        callUser: callUser,
        acceptCall: acceptCall,
        rejectCall: rejectCall,
        getPeers: function () { return Object.keys(peers); },
        isScreenSharing: function () { return screenSharing; },
        // Callbacks – set by app.js
        onRemoteStream: null,
        onPeerRemoved: null,
        onPeerCountChange: null,
        onOnlineUsers: null,
        onChatMessage: null,
        onCallIncoming: null,
        onCallAccepted: null,
        onCallRejected: null,
        onScreenShareStopped: null,
    };

    // ---- Initialisation ----

    async function fetchICEServers() {
        try {
            var res = await fetch(config.backendURL + "/api/ice-servers");
            var data = await res.json();
            iceServers = data.iceServers || [];
        } catch (err) {
            console.warn("[webrtc] failed to fetch ICE servers, using defaults:", err);
            iceServers = [{ urls: "stun:stun.l.google.com:19302" }];
        }
    }

    async function init() {
        await fetchICEServers();
        localStream = await navigator.mediaDevices.getUserMedia({ video: true, audio: true });
        connectWebSocket();
        return { localStream: localStream };
    }

    function connectWebSocket() {
        var token = window.BroAuth.getToken();
        if (!token) return;

        var wsEndpoint = config.wsURL + "/ws?token=" + encodeURIComponent(token);
        ws = new WebSocket(wsEndpoint);

        ws.onopen = function () { console.log("[ws] connected"); };
        ws.onmessage = function (evt) { handleSignalingMessage(JSON.parse(evt.data)); };
        ws.onclose = function () { console.log("[ws] disconnected"); };
        ws.onerror = function (err) { console.error("[ws] error:", err); };
    }

    // ---- Room management ----

    function joinRoom(roomId) {
        currentRoom = roomId;
        send({ type: "join", roomId: currentRoom });
    }

    function leaveRoom() {
        if (currentRoom && ws && ws.readyState === WebSocket.OPEN) {
            send({ type: "leave", roomId: currentRoom });
        }
        Object.keys(peers).forEach(removePeer);
        if (screenStream) {
            screenStream.getTracks().forEach(function (t) { t.stop(); });
            screenStream = null;
            screenSharing = false;
        }
        currentRoom = null;
    }

    // ---- Media toggles ----

    function toggleAudio() {
        if (!localStream) return false;
        var track = localStream.getAudioTracks()[0];
        if (track) { track.enabled = !track.enabled; return track.enabled; }
        return false;
    }

    function toggleVideo() {
        if (!localStream) return false;
        var track = localStream.getVideoTracks()[0];
        if (track) { track.enabled = !track.enabled; return track.enabled; }
        return false;
    }

    async function toggleScreenShare() {
        if (screenSharing) {
            // Stop screen share – revert to camera
            if (screenStream) {
                screenStream.getTracks().forEach(function (t) { t.stop(); });
                screenStream = null;
            }
            screenSharing = false;
            var camTrack = localStream.getVideoTracks()[0];
            replaceTrackInPeers(camTrack);
            return false;
        }
        try {
            screenStream = await navigator.mediaDevices.getDisplayMedia({ video: true });
            screenSharing = true;
            var screenTrack = screenStream.getVideoTracks()[0];
            replaceTrackInPeers(screenTrack);
            screenTrack.onended = function () {
                screenSharing = false;
                screenStream = null;
                replaceTrackInPeers(localStream.getVideoTracks()[0]);
                if (window.BroRTC.onScreenShareStopped) window.BroRTC.onScreenShareStopped();
            };
            return true;
        } catch (err) {
            console.warn("[webrtc] screen share cancelled:", err);
            return false;
        }
    }

    function replaceTrackInPeers(newTrack) {
        Object.values(peers).forEach(function (pd) {
            var senders = pd.pc.getSenders();
            var videoSender = senders.find(function (s) { return s.track && s.track.kind === "video"; });
            if (videoSender && newTrack) videoSender.replaceTrack(newTrack);
        });
    }

    // ---- Chat ----

    function sendChat(text) {
        if (!currentRoom || !text.trim()) return;
        send({ type: "chat", roomId: currentRoom, text: text.trim() });
    }

    // ---- Direct calling ----

    function callUser(targetUsername) {
        send({ type: "call-user", targetId: targetUsername });
    }

    function acceptCall(callerUsername, roomId) {
        send({ type: "call-accept", targetId: callerUsername, roomId: roomId });
    }

    function rejectCall(callerUsername) {
        send({ type: "call-reject", targetId: callerUsername });
    }

    // ---- Transport ----

    function send(msg) {
        if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg));
    }

    // ---- Signaling dispatcher ----

    function handleSignalingMessage(msg) {
        switch (msg.type) {
            case "peer-joined":   handlePeerJoined(msg.senderId); break;
            case "peer-left":     removePeer(msg.senderId); break;
            case "offer":         handleOffer(msg); break;
            case "answer":        handleAnswer(msg); break;
            case "ice-candidate": handleICECandidate(msg); break;

            case "online-users":
                if (window.BroRTC.onOnlineUsers) window.BroRTC.onOnlineUsers(JSON.parse(msg.payload));
                break;
            case "chat":
                if (window.BroRTC.onChatMessage) window.BroRTC.onChatMessage(msg);
                break;
            case "call-incoming":
                if (window.BroRTC.onCallIncoming) window.BroRTC.onCallIncoming(msg.senderId, msg.roomId);
                break;
            case "call-accepted":
                if (window.BroRTC.onCallAccepted) window.BroRTC.onCallAccepted(msg.senderId, msg.roomId);
                break;
            case "call-rejected":
                if (window.BroRTC.onCallRejected) window.BroRTC.onCallRejected(msg.senderId);
                break;
            default:
                console.warn("[webrtc] unknown message type:", msg.type);
        }
    }

    // ---- Peer Connection management ----

    function createPeerConnection(peerId) {
        var pc = new RTCPeerConnection({ iceServers: iceServers });

        // Add local tracks (audio always from camera, video may be screen)
        if (localStream) {
            localStream.getAudioTracks().forEach(function (t) { pc.addTrack(t, localStream); });
        }
        if (screenSharing && screenStream) {
            screenStream.getVideoTracks().forEach(function (t) { pc.addTrack(t, screenStream); });
        } else if (localStream) {
            localStream.getVideoTracks().forEach(function (t) { pc.addTrack(t, localStream); });
        }

        pc.ontrack = function (event) {
            var pd = peers[peerId];
            if (!pd) return;
            if (!pd.stream) pd.stream = new MediaStream();
            pd.stream.addTrack(event.track);
            if (window.BroRTC.onRemoteStream) window.BroRTC.onRemoteStream(peerId, pd.stream);
        };

        pc.onicecandidate = function (event) {
            if (event.candidate) {
                send({
                    type: "ice-candidate",
                    roomId: currentRoom,
                    targetId: peerId,
                    payload: JSON.parse(JSON.stringify(event.candidate)),
                });
            }
        };

        pc.onconnectionstatechange = function () {
            if (pc.connectionState === "disconnected" || pc.connectionState === "failed") {
                removePeer(peerId);
            }
        };

        peers[peerId] = { pc: pc, stream: null };
        return pc;
    }

    async function handlePeerJoined(peerId) {
        if (peers[peerId]) return;
        var pc = createPeerConnection(peerId);
        try {
            var offer = await pc.createOffer();
            await pc.setLocalDescription(offer);
            send({ type: "offer", roomId: currentRoom, targetId: peerId, payload: pc.localDescription });
        } catch (err) {
            console.error("[webrtc] createOffer error:", err);
        }
        if (window.BroRTC.onPeerCountChange) window.BroRTC.onPeerCountChange(Object.keys(peers).length);
    }

    async function handleOffer(msg) {
        var pc = peers[msg.senderId] ? peers[msg.senderId].pc : createPeerConnection(msg.senderId);
        try {
            await pc.setRemoteDescription(new RTCSessionDescription(msg.payload));
            var answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            send({ type: "answer", roomId: currentRoom, targetId: msg.senderId, payload: pc.localDescription });
        } catch (err) {
            console.error("[webrtc] handleOffer error:", err);
        }
        if (window.BroRTC.onPeerCountChange) window.BroRTC.onPeerCountChange(Object.keys(peers).length);
    }

    async function handleAnswer(msg) {
        var pd = peers[msg.senderId];
        if (!pd) return;
        try { await pd.pc.setRemoteDescription(new RTCSessionDescription(msg.payload)); }
        catch (err) { console.error("[webrtc] handleAnswer error:", err); }
    }

    async function handleICECandidate(msg) {
        var pd = peers[msg.senderId];
        if (!pd) return;
        try { await pd.pc.addIceCandidate(new RTCIceCandidate(msg.payload)); }
        catch (err) { console.error("[webrtc] addIceCandidate error:", err); }
    }

    function removePeer(peerId) {
        var pd = peers[peerId];
        if (pd) { pd.pc.close(); delete peers[peerId]; }
        if (window.BroRTC.onPeerRemoved) window.BroRTC.onPeerRemoved(peerId);
        if (window.BroRTC.onPeerCountChange) window.BroRTC.onPeerCountChange(Object.keys(peers).length);
    }
})();
