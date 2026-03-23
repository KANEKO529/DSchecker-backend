// internal/service/subscription_service.go
package service

import (
	"context"
	"errors"
	"time"

	"dscheckerapp/internal/model"
	"dscheckerapp/internal/repository"
)

type SubscriptionQueryRepository interface {
	FindLatestByUserID(ctx context.Context, userID int64) (*model.Subscription, error)
}

type UserQueryRepository interface {
	GetByClerkUserID(ctx context.Context, clerkUserID string) (*model.User, error)
}

type SubscriptionService struct {
	userRepo         UserQueryRepository
	subscriptionRepo SubscriptionQueryRepository
}

func NewSubscriptionService(
	userRepo UserQueryRepository,
	subscriptionRepo SubscriptionQueryRepository,
) *SubscriptionService {
	return &SubscriptionService{
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

func (s *SubscriptionService) GetMySubscriptionByClerkUserID(
	ctx context.Context,
	clerkUserID string,
) (*model.SubscriptionResponse, error) {
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return nil, err
	}

	return s.GetMySubscription(ctx, user.ID)
}

func (s *SubscriptionService) GetMySubscription(
	ctx context.Context,
	userID int64,
) (*model.SubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.FindLatestByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &model.SubscriptionResponse{
		StripePriceID:      sub.StripePriceID,
		Status:             sub.Status,
		IsActive:           isActiveSubscription(sub.Status, sub.EndedAt),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		CanceledAt:         sub.CanceledAt,
		EndedAt:            sub.EndedAt,
	}, nil
}

func isActiveSubscription(status string, endedAt *time.Time) bool {
	if endedAt != nil {
		return false
	}

	switch status {
	case "active", "trialing":
		return true
	default:
		return false
	}
}