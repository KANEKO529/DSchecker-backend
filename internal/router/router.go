package router

import (
	"dscheckerapp/internal/handler"
	"dscheckerapp/internal/repository"
	"dscheckerapp/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(db *pgxpool.Pool) *gin.Engine {
	r := gin.Default()

	r.GET("/health", handler.Health)

	userRepo := repository.NewUserRepository(db)

	userHandler := handler.NewUserHandler(userRepo)

	clerkWebhookService := service.NewClerkWebhookService(userRepo)
	clerkWebhookHandler := handler.NewClerkWebhookHandler(clerkWebhookService)

	api := r.Group("/api/v1")
	{
		api.GET("/users", userHandler.GetUsers)
		api.POST("/webhooks/clerk", clerkWebhookHandler.Handle)
	}

	return r
}