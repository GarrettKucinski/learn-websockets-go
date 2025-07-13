let selectedChat = "general";
let conn = null;

class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
}

function routeEvent(e) {
    e.preventDefault();
    if (!e.type) console.error("no type in event");

    const events = new Map([
        ["new message", () => console.log("new message")]
        ["default", (e) => console.error("unsupported event", e.type)]
    ]);

    const handler = events.has(e)
        ? events.get(e)
        : events.get("default")

    handler(e);
}

function sendEvent(eventName, payload) {
    const event = new Event(eventName, payload);

    conn.send(JSON.stringify(event))
}

function changeChatroom() {
    const newChat = document.getElementById("chatroom");
    if (newChat && newChat.value != selectedChat) {
        console.log(newChat);
    }

    return false;
}

function sendMessage(e) {
    e.preventDefault();

    const newMessage = document.getElementById("message");
    if (newMessage) {
        sendEvent("send message", newMessage.value);
    }
}

function connectWebsocket(otp) {
    if (!window.webSocket) {
        console.error("browser does not support websockts!");
        return;
    }

    console.log("websockets!");
    conn = new WebSocket(`ws://${document.location.host}/ws?otp=${otp}`);
    conn.onopen = function (evt) {
        document.getElementById("connection-header").innerHTML = "Websocket Connected!";
    }
    conn.onclose = function () { 
        document.getElementById("connection-header").innerHTML = "Websocket Disconnected!";
    }
    conn.onmessage = function (evt) {
        const eventData = JSON.parse(evt.data);
        const event = Object.assign(new Event, eventData);

        routeEvent(event);
    };
}

function login() {
    const username = document.getElementById("username").value;
    const password = document.getElementById("password").value;

    fetch("/login", {
        method: "post",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
        mode: "cors",
    }).then(res => {
        if (res.ok) return res.json();
        throw "unauthorized";
    }).then(data => {
        connectWebsocket(data.otp);
    }).catch(error => {
        console.error(error);
        return false;
    });
}

/**
 *  Websockets has a few events to be aware of
 * onclose: which is fired whenever the socket closes
 * onerror: fires whenever there is an error
 * onopen: fires whenver the connection is opened 
 * onmessage: fires whenever there is a message
 */

function init () {
    document.getElementById("chatroom-selection")
        .addEventListener("submit", changeChatroom);
    document.getElementById("chatroom-message")
        .addEventListener("submit", sendMessage);
    document.getElementById("login-form")
        .addEventListener("submit", login);
};

init();