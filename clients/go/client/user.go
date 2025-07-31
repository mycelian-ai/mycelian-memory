package client

import "time"

// User represents a Synapse user.
type User struct {
	ID          string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName,omitempty"`
	TimeZone    string    `json:"timeZone,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateUserRequest sent to Memory service.
type CreateUserRequest struct {
	UserID      string `json:"userId"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName,omitempty"`
	TimeZone    string `json:"timeZone,omitempty"`
}
