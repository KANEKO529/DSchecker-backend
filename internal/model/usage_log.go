// internal/model/usage_log.go
package model

import "time"

type UsageLog struct {
	ID              int64     `json:"id"`
	UserID          *int64    `json:"user_id,omitempty"`
	GuestIdentifier *string   `json:"guest_identifier,omitempty"`
	ActionType      string    `json:"action_type"`
	UsedAt          time.Time `json:"used_at"`
	CreatedAt       time.Time `json:"created_at"`
}
