package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients Clients
	sync.RWMutex

	otps RetentionMap

	handlers map[string]EventHandler
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients:  make(Clients),
		handlers: make(map[string]EventHandler),
		otps:     NewRetentionMap(ctx, 5*time.Second),
	}

	m.setupEventHandlers()

	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
}

func SendMessage(event Event, c *Client) error {
	var chatEvent SendMessageEvent

	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}

	broadcastMessage := NewMessageEvent{
		Sent: time.Now(),
		SendMessageEvent: SendMessageEvent{
			Message: chatEvent.Message,
			From:    chatEvent.From,
		},
	}

	data, err := json.Marshal(broadcastMessage)
	if err != nil {
		return fmt.Errorf("failed to marshall broadcast: %v", err)
	}

	outgoingEvent := Event{
		Payload: data,
		Type:    EventNewMessage,
	}

	for client := range c.manager.clients {
		client.egress <- outgoingEvent
	}

	return nil
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("no such event type")
	}
}

func (m *Manager) serveWs(w http.ResponseWriter, r *http.Request) {
	otp := r.URL.Query().Get("otp")

	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !m.otps.VerifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// if they get here they have a valid otp
	// we allow them to connect via websockets
	log.Println("New connection")
	// upgrade regular http connection to websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m)

	m.addClient(client)

	// Start two go routines per client
	// - one to read messages
	//  - one to write messages
	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var req userLoginRequest

	// read the request body from the http stream
	// more efficient that json.Marshall which loads the json into
	// memory first
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println("username or password is incorrect")
		return
	}

	// if our super secure auth passes
	if req.Password == "password" && req.Username == "gkucinski" {
		type response struct {
			OTP string `json:"otp"`
		}

		// issue a new otp to the user
		otp := m.otps.NewOTP()

		resp := response{
			OTP: otp.Key,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	// If credentials are wrong:
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "Invalid credentials"}`))
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// should make the origin configurable through
	// env vars in real application
	switch origin {
	case "https://localhost:8080":
		return true
	default:
		return false
	}
}
