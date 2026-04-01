// app.js – UI logic for Bro Media

(function () {
    "use strict";

    // DOM elements
    const joinScreen = document.getElementById("join-screen");
    const callScreen = document.getElementById("call-screen");
    const roomInput = document.getElementById("room-input");
    const joinBtn = document.getElementById("join-btn");
    const joinError = document.getElementById("join-error");
    const localVideo = document.getElementById("local-video");
    const videoGrid = document.getElementById("video-grid");
    const toggleAudioBtn = document.getElementById("toggle-audio");
    const toggleVideoBtn = document.getElementById("toggle-video");
    const hangUpBtn = document.getElementById("hang-up");
    const roomLabel = document.getElementById("room-label");
    const peerCount = document.getElementById("peer-count");

    let audioEnabled = true;
    let videoEnabled = true;

    // ---- Join Room ----

    joinBtn.addEventListener("click", async function () {
        const room = roomInput.value.trim();
        if (!room) {
            showError("Please enter a room ID");
            return;
        }

        joinBtn.disabled = true;
        joinBtn.textContent = "Connecting…";
        hideError();

        try {
            const { localStream } = await window.BroRTC.init();
            localVideo.srcObject = localStream;

            window.BroRTC.joinRoom(room);

            // Switch screens
            joinScreen.classList.remove("active");
            joinScreen.classList.add("hidden");
            callScreen.classList.remove("hidden");
            callScreen.classList.add("active");
            roomLabel.textContent = "Room: " + room;
        } catch (err) {
            console.error("[app] init error:", err);
            showError("Could not access camera/microphone. Please allow permissions.");
            joinBtn.disabled = false;
            joinBtn.textContent = "Join Room";
        }
    });

    // Allow Enter key to join
    roomInput.addEventListener("keydown", function (e) {
        if (e.key === "Enter") joinBtn.click();
    });

    // ---- Controls ----

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

    hangUpBtn.addEventListener("click", function () {
        window.BroRTC.leaveRoom();

        // Remove all remote videos
        document.querySelectorAll(".remote-video-container").forEach(function (el) {
            el.remove();
        });

        // Switch back to join screen
        callScreen.classList.remove("active");
        callScreen.classList.add("hidden");
        joinScreen.classList.remove("hidden");
        joinScreen.classList.add("active");
        joinBtn.disabled = false;
        joinBtn.textContent = "Join Room";
        peerCount.textContent = "0 peers";
    });

    // ---- WebRTC Callbacks ----

    window.BroRTC.onRemoteStream = function (peerId, stream) {
        let container = document.getElementById("peer-" + peerId);
        if (container) {
            // Update existing video
            container.querySelector("video").srcObject = stream;
            return;
        }

        // Create new video element
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
        peerCount.textContent = count + (count === 1 ? " peer" : " peers");
    };

    // ---- Helpers ----

    function showError(msg) {
        joinError.textContent = msg;
        joinError.classList.remove("hidden");
    }

    function hideError() {
        joinError.textContent = "";
        joinError.classList.add("hidden");
    }
})();
