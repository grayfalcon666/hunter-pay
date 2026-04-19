package wshub

import (
	"encoding/json"
	"strconv"
)

// ==========================================
// Client -> Server messages
// ==========================================

type ClientMessage struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

type JoinConvPayload struct {
	ConversationID int64 `json:"conversation_id"`
}

func (p *JoinConvPayload) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw["conversation_id"].(type) {
	case float64:
		p.ConversationID = int64(v)
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			p.ConversationID = id
		}
	}
	return nil
}

type LeaveConvPayload struct {
	ConversationID int64 `json:"conversation_id"`
}

func (p *LeaveConvPayload) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw["conversation_id"].(type) {
	case float64:
		p.ConversationID = int64(v)
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			p.ConversationID = id
		}
	}
	return nil
}

type SendMessagePayload struct {
	ConversationID int64  `json:"conversation_id"`
	Content       string `json:"content"`
}

// UnmarshalJSON supports both "content" and "Content" field names from frontend
func (p *SendMessagePayload) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// Handle conversation_id as both number (float64) and string
	switch v := raw["conversation_id"].(type) {
	case float64:
		p.ConversationID = int64(v)
	case string:
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			p.ConversationID = id
		}
	}
	// Accept both "content" and "Content"
	if c, ok := raw["content"].(string); ok {
		p.Content = c
	} else if c, ok := raw["Content"].(string); ok {
		p.Content = c
	}
	return nil
}

// ==========================================
// Server -> Client messages
// ==========================================

type ServerMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type NewMessagePayload struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversation_id"`
	SenderUsername string `json:"sender_username"`
	Content        string `json:"content"`
	IsRead         bool   `json:"is_read"`
	CreatedAt      int64  `json:"created_at"`
}

type MessageSentPayload struct {
	TempID  string           `json:"temp_id"`
	Message *NewMessagePayload `json:"message"`
}

type UnreadUpdatePayload struct {
	ConversationID int64 `json:"conversation_id"`
	UnreadCount    int32 `json:"unread_count"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// marshalMessage creates a JSON ServerMessage.
func marshalMessage(msgType string, payload interface{}) []byte {
	m := ServerMessage{Type: msgType, Payload: payload}
	data, _ := json.Marshal(m)
	return data
}
