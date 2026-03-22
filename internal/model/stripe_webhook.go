// internal/model/stripe_webhook.go
package model

import "time"

type SubscriptionStatus string

const (
	StatusActive             SubscriptionStatus = "active"
	StatusTrialing           SubscriptionStatus = "trialing"
	StatusPastDue            SubscriptionStatus = "past_due"
	StatusCanceled           SubscriptionStatus = "canceled"
	StatusUnpaid             SubscriptionStatus = "unpaid"
	StatusIncomplete         SubscriptionStatus = "incomplete"
	StatusIncompleteExpired  SubscriptionStatus = "incomplete_expired"
)

type UpsertSubscriptionInput struct {
	UserID int64 // internal user id

	StripeCustomerID     string // cus_xxx
	StripeSubscriptionID string // sub_xxx
	StripePriceID        string // price_xxx

	Status SubscriptionStatus

	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time

	CancelAtPeriodEnd bool

	CanceledAt *time.Time
	EndedAt    *time.Time

	LatestEventID string // evt_xxx
}