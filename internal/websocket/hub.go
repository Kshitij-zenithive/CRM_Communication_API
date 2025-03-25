// internal/websocket/hub.go (No changes needed)

package websocket

import (
	"log"
)

// Message represents a WebSocket message
type Message struct {
	Type    string `json:"type"`
	RoomID  string `json:"roomId"` // Now consistently using RoomID
	Payload any    `json:"payload"`
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Rooms      map[string]map[*Client]bool // RoomID -> Set of Clients
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Rooms:      make(map[string]map[*Client]bool),
	}
}

// Run starts the WebSocket hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			// Add client to the room
			if client.RoomID != "" {
				if h.Rooms[client.RoomID] == nil {
					h.Rooms[client.RoomID] = make(map[*Client]bool)
				}
				h.Rooms[client.RoomID][client] = true
			}
			log.Printf("Client %s registered to room %s", client.ID, client.RoomID)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				// Remove client from the room
				if h.Rooms[client.RoomID] != nil {
					delete(h.Rooms[client.RoomID], client)
					if len(h.Rooms[client.RoomID]) == 0 { // If room is empty, delete it.
						delete(h.Rooms, client.RoomID)
					}
				}
				close(client.send) // Close send channel
				log.Printf("Client %s unregistered from room %s", client.ID, client.RoomID)
			}

		case message := <-h.Broadcast:
			// Broadcast to clients in the specified room
			if room, ok := h.Rooms[message.RoomID]; ok {
				for client := range room {
					select {
					case client.send <- message:
					default:
						// If the client's send buffer is full, assume they are dead or stuck
						close(client.send)
						delete(h.Clients, client)
						// Remove client from the room
						if h.Rooms[client.RoomID] != nil {
							delete(h.Rooms[client.RoomID], client)
							if len(h.Rooms[client.RoomID]) == 0 {
								delete(h.Rooms, client.RoomID)
							}
						}
					}
				}
			} else {
				log.Printf("Room %s not found", message.RoomID) // Debugging
			}
		}
	}
}
