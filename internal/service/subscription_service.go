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
	FindCurrentByUserID(ctx context.Context, userID int64) (*model.Subscription, error)
}

type UserQueryRepository interface {
	GetByClerkUserID(ctx context.Context, clerkUserID string) (*model.User, error)
}

// stripe送信用
type StripeSubscriptionClient interface {
	CancelImmediately(ctx context.Context, stripeSubscriptionID string) error
	CancelAtPeriodEnd(ctx context.Context, stripeSubscriptionID string) error
	ResumeSubscription(ctx context.Context, stripeSubscriptionID string) error
}

type SubscriptionService struct {
	userRepo         UserQueryRepository
	subscriptionRepo SubscriptionQueryRepository
	stripeClient     StripeSubscriptionClient
}

var ErrSubscriptionNotCancelable = errors.New("subscription not cancelable")
var ErrSubscriptionNotResumable = errors.New("subscription not resumable")

func NewSubscriptionService(
	userRepo UserQueryRepository,
	subscriptionRepo SubscriptionQueryRepository,
	stripeClient StripeSubscriptionClient,
) *SubscriptionService {
	return &SubscriptionService{
		userRepo:         userRepo,
		subscriptionRepo: subscriptionRepo,
		stripeClient:     stripeClient,
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
	sub, err := s.subscriptionRepo.FindCurrentByUserID(ctx, userID)
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

// Serviceを認証に依存させないために分離
// 外部（Handler）から呼ばれる用(入口)
// ここは名前を変えない.
func (s *SubscriptionService) CancelMySubscriptionByClerkUserID(
	ctx context.Context,
	clerkUserID string,
) error {
	// clerk_id取得
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return err
	}

	return s.CancelMySubscriptionAtPeriodEnd(ctx, user.ID)
}

// 純粋なビジネスロジック
// ① 現在の契約を確認
// ② 解約できるか判定
// ③ Stripeに命令を送る
// ④ 終わり（DB更新しない）
func (s *SubscriptionService) CancelMySubscriptionAtPeriodEnd(
	ctx context.Context,
	userID int64,
) error {

	// ① 現在の契約を確認
	sub, err := s.subscriptionRepo.FindCurrentByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if sub.StripeSubscriptionID == "" {
		return repository.ErrSubscriptionNotFound
	}

	// すでに終了している契約は解約予約できない
	if sub.EndedAt != nil {
		return ErrSubscriptionNotCancelable
	}

	// status 判定
	// active, trialingのみを通す
	// もし、それ以外ならエラー
	switch sub.Status {
	case "active", "trialing":
	default:
		return ErrSubscriptionNotCancelable
	}

	// すでに解約予約済みなら何もしない
	// 冪等性
	if sub.CancelAtPeriodEnd {
		return nil
	}

	//ここでキャンセル処理：満了キャンセル
	if err := s.stripeClient.CancelAtPeriodEnd(ctx, sub.StripeSubscriptionID); err != nil {
		return err
	}

	return nil
}

// handlerから呼び出し 現在の実装では使用しない、
// func (s *SubscriptionService) CancelMySubscriptionImmediatelyByClerkUserID(
// 	ctx context.Context,
// 	clerkUserID string,
// ) error {
// 	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
// 	if err != nil {
// 		return err
// 	}

// 	return s.CancelMySubscriptionImmediately(ctx, user.ID)
// }

func (s *SubscriptionService) CancelMySubscriptionImmediately(
	ctx context.Context,
	userID int64,
) error {
	sub, err := s.subscriptionRepo.FindCurrentByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if sub.StripeSubscriptionID == "" {
		return repository.ErrSubscriptionNotFound
	}

	// すでに終了済みなら不可
	if sub.EndedAt != nil {
		return ErrSubscriptionNotCancelable
	}

	// 即時キャンセルを許可する状態
	switch sub.Status {
	case "active", "trialing":
	default:
		return ErrSubscriptionNotCancelable
	}

	if err := s.stripeClient.CancelImmediately(ctx, sub.StripeSubscriptionID); err != nil {
		return err
	}

	return nil
}

func (s *SubscriptionService) ResumeMySubscriptionByClerkUserID(
	ctx context.Context,
	clerkUserID string,
) error {
	user, err := s.userRepo.GetByClerkUserID(ctx, clerkUserID)
	if err != nil {
		return err
	}

	return s.ResumeMySubscription(ctx, user.ID)
}

func (s *SubscriptionService) ResumeMySubscription(
	ctx context.Context,
	userID int64,
) error {
	sub, err := s.subscriptionRepo.FindCurrentByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if sub.StripeSubscriptionID == "" {
		return repository.ErrSubscriptionNotFound
	}

	if sub.EndedAt != nil {
		return ErrSubscriptionNotResumable
	}

	switch sub.Status {
	case "active", "trialing":
	default:
		return ErrSubscriptionNotResumable
	}

	// すでに継続状態なら成功扱い
	if !sub.CancelAtPeriodEnd {
		return nil
	}

	if err := s.stripeClient.ResumeSubscription(ctx, sub.StripeSubscriptionID); err != nil {
		return err
	}

	return nil
}
