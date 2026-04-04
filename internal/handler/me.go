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
	meService *service.MeService
}

type UpdateMyProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}

func NewMeHandler(meService *service.MeService) *MeHandler {
	return &MeHandler{meService: meService}
}

func (h *UserHandler) Me(c *gin.Context) {
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

	user, err := h.UserRepo.GetByClerkUserID(c.Request.Context(), clerkUserID)
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

func (h *MeHandler) UpdateMyProfile(c *gin.Context) {
	var req dto.UpdateMyProfileRequest

	// ここでリクエストボディを取得
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid request",
		})
		return
	}

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

	// servise呼び出し
	if err := h.meService.UpdateMyProfile(c.Request.Context(), clerkUserID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
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
