// internal/service/clerk_webhook_service.go
// handolerで受信されたHTTPリクエストをrepository(DB操作)に流すための通過処理
package service

import (
	"context"
	"log"
	"strings"

	"dscheckerapp/internal/model"
	"dscheckerapp/internal/repository"
)

type ClerkWebhookService struct {
	UserRepo *repository.UserRepository
}

func NewClerkWebhookService(userRepo *repository.UserRepository) *ClerkWebhookService {
	return &ClerkWebhookService{UserRepo: userRepo}
}

// 新規追加時
func (s *ClerkWebhookService) HandleUserCreated(ctx context.Context, payload model.ClerkUserPayload) error {
	name := extractDisplayName(payload)

	user := model.User{
		ClerkUserID: payload.ID,
		Email:       extractPrimaryEmail(payload),
		UserName:    &name,
		Role:        "user",
		Status:      "active",
	}

	err := s.UserRepo.CreateOrReRegisterUser(ctx, user)
	if err != nil {
		log.Printf("[CLERK WEBHOOK] HandleUserCreated failed: clerk_user_id=%s email=%s err=%v",
			user.ClerkUserID, user.Email, err)
		return err
	}

	log.Printf("[CLERK WEBHOOK] HandleUserCreated success: clerk_user_id=%s email=%s",
		user.ClerkUserID, user.Email)

	return nil
}

// 更新時
func (s *ClerkWebhookService) HandleUserUpdated(ctx context.Context, payload model.ClerkUserPayload) error {
	name := extractDisplayName(payload)
	return s.UserRepo.UpdateProfileByClerkID(
		ctx,
		payload.ID,
		extractPrimaryEmail(payload),
		&name,
	)
}

func (s *ClerkWebhookService) HandleUserDeleted(ctx context.Context, clerkUserID string) error {
	return s.UserRepo.DeleteUserByClerkID(ctx, clerkUserID)
}

func extractPrimaryEmail(u model.ClerkUserPayload) string {
	for _, email := range u.EmailAddresses {
		if email.ID == u.PrimaryEmailAddressID {
			return email.EmailAddress
		}
	}
	if len(u.EmailAddresses) > 0 {
		return u.EmailAddresses[0].EmailAddress
	}
	return "user_" + u.ID + "@example.com"
}

// helper function
func extractDisplayName(payload model.ClerkUserPayload) string {
	first := strings.TrimSpace(payload.FirstName)
	last := strings.TrimSpace(payload.LastName)

	full := strings.TrimSpace(last + first)
	if full != "" {
		return full
	}
	if first != "" {
		return first
	}
	if last != "" {
		return last
	}

	email := extractPrimaryEmail(payload)
	if email != "" {
		parts := strings.Split(email, "@")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	return "user"
}
