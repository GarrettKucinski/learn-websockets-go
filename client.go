package main

import (
	"log"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]bool

type Client struct {
	connection *websocket.Conn
	manager    *Manager

	// egress is used to avoid concurrent writes on the websocket connection
	egress chan []byte
}

func NewClient(conn *websocket.Conn, m *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    m,
		egress:     make(chan []byte),
	}
}

func (c *Client) readMessages() {
	defer func() {
		// The break in the for loop with trigger on error
		// the go routine will end triggering this function to cleanup connection
		c.manager.removeClient(c)
	}()
	for {
		messageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			// if the connection was closed without the client or server sending a close message
			// we want to log it, this means something has gone wrong
			closeConnectError := websocket.IsUnexpectedCloseError(
				err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure,
			)
			if closeConnectError {
				log.Printf("error reading message %v", err)
			}
			break
		}

		for wsclient := range c.manager.clients {
			wsclient.egress <- payload
		}

		log.Println(messageType)
		log.Println(string(payload))
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

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

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("failed to send message: %v", err)
			}
			log.Println("message sent")
		}
	}
}
