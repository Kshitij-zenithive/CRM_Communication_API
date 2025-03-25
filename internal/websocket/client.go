// internal/websocket/client.go (No changes needed)
package websocket

import (
	"crm-communication-api/internal/middleware"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow connections from any origin
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan Message

	//The user Id
	ID string

	//The room Id
	RoomID string
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Handle incoming messages (optional)
		log.Printf("Received message: %s", message)
		// You might want to unmarshal the message and process it based on its type.
		// Example:
		// var incomingMsg Message
		// if err := json.Unmarshal(message, &incomingMsg); err == nil {
		//   // Process based on incomingMsg.Type
		// }
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Marshal the Message struct into JSON.
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshaling message: %v", err)
				return
			}
			w.Write(jsonMessage)

			// Add queued chat messages to the current websocket message. (Not used for now)
			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				jsonMsg, err := json.Marshal(msg) // Marshal each message
				if err != nil {
					log.Printf("Error marshaling message: %v", err)
					continue // Skip this message if marshaling fails
				}
				w.Write(jsonMsg)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {

	// Get the room ID from the query parameter (e.g., client ID or conversation ID)
	roomID := r.URL.Query().Get("roomId") // Now using roomId consistently
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Get User ID from context
	userID, err := middleware.GetUserIDFromContext(r.Context())
	var id string
	if err != nil {
		id = uuid.NewString() // Generate a unique ID for the client
	} else {
		id = userID.String()
	}
	client := &Client{
		ID:     id, // Assign the user ID or generated ID.
		hub:    hub,
		conn:   conn,
		send:   make(chan Message, 256),
		RoomID: roomID, // Use the provided room ID
	}

	client.hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
