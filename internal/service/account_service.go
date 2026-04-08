// internal/service/account_service.go
// dschecker(clerk)のアカウント削除ロジック
// 必ずサブスクの状態を確認(契約中なら即時キャンセルが成功したらclerkのアカウントを削除)
package service

import (
	"context"
	"dscheckerapp/internal/repository"
	"errors"
)

type ClerkUserDeletionClient interface {
	DeleteUser(ctx context.Context, clerkUserID string) error
}

type AccountService struct {
	userRepo         UserQueryRepository
	subscriptionRepo SubscriptionQueryRepository
	stripeClient     StripeSubscriptionClient
	clerkClient      ClerkUserDeletionClient
}

func NewAccountService(
	userRepo UserQueryRepository,
	subscriptionRepo SubscriptionQueryRepository,
	stripeClient StripeSubscriptionClient,
	clerkClient ClerkUserDeletionClient,
) *AccountService {
	return &AccountService{
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		stripeClient:     stripeClient,
		clerkClient:      clerkClient,
	}
}

var ErrAccountDeletionNotAllowed = errors.New("account deletion not allowed")
var ErrSubscriptionPastDue = errors.New("subscription past due")

func (s *AccountService) DeleteMyAccountByClerkUserID(
	ctx context.Context,
	clerkUserID string,
) error {
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return err
	}

	return s.DeleteMyAccount(ctx, user.ID, user.ClerkUserID)
}

func (s *AccountService) DeleteMyAccount(
	ctx context.Context,
	userID int64,
	clerkUserID string,
) error {
	sub, err := s.subscriptionRepo.FindCurrentByUserID(ctx, userID)
	if err != nil && !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return err
	}

	if err == nil && sub != nil {
		if sub.EndedAt == nil {
			switch sub.Status {
			case "past_due":
				return ErrSubscriptionPastDue
			case "active", "trialing":
				if err := s.stripeClient.CancelImmediately(ctx, sub.StripeSubscriptionID); err != nil {
					return err
				}
			}
		}
	}

	if err := s.clerkClient.DeleteUser(ctx, clerkUserID); err != nil {
		return err
	}

	return nil
}
