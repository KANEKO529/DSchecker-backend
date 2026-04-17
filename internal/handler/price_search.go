// internal/handler/price_search.go
package handler

import (
	"errors"
	"net/http"

	"dscheckerapp/internal/client"
	"dscheckerapp/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PriceSearchHandler struct {
	PriceSearchService *service.PriceSearchService
}

func NewPriceSearchHandler(priceSearchService *service.PriceSearchService) *PriceSearchHandler {
	return &PriceSearchHandler{
		PriceSearchService: priceSearchService,
	}
}

type CreatePriceSearchRequest struct {
	ModelNumber string `json:"model_number"`
}

func (h *PriceSearchHandler) Create(c *gin.Context) {
	var req CreatePriceSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "リクエスト形式が不正です",
			},
		})
		return
	}

	ctx := c.Request.Context()

	// contextにuser_idがあるかチェック
	var userIDPtr *int64
	if v, exists := c.Get("user_id"); exists {
		if userID, ok := v.(int64); ok {
			userIDPtr = &userID
		}
	}
	// もしnilだったら
	var guestIDPtr *string
	if userIDPtr == nil {
		// ここでクッキーのIDを保存
		guestID, err := c.Cookie("guest_id")

		if err != nil || guestID == "" {
			guestID = uuid.NewString()
			c.SetCookie("guest_id", guestID, 60*60*24*30, "/", "", false, true)
		}

		guestIDPtr = &guestID
	}

	input := service.ExecutePriceSearchInput{
		ModelNumber:     req.ModelNumber,
		UserID:          userIDPtr,
		GuestIdentifier: guestIDPtr,
	}

	// ここで実行
	res, err := h.PriceSearchService.Execute(ctx, input)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidModelNumber):
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "INVALID_MODEL_NUMBER",
					"message": "modelNumber is required",
				},
			})

		case errors.Is(err, service.ErrUsageLimitExceeded):
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "USAGE_LIMIT_EXCEEDED",
					"message": "利用回数の上限に達しました",
				},
				"usage": res.Usage,
			})

		case errors.Is(err, client.ErrItemNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "ITEM_NOT_FOUND",
					"message": "商品が見つかりません",
				},
				"usage": res.Usage,
			})

		default:
			c.JSON(http.StatusBadGateway, gin.H{
				"status": "error",
				"error": gin.H{
					"code":    "UPSTREAM_ERROR",
					"message": "検索処理に失敗しました",
				},
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"data":   res.Data,
		"usage":  res.Usage,
	})
}
