// internal/router.go
package router

import (
	"log"
	"os"
	"time"

	"dscheckerapp/internal/auth"
	"dscheckerapp/internal/handler"
	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/middleware"
	"dscheckerapp/internal/repository"
	"dscheckerapp/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(db *pgxpool.Pool, stripeCfg *lib.StripeConfig) *gin.Engine {
	r := gin.Default()

	frontendOrigin := os.Getenv("CORS_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:3000"
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", handler.Health)

	// lib
	stripeClient := lib.NewStripeClient()


	// repository
	userRepo := repository.NewUserRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)

	// service
	clerkWebhookService := service.NewClerkWebhookService(userRepo)
	stripeWebhookService := service.NewStripeWebhookService(subscriptionRepo)
	billingService := service.NewBillingService(stripeCfg, userRepo, subscriptionRepo)
	subscriptionService := service.NewSubscriptionService(
		userRepo,
		subscriptionRepo,
		stripeClient,
	)

	// handler
	userHandler := handler.NewUserHandler(userRepo)
	clerkWebhookHandler := handler.NewClerkWebhookHandler(clerkWebhookService)
	stripeWebhookHandler := handler.NewStripeWebhookHandler(
		stripeWebhookService,
		stripeCfg,
	)
	billingHandler := handler.NewBillingHandler(billingService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)

	verifier, err := auth.NewClerkVerifier()
	if err != nil {
		log.Fatalf("failed to initialize clerk verifier: %v", err)
	}

	authMiddleware := middleware.NewAuthMiddleware(verifier)

	api := r.Group("/api/v1")
	{
		api.GET("/users", userHandler.GetUsers)
		api.POST("/webhooks/clerk", clerkWebhookHandler.Handle)
		api.POST("/webhooks/stripe", stripeWebhookHandler.Handle)

		protected := api.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.GET("/me", userHandler.Me)
			
			protected.GET("/me/subscription", subscriptionHandler.GetMySubscription)
			protected.POST("/me/subscription/cancel", subscriptionHandler.CancelMySubscription)
			protected.POST("/me/subscription/resume", subscriptionHandler.ResumeMySubscription)

			protected.POST("/billing/checkout-session", billingHandler.CreateCheckoutSession)
		}
	}

	return r
}