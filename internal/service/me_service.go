// internal/service/me_service.go
package service

import (
	"context"
	"errors"
	"strings"

	"dscheckerapp/internal/dto"
	"dscheckerapp/internal/lib"
)

type MeService struct {
	clerkClient *lib.ClerkClient
}

type UpdateMyProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}

func NewMeService(clerkClient *lib.ClerkClient) *MeService {
	return &MeService{
		clerkClient: clerkClient,
	}
}

func (s *MeService) UpdateMyProfile(ctx context.Context, clerkUserID string, req dto.UpdateMyProfileRequest) error {
	username := strings.TrimSpace(req.Username)
	if username == "" {
		return errors.New("username is required")
	}

	return s.clerkClient.UpdateUsername(ctx, clerkUserID, username)
}
