// internal/model/subscription.go
package model

import "time"

type Subscription struct {
	ID                   int64      `db:"id"`
	UserID               int64      `db:"user_id"`
	StripeCustomerID     string     `db:"stripe_customer_id"`
	StripeSubscriptionID string     `db:"stripe_subscription_id"`
	StripePriceID        string     `db:"stripe_price_id"`
	Status               string     `db:"status"`
	CurrentPeriodStart   *time.Time `db:"current_period_start"`
	CurrentPeriodEnd     *time.Time `db:"current_period_end"`
	CancelAtPeriodEnd    bool       `db:"cancel_at_period_end"`
	CanceledAt           *time.Time `db:"canceled_at"`
	EndedAt              *time.Time `db:"ended_at"`
	LatestEventID        string     `db:"latest_event_id"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
}

type SubscriptionResponse struct {
	StripePriceID       string     `json:"stripe_price_id"`
	Status              string     `json:"status"`
	IsActive            bool       `json:"is_active"`
	CurrentPeriodStart  *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd    *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd   bool       `json:"cancel_at_period_end"`
	CanceledAt          *time.Time `json:"canceled_at,omitempty"`
	EndedAt             *time.Time `json:"ended_at,omitempty"`
}