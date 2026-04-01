// chat.js – Chat message rendering for Bro Media

(function () {
    "use strict";

    window.BroChat = {
        addMessage: addMessage,
        clear: clearMessages,
    };

    function addMessage(senderId, text, timestamp, isSelf) {
        var container = document.getElementById("chat-messages");
        if (!container) return;

        var msgEl = document.createElement("div");
        msgEl.className = "chat-msg" + (isSelf ? " chat-msg-self" : "");

        var header = document.createElement("div");
        header.className = "chat-msg-header";

        var sender = document.createElement("span");
        sender.className = "chat-msg-sender";
        sender.textContent = isSelf ? "You" : senderId;

        var timeEl = document.createElement("span");
        timeEl.className = "chat-msg-time";
        if (timestamp) {
            var d = new Date(timestamp);
            timeEl.textContent =
                d.getHours().toString().padStart(2, "0") +
                ":" +
                d.getMinutes().toString().padStart(2, "0");
        }

        header.appendChild(sender);
        header.appendChild(timeEl);

        var body = document.createElement("div");
        body.className = "chat-msg-body";
        body.textContent = text; // textContent prevents XSS

        msgEl.appendChild(header);
        msgEl.appendChild(body);
        container.appendChild(msgEl);
        container.scrollTop = container.scrollHeight;
    }

    function clearMessages() {
        var container = document.getElementById("chat-messages");
        if (container) container.innerHTML = "";
    }
})();
