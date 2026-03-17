// internal/model/user.go
package model

import "time"

type User struct {
	ID          int64      `json:"id"`
	ClerkUserID string     `json:"clerk_user_id"`
	Role        string     `json:"role"`
	UserName    *string    `json:"user_name"`
	Email       string     `json:"email"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type CreateUserRequest struct {
	ClerkUserID string  `json:"clerk_user_id" binding:"required"`
	UserName    *string `json:"user_name"`
	Email       string  `json:"email" binding:"required,email"`
}