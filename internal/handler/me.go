// intenal/handler/me.go
package handler

import (
	"dscheckerapp/internal/dto"
	"dscheckerapp/internal/service"

	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type MeHandler struct {
	meService      *service.MeService
	accountService *service.AccountService
}

type UpdateMyProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}

func NewMeHandler(meService *service.MeService, accountService *service.AccountService) *MeHandler {
	return &MeHandler{meService: meService, accountService: accountService}
}

func (h *MeHandler) Me(c *gin.Context) {
	clerkUserIDValue, exists := c.Get("clerk_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "error",
			"error":  "clerk_user_id not found in context",
		})
		return
	}

	clerkUserID, ok := clerkUserIDValue.(string)
	if !ok || clerkUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status": "error",
			"error":  "invalid clerk_user_id",
		})
		return
	}

	// user, err := h.UserRepo.GetByClerkUserID(c.Request.Context(), clerkUserID)
	user, err := h.meService.GetMe(c.Request.Context(), clerkUserID)
	if err != nil {
		log.Printf("[ME] GetByClerkUserID error: %#v", err)

		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"status": "error",
				"error":  "user not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"user": user,
		},
	})
}

// func (h *MeHandler) UpdateMyProfile(c *gin.Context) {
// 	var req dto.UpdateMyProfileRequest

// 	// ここでリクエストボディを取得
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"status":  "error",
// 			"message": "invalid request",
// 		})
// 		return
// 	}

// 	clerkUserIDValue, exists := c.Get("clerk_user_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"status":  "error",
// 			"message": "unauthorized",
// 		})
// 		return
// 	}

// 	clerkUserID, ok := clerkUserIDValue.(string)
// 	if !ok || clerkUserID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"status":  "error",
// 			"message": "unauthorized",
// 		})
// 		return
// 	}

// 	// servise呼び出し
// 	if err := h.meService.UpdateMyProfile(c.Request.Context(), clerkUserID, req); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"status":  "error",
// 			"message": err.Error(),
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"status":  "success",
// 		"message": "profile updated",
// 	})
// }

func (h *MeHandler) UpdateMyProfile(c *gin.Context) {
	var req dto.UpdateMyProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[UpdateMyProfile] bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid request",
		})
		return
	}

	log.Printf("req ", req)

	log.Printf("[UpdateMyProfile] req firstName=%q lastName=%q", req.FirstName, req.LastName)

	clerkUserIDValue, exists := c.Get("clerk_user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "unauthorized",
		})
		return
	}

	clerkUserID, ok := clerkUserIDValue.(string)
	if !ok || clerkUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "unauthorized",
		})
		return
	}

	if err := h.meService.UpdateMyProfile(c.Request.Context(), clerkUserID, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "profile updated",
	})
}

func (h *MeHandler) DeleteMyAccount(c *gin.Context) {
	clerkUserID := c.GetString("clerk_user_id")
	if clerkUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	err := h.accountService.DeleteMyAccountByClerkUserID(
		c.Request.Context(),
		clerkUserID,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSubscriptionPastDue):
			c.JSON(http.StatusConflict, gin.H{
				"error": "subscription past due",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to delete account",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "account deleted",
	})
}
