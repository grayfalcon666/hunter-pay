package wshub

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/grayfalcon666/escrow-bounty/db"
	"github.com/grayfalcon666/escrow-bounty/models"
)

// Hub manages all WebSocket clients and message routing.
type Hub struct {
	store db.Store // database store for persisting messages

	// conversations: convID -> set of clients
	conversations map[int64]map[*Client]bool
	// users: username -> set of clients (for direct message routing)
	users map[string]map[*Client]bool
	// channels
	register   chan *Client
	unregister chan *Client
	broadcast  chan *OutboundMessage

	mu sync.RWMutex
}

type OutboundMessage struct {
	ConversationID int64
	Payload        []byte
	ExcludeClient  *Client
}

// NewHub creates a new Hub instance.
func NewHub(store db.Store) *Hub {
	return &Hub{
		store:        store,
		conversations: make(map[int64]map[*Client]bool),
		users:         make(map[string]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan *OutboundMessage, 256),
	}
}

// Run starts the hub's main loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// Register client under their username
			if _, ok := h.users[client.username]; !ok {
				h.users[client.username] = make(map[*Client]bool)
			}
			h.users[client.username][client] = true
			h.mu.Unlock()
			log.Printf("WS Hub: REGISTER user=%s, totalClientsForUser=%d", client.username, len(h.users[client.username]))

		case client := <-h.unregister:
			h.mu.Lock()
			// Remove from all conversation groups
			for convID := range client.groups {
				if clients, ok := h.conversations[convID]; ok {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.conversations, convID)
					}
				}
			}
			// Remove from users map
			if clients, ok := h.users[client.username]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.users, client.username)
				}
			}
			h.mu.Unlock()
			log.Printf("WS Hub: UNREGISTER user=%s", client.username)

		case msg := <-h.broadcast:
			log.Printf("[DEBUG] broadcast case: convID=%d, clients in conv=%v", msg.ConversationID, h.conversations[msg.ConversationID] != nil)
			h.mu.RLock()
			if clients, ok := h.conversations[msg.ConversationID]; ok {
				log.Printf("[DEBUG] broadcast: found %d clients in conv %d", len(clients), msg.ConversationID)
				for client := range clients {
					if client != msg.ExcludeClient {
						select {
						case client.send <- msg.Payload:
						default:
							// client send buffer full, skip
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// JoinConversation adds a client to a conversation room.
func (h *Hub) JoinConversation(client *Client, convID int64) {
	log.Printf("[DEBUG] JoinConversation called: user=%s, convID=%d", client.username, convID)
	h.mu.Lock()
	defer h.mu.Unlock()
	log.Printf("[DEBUG] JoinConversation: acquired lock, user=%s, convID=%d, existing clients=%v", client.username, convID, h.conversations[convID])
	if _, ok := h.conversations[convID]; !ok {
		h.conversations[convID] = make(map[*Client]bool)
	}
	h.conversations[convID][client] = true
	client.groups[convID] = true
	log.Printf("[DEBUG] JoinConversation: SUCCESS, user=%s now in conv=%d, total clients in conv=%d", client.username, convID, len(h.conversations[convID]))
}

// LeaveConversation removes a client from a conversation room.
func (h *Hub) LeaveConversation(client *Client, convID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.conversations[convID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.conversations, convID)
		}
	}
	delete(client.groups, convID)
}

// SaveMessage persists a private message to the database.
func (h *Hub) SaveMessage(convID int64, content, sender string) (*models.AllMessage, error) {
	log.Printf("WS Hub: SaveMessage called - convID=%d, sender=%s, content=%q", convID, sender, content)
	convIDPtr := convID
	msg, err := h.store.CreateMessageV2(context.Background(), models.MessageTypePrivate, &convIDPtr, nil, sender, content)
	if err != nil {
		log.Printf("WS Hub: SaveMessage FAILED - err=%v", err)
		return nil, err
	}
	log.Printf("WS Hub: SaveMessage SUCCESS - msgID=%d", msg.ID)
	return msg, nil
}

// BroadcastToConversation sends a message to all clients in a conversation.
func (h *Hub) BroadcastToConversation(convID int64, payload []byte, exclude *Client) {
	log.Printf("[DEBUG] BroadcastToConversation called: convID=%d, exclude=%v", convID, exclude != nil)
	h.broadcast <- &OutboundMessage{
		ConversationID: convID,
		Payload:        payload,
		ExcludeClient:  exclude,
	}
}

// BroadcastToUser sends a message to all clients of a specific user.
func (h *Hub) BroadcastToUser(username string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.users[username]; ok {
		for client := range clients {
			select {
			case client.send <- payload:
			default:
			}
		}
	}
}

// SendPrivateMessage sends a message to a specific conversation.
func (h *Hub) SendPrivateMessage(convID int64, msg interface{}) {
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WS Hub: failed to marshal message: %v", err)
		return
	}
	h.BroadcastToConversation(convID, payload, nil)
}
