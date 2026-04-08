// internal/service/me_service.go
package service

import (
	"context"
	"errors"
	"strings"

	"dscheckerapp/internal/dto"
	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/model"
)

type MeService struct {
	clerkClient *lib.ClerkClient
	userRepo    UserQueryRepository
}

// type UpdateMyProfileRequest struct {
// 	FirstName string `json:"firstName"`
// 	LastName  string `json:"lastName"`
// }

func NewMeService(clerkClient *lib.ClerkClient, userRepo UserQueryRepository) *MeService {
	return &MeService{
		clerkClient: clerkClient,
		userRepo:    userRepo,
	}
}

func (s *MeService) GetMe(ctx context.Context, clerkUserID string) (*model.User, error) {
	return s.userRepo.GetByClerkUserID(ctx, clerkUserID)
}

func (s *MeService) UpdateMyProfile(ctx context.Context, clerkUserID string, req dto.UpdateMyProfileRequest) error {
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)

	if firstName == "" {
		return errors.New("firstName is required")
	}
	if lastName == "" {
		return errors.New("lastName is required")
	}

	return s.clerkClient.UpdateName(ctx, clerkUserID, firstName, lastName)
}
