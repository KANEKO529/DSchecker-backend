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
	name := extractUserName(payload)

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
	name := extractUserName(payload)

	user := model.User{
		ClerkUserID: payload.ID,
		Email:       extractPrimaryEmail(payload),
		UserName:    &name,
	}
	return s.UserRepo.UpdateUserByClerkID(ctx, user)
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
func extractUserName(u model.ClerkUserPayload) string {
	if strings.TrimSpace(u.Username) != "" {
		return strings.TrimSpace(u.Username)
	}

	full := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if full != "" {
		return full
	}

	if len(u.ID) >= 8 {
		return "User_" + u.ID[:8]
	}
	return "User_" + u.ID
}
