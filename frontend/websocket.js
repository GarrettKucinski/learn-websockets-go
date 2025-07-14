let selectedChat = "general";
let conn = null;

class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
}

class SendMessageEvent {
    constructor(message, from) {
        this.message = message;
        this.from = from;
    }
}

class NewMessageEvent extends SendMessageEvent{
    constructor(message, from, sent) {
        super(message, from);
        this.sent = sent;
    }
}

function routeEvent(wsEvent) {
    if (!wsEvent.type) console.error("no type in event");

    const events = new Map([
        ["new message", () => {
            const messageEvent = Object.assign(new NewMessageEvent, wsEvent.payload);
            appendChatMessage(messageEvent);
        }],
        ["default", () => console.error("unsupported event", wsEvent.type)]
    ]);

    const handler = events.get(wsEvent.type) ?? events.get("default");

    handler(wsEvent);
}

function appendChatMessage (messageEvent) {
    const { sent, from, message } = messageEvent ?? {};

    const sentTime = new Date(sent ?? Date.now());
    const formattedMessage = `${sentTime.toLocaleString()} ${from}: ${message}`
    const messageWindow = document.getElementById("chat-messages");
    console.log(messageWindow, messageEvent.payload);

    messageWindow.innerHTML = messageWindow.innerHTML + "\n" + formattedMessage;
    messageWindow.scrollTop = messageWindow.scrollHeight;
}

function sendEvent(eventName, payload) {
    const event = new Event(eventName, payload);

    console.log("connection", conn)
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
        const outgoingMessage = new SendMessageEvent(newMessage.value, "garrett");
        sendEvent("send message", outgoingMessage);
    }
}

function connectWebsocket(otp) {
    if (!window.WebSocket) {
        console.error("browser does not support websockts!");
        return;
    }

    conn = new WebSocket(`wss://${document.location.host}/ws?otp=${otp}`);
    conn.onopen = function () {
        document.getElementById("connection-header").innerHTML = "Websocket Connected!";
    };
    conn.onclose = function () { 
        document.getElementById("connection-header").innerHTML = "Websocket Disconnected!";
    };
    conn.onerror = function (evt) {
        console.error("WebSocket error", evt);
    };
    conn.onmessage = function (evt) {
        const eventData = JSON.parse(evt.data);
        const event = Object.assign(new Event, eventData);

        routeEvent(event);
    };

    console.log(conn);
}

async function login(e) {
    e.preventDefault();

    const username = document.getElementById("username").value;
    const password = document.getElementById("password").value;

    try {
        const res = await fetch("/login", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ username, password }),
            mode: "cors",
        });

        const response = await res.json();

        if (!res.ok) throw "unauthorized";

        connectWebsocket(response.otp);
    } catch(error) {
        console.error(error);
        return false;
    }
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