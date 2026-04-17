// internal/service/price_search_service.go
package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"dscheckerapp/internal/client"
	"dscheckerapp/internal/model"
	"dscheckerapp/internal/repository"
)

// このユーザーはどの階層か
// 一般利用者の tier
type UsageTier string

const (
	TierGuest UsageTier = "guest"
	TierFree  UsageTier = "free"
	TierPro   UsageTier = "pro"
)

const ActionTypePriceSearch = "price_search"

var tierLimits = map[UsageTier]int{
	TierGuest: 10,
	TierFree:  15,
	TierPro:   -1,
}

var ErrUsageLimitExceeded = errors.New("usage limit exceeded")
var ErrInvalidModelNumber = errors.New("invalid model number")

type PriceSearchService struct {
	UsageLogRepo     *repository.UsageLogRepository
	SubscriptionRepo *repository.SubscriptionRepository
	KotoDBClient     *client.KotoDBClient
}

func NewPriceSearchService(
	usageLogRepo *repository.UsageLogRepository,
	subscriptionRepo *repository.SubscriptionRepository,
	kotoDBClient *client.KotoDBClient,
) *PriceSearchService {
	return &PriceSearchService{
		UsageLogRepo:     usageLogRepo,
		SubscriptionRepo: subscriptionRepo,
		KotoDBClient:     kotoDBClient,
	}
}

type ExecutePriceSearchInput struct {
	ModelNumber     string
	UserID          *int64
	GuestIdentifier *string
}

type UsageInfo struct {
	PlanType       UsageTier  `json:"planType"`
	Limit          int        `json:"limit"`
	UsedCount      int        `json:"usedCount"`
	RemainingCount int        `json:"remainingCount"`
	IsLimited      bool       `json:"isLimited"`
	ResetAt        *time.Time `json:"resetAt,omitempty"`
}

type PriceSearchItem struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	ModelNumber  string  `json:"modelNumber"`
	ImageURL     *string `json:"imageUrl,omitempty"`
	RegularPrice *int64  `json:"regularPrice,omitempty"`
	MercariURL   *string `json:"mercariUrl,omitempty"`
	MarketPrice  *int64  `json:"marketPrice,omitempty"`
}

type ExecutePriceSearchResult struct {
	Data struct {
		Item *PriceSearchItem `json:"item"`
	} `json:"data"`
	Usage UsageInfo `json:"usage"`
}

func (s *PriceSearchService) Execute(
	ctx context.Context,
	input ExecutePriceSearchInput,
) (*ExecutePriceSearchResult, error) {
	modelNumber := strings.TrimSpace(strings.ToUpper(input.ModelNumber))
	if modelNumber == "" {
		return nil, ErrInvalidModelNumber
	}

	planType := TierGuest
	limit := tierLimits[TierGuest]
	since := time.Now().Add(-1 * time.Hour)

	var usedCount int
	var err error

	// ユーザ判定
	switch {
	case input.UserID != nil:

		// --- プラン判定 ---
		planType, limit, err = s.resolvePlan(ctx, *input.UserID)
		if err != nil {
			return nil, err
		}

		// --- 使用回数 ---
		usedCount, err = s.UsageLogRepo.CountRecentByUserID(
			ctx,
			*input.UserID,
			ActionTypePriceSearch,
			since,
		)

	case input.GuestIdentifier != nil && *input.GuestIdentifier != "":
		usedCount, err = s.UsageLogRepo.CountRecentByGuestIdentifier(
			ctx,
			*input.GuestIdentifier,
			ActionTypePriceSearch,
			since,
		)

	default:
		return nil, errors.New("actor is required")
	}
	if err != nil {
		return nil, err
	}

	// ここで制御
	if limit != -1 && usedCount >= limit {

		log.Printf("ErrUsageLimitExceeded usedCount=%d, limit=%d", usedCount, limit)

		return &ExecutePriceSearchResult{
			Usage: UsageInfo{
				PlanType:       planType,
				Limit:          limit,
				UsedCount:      usedCount,
				RemainingCount: 0,
				IsLimited:      true,
			},
		}, ErrUsageLimitExceeded
	}

	// Rails API call
	railsRes, err := s.KotoDBClient.SearchByModelNumber(ctx, modelNumber)
	if err != nil {
		if errors.Is(err, client.ErrItemNotFound) {
			now := time.Now()
			log := model.UsageLog{
				UserID:          input.UserID,
				GuestIdentifier: input.GuestIdentifier,
				ActionType:      ActionTypePriceSearch,
				UsedAt:          now,
			}
			if createErr := s.UsageLogRepo.Create(ctx, log); createErr != nil {
				return nil, createErr
			}

			return &ExecutePriceSearchResult{
				Usage: UsageInfo{
					PlanType:       planType,
					Limit:          limit,
					UsedCount:      usedCount + 1,
					RemainingCount: calcRemaining(limit, usedCount+1),
					IsLimited:      false,
				},
			}, client.ErrItemNotFound
		}
		return nil, err
	}

	now := time.Now()
	log := model.UsageLog{
		UserID:          input.UserID,
		GuestIdentifier: input.GuestIdentifier,
		ActionType:      ActionTypePriceSearch,
		UsedAt:          now,
	}
	//	ここでDB追加
	if err := s.UsageLogRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	result := &ExecutePriceSearchResult{
		Usage: UsageInfo{
			PlanType:       planType,
			Limit:          limit,
			UsedCount:      usedCount + 1,
			RemainingCount: calcRemaining(limit, usedCount+1),
			IsLimited:      false,
		},
	}

	result.Data.Item = &PriceSearchItem{
		ID:           railsRes.Data.ID,
		Name:         railsRes.Data.ItemName,
		ModelNumber:  railsRes.Data.ModelNumber,
		ImageURL:     railsRes.Data.ImageURL,
		RegularPrice: railsRes.Data.RegularPrice,
		MercariURL:   railsRes.Data.MercariURL,
		MarketPrice:  railsRes.Data.MarketPrice,
	}

	return result, nil
}

func (s *PriceSearchService) resolvePlan(ctx context.Context, userID int64) (UsageTier, int, error) {
	sub, err := s.SubscriptionRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return "", 0, err
	}

	if sub != nil {
		return TierPro, tierLimits[TierPro], nil
	}

	return TierFree, tierLimits[TierFree], nil
}

func calcRemaining(limit, used int) int {
	if limit == -1 {
		return -1
	}
	return limit - used
}
