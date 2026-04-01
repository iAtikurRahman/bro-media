// app.js – Dashboard UI logic for Bro Media

(function () {
    "use strict";

    // ---- Auth gate ----
    if (!window.BroAuth || !window.BroAuth.isLoggedIn()) {
        window.location.href = "/login";
        return;
    }

    var username = window.BroAuth.getUsername();
    var inCall = false;
    var currentRoom = null;
    var audioEnabled = true;
    var videoEnabled = true;

    // ---- DOM refs ----
    var userDisplay       = document.getElementById("user-display");
    var logoutBtn         = document.getElementById("logout-btn");
    var roomInput         = document.getElementById("room-input");
    var joinRoomBtn       = document.getElementById("join-room-btn");
    var onlineUsersList   = document.getElementById("online-users-list");
    var onlineCount       = document.getElementById("online-count");
    var welcomeView       = document.getElementById("welcome-view");
    var callView          = document.getElementById("call-view");
    var videoGrid         = document.getElementById("video-grid");
    var localVideo        = document.getElementById("local-video");
    var callRoomLabel     = document.getElementById("call-room-label");
    var callPeerCount     = document.getElementById("call-peer-count");
    var toggleAudioBtn    = document.getElementById("toggle-audio");
    var toggleVideoBtn    = document.getElementById("toggle-video");
    var shareScreenBtn    = document.getElementById("share-screen");
    var toggleChatBtn     = document.getElementById("toggle-chat");
    var hangUpBtn         = document.getElementById("hang-up");
    var chatPanel         = document.getElementById("chat-panel");
    var chatInput         = document.getElementById("chat-input");
    var sendChatBtn       = document.getElementById("send-chat");
    var closeChatBtn      = document.getElementById("close-chat");
    var incomingModal     = document.getElementById("incoming-call-modal");
    var callerNameEl      = document.getElementById("caller-name");
    var acceptCallBtn     = document.getElementById("accept-call");
    var rejectCallBtn     = document.getElementById("reject-call");

    // Show logged-in user
    userDisplay.textContent = username;

    // Init WebRTC + WebSocket
    initApp();

    async function initApp() {
        try {
            var result = await window.BroRTC.init();
            localVideo.srcObject = result.localStream;
        } catch (err) {
            console.error("[app] init error:", err);
            alert("Could not access camera/microphone. Please allow permissions.");
        }
    }

    // ---- Logout ----
    logoutBtn.addEventListener("click", function () {
        window.BroRTC.leaveRoom();
        window.BroAuth.clearAuth();
        window.location.href = "/login";
    });

    // ---- Join Room ----
    joinRoomBtn.addEventListener("click", function () {
        var room = roomInput.value.trim();
        if (!room) return;
        startCall(room);
    });

    roomInput.addEventListener("keydown", function (e) {
        if (e.key === "Enter") joinRoomBtn.click();
    });

    function startCall(roomId) {
        if (inCall) {
            window.BroRTC.leaveRoom();
            cleanupCall();
        }
        currentRoom = roomId;
        inCall = true;
        window.BroRTC.joinRoom(roomId);
        welcomeView.classList.remove("active");
        welcomeView.classList.add("hidden");
        callView.classList.remove("hidden");
        callView.classList.add("active");
        callRoomLabel.textContent = "Room: " + roomId;
        window.BroChat.clear();
    }

    function cleanupCall() {
        inCall = false;
        currentRoom = null;
        document.querySelectorAll(".remote-video-container").forEach(function (el) { el.remove(); });
        callView.classList.remove("active");
        callView.classList.add("hidden");
        welcomeView.classList.remove("hidden");
        welcomeView.classList.add("active");
        chatPanel.classList.add("hidden");
        callPeerCount.textContent = "0 peers";
        audioEnabled = true;
        videoEnabled = true;
        toggleAudioBtn.textContent = "🎤";
        toggleAudioBtn.classList.remove("muted");
        toggleVideoBtn.textContent = "📹";
        toggleVideoBtn.classList.remove("muted");
        shareScreenBtn.classList.remove("active-share");
    }

    // ---- Call Controls ----
    toggleAudioBtn.addEventListener("click", function () {
        audioEnabled = window.BroRTC.toggleAudio();
        toggleAudioBtn.textContent = audioEnabled ? "🎤" : "🔇";
        toggleAudioBtn.classList.toggle("muted", !audioEnabled);
    });

    toggleVideoBtn.addEventListener("click", function () {
        videoEnabled = window.BroRTC.toggleVideo();
        toggleVideoBtn.textContent = videoEnabled ? "📹" : "🚫";
        toggleVideoBtn.classList.toggle("muted", !videoEnabled);
    });

    shareScreenBtn.addEventListener("click", async function () {
        var sharing = await window.BroRTC.toggleScreenShare();
        shareScreenBtn.classList.toggle("active-share", sharing);
    });

    window.BroRTC.onScreenShareStopped = function () {
        shareScreenBtn.classList.remove("active-share");
    };

    toggleChatBtn.addEventListener("click", function () {
        chatPanel.classList.toggle("hidden");
    });

    closeChatBtn.addEventListener("click", function () {
        chatPanel.classList.add("hidden");
    });

    hangUpBtn.addEventListener("click", function () {
        window.BroRTC.leaveRoom();
        cleanupCall();
    });

    // ---- Chat ----
    sendChatBtn.addEventListener("click", sendChatMessage);
    chatInput.addEventListener("keydown", function (e) {
        if (e.key === "Enter") sendChatMessage();
    });

    function sendChatMessage() {
        var text = chatInput.value.trim();
        if (!text) return;
        window.BroRTC.sendChat(text);
        chatInput.value = "";
    }

    // ---- WebRTC Callbacks ----
    window.BroRTC.onRemoteStream = function (peerId, stream) {
        var container = document.getElementById("peer-" + peerId);
        if (container) {
            container.querySelector("video").srcObject = stream;
            return;
        }
        container = document.createElement("div");
        container.className = "video-container remote-video-container";
        container.id = "peer-" + peerId;

        var video = document.createElement("video");
        video.autoplay = true;
        video.playsinline = true;
        video.srcObject = stream;

        var label = document.createElement("span");
        label.className = "video-label";
        label.textContent = peerId;

        container.appendChild(video);
        container.appendChild(label);
        videoGrid.appendChild(container);
    };

    window.BroRTC.onPeerRemoved = function (peerId) {
        var el = document.getElementById("peer-" + peerId);
        if (el) el.remove();
    };

    window.BroRTC.onPeerCountChange = function (count) {
        callPeerCount.textContent = count + (count === 1 ? " peer" : " peers");
    };

    // ---- Online Users ----
    window.BroRTC.onOnlineUsers = function (users) {
        onlineUsersList.innerHTML = "";
        var count = 0;
        users.forEach(function (u) {
            if (u === username) return;
            count++;
            var li = document.createElement("li");
            li.className = "user-item";

            var nameSpan = document.createElement("span");
            nameSpan.className = "user-item-name";
            nameSpan.textContent = u;

            var callBtn = document.createElement("button");
            callBtn.className = "btn btn-sm btn-primary user-call-btn";
            callBtn.textContent = "📞";
            callBtn.title = "Call " + u;
            callBtn.addEventListener("click", function () {
                window.BroRTC.callUser(u);
                var roomId = privateRoomId(username, u);
                startCall(roomId);
            });

            li.appendChild(nameSpan);
            li.appendChild(callBtn);
            onlineUsersList.appendChild(li);
        });
        onlineCount.textContent = count;
    };

    // ---- Incoming Call ----
    window.BroRTC.onCallIncoming = function (caller, roomId) {
        callerNameEl.textContent = caller + " is calling you…";
        incomingModal.classList.remove("hidden");

        acceptCallBtn.onclick = function () {
            incomingModal.classList.add("hidden");
            window.BroRTC.acceptCall(caller, roomId);
            startCall(roomId);
        };
        rejectCallBtn.onclick = function () {
            incomingModal.classList.add("hidden");
            window.BroRTC.rejectCall(caller);
        };
    };

    window.BroRTC.onCallAccepted = function () {
        // Caller is already in the room – peer will join automatically
    };

    window.BroRTC.onCallRejected = function (responder) {
        alert(responder + " rejected your call.");
        if (inCall) { window.BroRTC.leaveRoom(); cleanupCall(); }
    };

    // ---- Chat Messages ----
    window.BroRTC.onChatMessage = function (msg) {
        var isSelf = msg.senderId === username;
        window.BroChat.addMessage(msg.senderId, msg.text, msg.timestamp, isSelf);
        if (chatPanel.classList.contains("hidden")) chatPanel.classList.remove("hidden");
    };

    // ---- Helper ----
    function privateRoomId(a, b) {
        return a < b ? "dm-" + a + "-" + b : "dm-" + b + "-" + a;
    }
})();
