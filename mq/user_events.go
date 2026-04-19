package mq

import "time"

// UserCreatedEvent 用户创建事件
type UserCreatedEvent struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	RequestId string    `json:"request_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ProfileCreatedEvent 用户资料创建事件
type ProfileCreatedEvent struct {
	Username     string `json:"username"`
	RequestId    string `json:"request_id"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// AccountCreatedEvent 账户创建事件
type AccountCreatedEvent struct {
	Username     string `json:"username"`
	AccountId    int64  `json:"account_id"`
	RequestId    string `json:"request_id"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// UserInitializationCompletedEvent 用户初始化完成事件
type UserInitializationCompletedEvent struct {
	Username  string `json:"username"`
	RequestId string `json:"request_id"`
	Status    string `json:"status"` // INITIALIZED, FAILED, PARTIALLY_INITIALIZED
}
