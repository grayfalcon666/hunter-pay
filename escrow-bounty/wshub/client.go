package wshub

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MaxMessageSize = 512 * 1024 // 512KB
	SendBufSize    = 256
	PongWait       = 60 * time.Second
	PingInterval   = 30 * time.Second
	WriteDeadline  = 10 * time.Second
)

// Client represents a single WebSocket client connection.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	username string
	groups   map[int64]bool // conversation IDs this client has joined
}

// NewClient creates a new Client.
func NewClient(hub *Hub, conn *websocket.Conn, username string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, SendBufSize),
		username: username,
		groups:   make(map[int64]bool),
	}
}

// readPump handles incoming WebSocket messages.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			break
		}

		// Parse and handle client message
		var msg ClientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("WS: failed to parse message: %v", err)
			continue
		}
		c.handleMessage(&msg)
	}
}

// writePump handles outgoing WebSocket messages.
func (c *Client) writePump() {
	ticker := time.NewTicker(PingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(WriteDeadline))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(WriteDeadline))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming client messages.
func (c *Client) handleMessage(msg *ClientMessage) {
	log.Printf("WS [%s]: handleMessage action=%s, payloadLen=%d", c.username, msg.Action, len(msg.Payload))
	switch msg.Action {
	case "join_conv": {
		log.Printf("[DEBUG] join_conv case: user=%s, payload=%s", c.username, string(msg.Payload))
		var payload JoinConvPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("[DEBUG] join_conv unmarshal error: user=%s, err=%v, payload=%s", c.username, err, string(msg.Payload))
			c.sendError("INVALID_PAYLOAD", "Invalid join_conv payload")
			return
		}
		log.Printf("[DEBUG] join_conv unmarshal success: user=%s, convID=%d", c.username, payload.ConversationID)
		c.hub.JoinConversation(c, payload.ConversationID)
		log.Printf("WS [%s]: joined conversation %d", c.username, payload.ConversationID)
	}

	case "leave_conv":
		var payload LeaveConvPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid leave_conv payload")
			return
		}
		c.hub.LeaveConversation(c, payload.ConversationID)

	case "send_message":
		log.Printf("WS [%s]: >>> send_message received, raw payload: %s", c.username, string(msg.Payload))
		var payload SendMessagePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			log.Printf("WS [%s]: send_message unmarshal error: %v", c.username, err)
			c.sendError("INVALID_PAYLOAD", "Invalid send_message payload")
			return
		}
		log.Printf("WS [%s]: send_message parsed - convID=%d, contentLen=%d", c.username, payload.ConversationID, len(payload.Content))
		// 1. Persist message to database
		savedMsg, err := c.hub.SaveMessage(payload.ConversationID, payload.Content, c.username)
		if err != nil {
			c.sendError("INTERNAL_ERROR", "Failed to send message")
			log.Printf("WS [%s]: send_message save error: %v", c.username, err)
			return
		}
		// 2. Broadcast saved message to other participants (excluding sender)
		resp := marshalMessage("new_message", NewMessagePayload{
			ID:             savedMsg.ID,
			ConversationID: *savedMsg.ConversationID,
			SenderUsername: savedMsg.SenderUsername,
			Content:        savedMsg.Content,
			IsRead:         false,
			CreatedAt:      savedMsg.CreatedAt.Unix(),
		})
		c.hub.BroadcastToConversation(payload.ConversationID, resp, c)
		// 3. Send confirmation back to sender
		c.send <- marshalMessage("message_sent", MessageSentPayload{
			Message: &NewMessagePayload{
				ID:             savedMsg.ID,
				ConversationID: *savedMsg.ConversationID,
				SenderUsername: savedMsg.SenderUsername,
				Content:        savedMsg.Content,
				IsRead:         false,
				CreatedAt:      savedMsg.CreatedAt.Unix(),
			},
		})

	case "ping":
		c.send <- marshalMessage("pong", struct{}{})
	}
}

// sendError sends an error message to the client.
func (c *Client) sendError(code, message string) {
	c.send <- marshalMessage("error", ErrorPayload{Code: code, Message: message})
}
