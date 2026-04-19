package models

import "time"

type Chat struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	BountyID         int64     `gorm:"uniqueIndex;not null" json:"bounty_id"`
	EmployerUsername  string    `gorm:"type:varchar(255);not null" json:"employer_username"`
	HunterUsername   string    `gorm:"type:varchar(255);not null" json:"hunter_username"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ChatMessage struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatID         int64     `gorm:"not null;index" json:"chat_id"`
	SenderUsername string    `gorm:"type:varchar(255);not null" json:"sender_username"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// MessageType constants
const (
	MessageTypeBounty   = "bounty"
	MessageTypePrivate  = "private"
)

// PrivateConversation is a user-to-user chat session.
type PrivateConversation struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	User1Username   string    `gorm:"type:varchar(255);not null;index:idx_private_conv_user1" json:"user1_username"`
	User2Username   string    `gorm:"type:varchar(255);not null;index:idx_private_conv_user2" json:"user2_username"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// AllMessage is the unified message model for both bounty-scoped and private chats.
type AllMessage struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageType    string    `gorm:"type:varchar(20);not null" json:"message_type"` // "bounty" | "private"
	ConversationID *int64    `gorm:"index:idx_all_messages_conversation" json:"conversation_id,omitempty"`
	BountyChatID   *int64    `gorm:"index:idx_all_messages_bounty_chat" json:"bounty_chat_id,omitempty"`
	SenderUsername string    `gorm:"type:varchar(255);not null;index:idx_all_messages_sender" json:"sender_username"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	IsRead         bool      `gorm:"not null;default:false" json:"is_read"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// ConversationUnreadCount tracks per-user unread counts for private conversations.
type ConversationUnreadCount struct {
	Username       string `gorm:"primaryKey;not null" json:"username"`
	ConversationID int64  `gorm:"primaryKey;not null;autoincrement:false" json:"conversation_id"`
	UnreadCount    int    `gorm:"not null;default:0" json:"unread_count"`
}

// Comment is a threaded comment on a bounty page.
type Comment struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	BountyID       int64      `gorm:"not null;index:idx_comments_bounty" json:"bounty_id"`
	ParentID       *int64     `gorm:"index:idx_comments_parent" json:"parent_id,omitempty"`
	AuthorUsername string     `gorm:"type:varchar(255);not null" json:"author_username"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	Replies        []Comment  `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}
