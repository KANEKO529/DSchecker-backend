// internal/service/stripe_webhook_service.go
package service

import (
	"encoding/json"
	"errors"
	"time"

	"dscheckerapp/internal/model"
	"github.com/stripe/stripe-go/v84"
)

type SubscriptionRepository interface {
	FindUserIDByStripeCustomerID(customerID string) (int64, error)
	UpsertSubscription(input model.UpsertSubscriptionInput) error
}

type StripeWebhookService struct {
	subscriptionRepo SubscriptionRepository
}

func NewStripeWebhookService(subscriptionRepo SubscriptionRepository) *StripeWebhookService {
	return &StripeWebhookService{
		subscriptionRepo: subscriptionRepo,
	}
}

func (s *StripeWebhookService) HandleEvent(event stripe.Event) error {
	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		return s.handleSubscriptionEvent(event)
	default:
		return nil
	}
}

func (s *StripeWebhookService) handleSubscriptionEvent(event stripe.Event) error {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return err
	}

	customerID := sub.Customer.ID
	if customerID == "" {
		return errors.New("missing stripe customer id")
	}

	userID, err := s.subscriptionRepo.FindUserIDByStripeCustomerID(customerID)
	if err != nil {
		return err
	}

	var currentPeriodStart int64
	var currentPeriodEnd int64
	priceID := ""

	if len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]

		currentPeriodStart = item.CurrentPeriodStart
		currentPeriodEnd = item.CurrentPeriodEnd

		if item.Price != nil {
			priceID = item.Price.ID
		}
	}

	input := model.UpsertSubscriptionInput{
		UserID:               userID,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: sub.ID,
		StripePriceID:        priceID,
		Status:               model.SubscriptionStatus(sub.Status),
		CurrentPeriodStart:   unixToTimePtr(currentPeriodStart),
		CurrentPeriodEnd:     unixToTimePtr(currentPeriodEnd),
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		CanceledAt:           unixToTimePtr(sub.CanceledAt),
		EndedAt:              unixToTimePtr(sub.EndedAt),
		LatestEventID:        event.ID,
	}

	return s.subscriptionRepo.UpsertSubscription(input)
}

func unixToTimePtr(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}