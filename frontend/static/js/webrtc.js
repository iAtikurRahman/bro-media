// webrtc.js – WebRTC & signaling logic for Bro Media

(function () {
    "use strict";

    const config = window.BRO_CONFIG;

    let localStream = null;
    let ws = null;
    let clientId = null;
    let currentRoom = null;
    let iceServers = [];

    // Map of peerId -> { pc: RTCPeerConnection, stream: MediaStream }
    const peers = {};

    // ---- Public API exposed to app.js ----

    window.BroRTC = {
        init,
        joinRoom,
        leaveRoom,
        toggleAudio,
        toggleVideo,
        getPeers: () => Object.keys(peers),
    };

    // Fetch ICE server configuration from backend
    async function fetchICEServers() {
        try {
            const res = await fetch(config.backendURL + "/api/ice-servers");
            const data = await res.json();
            iceServers = data.iceServers || [];
        } catch (err) {
            console.warn("[webrtc] failed to fetch ICE servers, using defaults:", err);
            iceServers = [{ urls: "stun:stun.l.google.com:19302" }];
        }
    }

    // Initialize: get media and ICE config
    async function init() {
        clientId = "user-" + Math.random().toString(36).substring(2, 9);

        await fetchICEServers();

        localStream = await navigator.mediaDevices.getUserMedia({
            video: true,
            audio: true,
        });

        return { clientId, localStream };
    }

    // Connect WebSocket and join a room
    function joinRoom(roomId) {
        currentRoom = roomId;
        const wsEndpoint = config.wsURL + "/ws?clientId=" + encodeURIComponent(clientId);

        ws = new WebSocket(wsEndpoint);

        ws.onopen = function () {
            console.log("[ws] connected");
            send({ type: "join", roomId: currentRoom });
        };

        ws.onmessage = function (evt) {
            const msg = JSON.parse(evt.data);
            handleSignalingMessage(msg);
        };

        ws.onclose = function () {
            console.log("[ws] disconnected");
        };

        ws.onerror = function (err) {
            console.error("[ws] error:", err);
        };
    }

    // Leave room and clean up
    function leaveRoom() {
        if (ws && ws.readyState === WebSocket.OPEN) {
            send({ type: "leave", roomId: currentRoom });
            ws.close();
        }

        Object.keys(peers).forEach(removePeer);

        if (localStream) {
            localStream.getTracks().forEach(function (t) { t.stop(); });
            localStream = null;
        }
        currentRoom = null;
    }

    function toggleAudio() {
        if (!localStream) return false;
        const track = localStream.getAudioTracks()[0];
        if (track) {
            track.enabled = !track.enabled;
            return track.enabled;
        }
        return false;
    }

    function toggleVideo() {
        if (!localStream) return false;
        const track = localStream.getVideoTracks()[0];
        if (track) {
            track.enabled = !track.enabled;
            return track.enabled;
        }
        return false;
    }

    // ---- Signaling ----

    function send(msg) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(msg));
        }
    }

    function handleSignalingMessage(msg) {
        switch (msg.type) {
            case "peer-joined":
                handlePeerJoined(msg.senderId);
                break;
            case "peer-left":
                removePeer(msg.senderId);
                break;
            case "offer":
                handleOffer(msg);
                break;
            case "answer":
                handleAnswer(msg);
                break;
            case "ice-candidate":
                handleICECandidate(msg);
                break;
            default:
                console.warn("[webrtc] unknown message type:", msg.type);
        }
    }

    // ---- Peer Connection Management ----

    function createPeerConnection(peerId) {
        const pc = new RTCPeerConnection({ iceServers: iceServers });

        // Add local tracks
        if (localStream) {
            localStream.getTracks().forEach(function (track) {
                pc.addTrack(track, localStream);
            });
        }

        // Handle incoming tracks
        pc.ontrack = function (event) {
            let peerData = peers[peerId];
            if (!peerData) return;

            if (!peerData.stream) {
                peerData.stream = new MediaStream();
            }
            peerData.stream.addTrack(event.track);

            // Fire UI callback
            if (window.BroRTC.onRemoteStream) {
                window.BroRTC.onRemoteStream(peerId, peerData.stream);
            }
        };

        // Send ICE candidates
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
            console.log("[webrtc] peer " + peerId + " connection state:", pc.connectionState);
            if (pc.connectionState === "disconnected" || pc.connectionState === "failed") {
                removePeer(peerId);
            }
        };

        peers[peerId] = { pc: pc, stream: null };
        return pc;
    }

    // When a new peer joins, we are the polite side – create offer
    async function handlePeerJoined(peerId) {
        if (peers[peerId]) return; // already connected

        const pc = createPeerConnection(peerId);

        try {
            const offer = await pc.createOffer();
            await pc.setLocalDescription(offer);
            send({
                type: "offer",
                roomId: currentRoom,
                targetId: peerId,
                payload: pc.localDescription,
            });
        } catch (err) {
            console.error("[webrtc] createOffer error:", err);
        }

        if (window.BroRTC.onPeerCountChange) {
            window.BroRTC.onPeerCountChange(Object.keys(peers).length);
        }
    }

    async function handleOffer(msg) {
        let pc;
        if (peers[msg.senderId]) {
            pc = peers[msg.senderId].pc;
        } else {
            pc = createPeerConnection(msg.senderId);
        }

        try {
            await pc.setRemoteDescription(new RTCSessionDescription(msg.payload));
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            send({
                type: "answer",
                roomId: currentRoom,
                targetId: msg.senderId,
                payload: pc.localDescription,
            });
        } catch (err) {
            console.error("[webrtc] handleOffer error:", err);
        }

        if (window.BroRTC.onPeerCountChange) {
            window.BroRTC.onPeerCountChange(Object.keys(peers).length);
        }
    }

    async function handleAnswer(msg) {
        const peer = peers[msg.senderId];
        if (!peer) return;
        try {
            await peer.pc.setRemoteDescription(new RTCSessionDescription(msg.payload));
        } catch (err) {
            console.error("[webrtc] handleAnswer error:", err);
        }
    }

    async function handleICECandidate(msg) {
        const peer = peers[msg.senderId];
        if (!peer) return;
        try {
            await peer.pc.addIceCandidate(new RTCIceCandidate(msg.payload));
        } catch (err) {
            console.error("[webrtc] addIceCandidate error:", err);
        }
    }

    function removePeer(peerId) {
        const peer = peers[peerId];
        if (peer) {
            peer.pc.close();
            delete peers[peerId];
        }

        if (window.BroRTC.onPeerRemoved) {
            window.BroRTC.onPeerRemoved(peerId);
        }
        if (window.BroRTC.onPeerCountChange) {
            window.BroRTC.onPeerCountChange(Object.keys(peers).length);
        }
    }
})();
