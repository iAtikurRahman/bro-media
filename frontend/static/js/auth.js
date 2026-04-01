// auth.js – Authentication helpers for Bro Media

(function () {
    "use strict";

    window.BroAuth = {
        getToken: function () {
            return localStorage.getItem("bro_token");
        },
        getUsername: function () {
            return localStorage.getItem("bro_username");
        },
        setAuth: function (token, username) {
            localStorage.setItem("bro_token", token);
            localStorage.setItem("bro_username", username);
        },
        clearAuth: function () {
            localStorage.removeItem("bro_token");
            localStorage.removeItem("bro_username");
        },
        isLoggedIn: function () {
            return !!this.getToken();
        },
    };

    // ---- Login page logic ----
    var loginForm = document.getElementById("login-form");
    var signupForm = document.getElementById("signup-form");

    if (!loginForm || !signupForm) return; // not on login page

    // If already logged in, go to dashboard
    if (window.BroAuth.isLoggedIn()) {
        window.location.href = "/";
        return;
    }

    // Tab switching
    document.querySelectorAll(".auth-tab").forEach(function (tab) {
        tab.addEventListener("click", function () {
            document.querySelectorAll(".auth-tab").forEach(function (t) {
                t.classList.remove("active");
            });
            document.querySelectorAll(".auth-form").forEach(function (f) {
                f.classList.remove("active");
            });
            tab.classList.add("active");
            document.getElementById(tab.dataset.tab + "-form").classList.add("active");
            hideError();
        });
    });

    loginForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        var username = document.getElementById("login-username").value.trim();
        var password = document.getElementById("login-password").value;

        if (!username || !password) {
            showError("Please fill in all fields");
            return;
        }

        try {
            var res = await fetch(window.BRO_CONFIG.backendURL + "/api/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username: username, password: password }),
            });
            var data = await res.json();
            if (!res.ok) {
                showError(data.error || "Login failed");
                return;
            }
            window.BroAuth.setAuth(data.token, data.username);
            window.location.href = "/";
        } catch (err) {
            showError("Network error. Is the backend running?");
        }
    });

    signupForm.addEventListener("submit", async function (e) {
        e.preventDefault();
        var username = document.getElementById("signup-username").value.trim();
        var email = document.getElementById("signup-email").value.trim();
        var password = document.getElementById("signup-password").value;

        if (!username || !email || !password) {
            showError("Please fill in all fields");
            return;
        }

        try {
            var res = await fetch(window.BRO_CONFIG.backendURL + "/api/signup", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    username: username,
                    email: email,
                    password: password,
                }),
            });
            var data = await res.json();
            if (!res.ok) {
                showError(data.error || "Signup failed");
                return;
            }
            window.BroAuth.setAuth(data.token, data.username);
            window.location.href = "/";
        } catch (err) {
            showError("Network error. Is the backend running?");
        }
    });

    function showError(msg) {
        var el = document.getElementById("auth-error");
        if (el) {
            el.textContent = msg;
            el.classList.remove("hidden");
        }
    }

    function hideError() {
        var el = document.getElementById("auth-error");
        if (el) {
            el.textContent = "";
            el.classList.add("hidden");
        }
    }
})();
