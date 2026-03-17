// internal/service/clerk_webhook_service.go
// handolerで受信されたHTTPリクエストをrepository(DB操作)に流すための通過処理
package service

import (
	"context"
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

func (s *ClerkWebhookService) HandleUserCreated(ctx context.Context, payload model.ClerkUserPayload) error {
	name := extractFullName(payload)

	user := model.User{
		ClerkUserID: payload.ID,
		Email:       extractPrimaryEmail(payload),
		UserName:    &name,
		Role:        "user",
	}
	return s.UserRepo.UpsertUser(ctx, user)
}

func (s *ClerkWebhookService) HandleUserUpdated(ctx context.Context, payload model.ClerkUserPayload) error {
	name := extractFullName(payload)

	user := model.User{
		ClerkUserID: payload.ID,
		Email:       extractPrimaryEmail(payload),
		UserName:    &name,
		Role:        "user",
	}
	return s.UserRepo.UpsertUser(ctx, user)
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

func extractFullName(u model.ClerkUserPayload) string {
	full := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if full != "" {
		return full
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.LastName != "" {
		return u.LastName
	}
	if len(u.ID) >= 8 {
		return "User " + u.ID[:8]
	}
	return "User " + u.ID
}