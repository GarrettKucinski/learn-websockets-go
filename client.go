package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// always have a ping intervale that is lower than the pong wait
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 9) / 10
)

type Clients map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    m,
		egress:     make(chan Event),
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	// reset the pong wait timer so the connection doesn't close
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}

func (c *Client) readMessages() {
	defer func() {
		// The break in the for loop with trigger on error
		// the go routine will end triggering this function to cleanup connection
		c.manager.removeClient(c)
	}()

	// start the timeer and wait for a pong message
	// will return if pong is not received in time
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		// close the connection if the read deadline is exceeded
		// assumes the connection has closed on the client side
		// because we didn't receive a pong from the client in time
		return
	}

	// you should know how long your messages can be
	// prevent jumbo frames (really large messages) by setting a limit size in bytes
	// for how big a message can be
	c.connection.SetReadLimit(512)

	// calls the handler to reset the timer waiting for a pong message
	c.connection.SetPongHandler(c.pongHandler)

	for {
		// ReadMessage is a blocking call that stops the loop until a message is received
		_, payload, err := c.connection.ReadMessage()
		if err != nil {
			// if the connection was closed without the client or server sending a close message
			// we want to log it, this means something has gone wrong
			closeConnectError := websocket.IsUnexpectedCloseError(
				// These are reasonable errors to expect for clients disconnecting
				// or abonormal connection loss, so it makes sense to filter them out
				err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
			)
			if closeConnectError {
				log.Printf("error reading message %v", err)
			}
			break
		}

		var request Event

		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		// we have an event
		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println("error handling message:", err)
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		// triggered by any of the returns in the for/select
		// basically cleanup the client connection on error
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				// We send a close message to the other side of this connection in the case of error
				// we do this to notify the other side that something is wrong and
				// this connection needs to be closed
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				// We are removing the client because they sent one bad message which
				// seems a little severe, but hey, it's just a demo
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			log.Println("message sent")

		case <-ticker.C:
			log.Println("Ping")
			// Send a Ping to the Client
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println("writemsg error", err)
				return
			}
		}
	}
}
