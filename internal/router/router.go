// internal/router.go
package router

import (
	"log"
	"os"
	"time"

	"dscheckerapp/internal/auth"
	"dscheckerapp/internal/client"
	"dscheckerapp/internal/handler"
	"dscheckerapp/internal/lib"
	"dscheckerapp/internal/middleware"
	"dscheckerapp/internal/repository"
	"dscheckerapp/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRouter(db *pgxpool.Pool, stripeCfg *lib.StripeConfig, clerkCfg *lib.ClerkConfig) *gin.Engine {
	r := gin.Default()

	frontendOrigin := os.Getenv("CORS_ORIGIN")
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:3000"
	}
	kotoDBBaseURL := os.Getenv("KOTODB_API_BASE_URL")
	kotoDBInternalAPIKey := os.Getenv("KOTODB_INTERNAL_API_KEY")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", handler.Health)

	kotoDBClient := client.NewKotoDBClient(kotoDBBaseURL, kotoDBInternalAPIKey)

	// lib
	stripeClient := lib.NewStripeClient()
	clerkClient := lib.NewClerkClient(clerkCfg)

	// repository
	userRepo := repository.NewUserRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	usageLogRepo := repository.NewUsageLogRepository(db)

	// service
	meService := service.NewMeService(clerkClient, userRepo)
	accountService := service.NewAccountService(
		userRepo,
		subscriptionRepo,
		stripeClient,
		clerkClient,
	)
	clerkWebhookService := service.NewClerkWebhookService(userRepo)
	stripeWebhookService := service.NewStripeWebhookService(subscriptionRepo)
	billingService := service.NewBillingService(stripeCfg, stripeClient, userRepo, subscriptionRepo)
	subscriptionService := service.NewSubscriptionService(
		userRepo,
		subscriptionRepo,
		stripeClient,
	)
	priceSearchService := service.NewPriceSearchService(
		usageLogRepo,
		subscriptionRepo,
		kotoDBClient,
	)

	// handler
	userHandler := handler.NewUserHandler(userRepo)
	priceSearchHandler := handler.NewPriceSearchHandler(priceSearchService)

	meHandler := handler.NewMeHandler(meService, accountService)
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

	authMiddleware := middleware.NewAuthMiddleware(verifier, userRepo)

	api := r.Group("/api/v1")
	{
		api.GET("/users", userHandler.GetUsers)
		api.POST("/webhooks/clerk", clerkWebhookHandler.Handle)
		api.POST("/webhooks/stripe", stripeWebhookHandler.Handle)
		// RequireAuth()は、トークンがあれば検証、401にしない
		api.POST("/price-searches", authMiddleware.OptionalAuth(), priceSearchHandler.Create)

		protected := api.Group("")
		// RequireAuth()はトークン必須、必ずログイン済みユーザを通したい時、なければ401
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.GET("/me", meHandler.Me)
			protected.DELETE("/me", meHandler.DeleteMyAccount)

			protected.PATCH("/me/profile", meHandler.UpdateMyProfile)

			protected.GET("/me/subscription", subscriptionHandler.GetMySubscription)
			protected.POST("/me/subscription/cancel", subscriptionHandler.CancelMySubscription)
			protected.POST("/me/subscription/resume", subscriptionHandler.ResumeMySubscription)

			protected.POST("/billing/checkout-session", billingHandler.CreateCheckoutSession)
			protected.GET("/billing/summary", billingHandler.GetBillingSummary)
			// protected.GET("/billing/payment-methods", billingHandler.GetPaymentMethods)
			// protected.GET("/billing/invoices", billingHandler.GetListInvoices)
			protected.POST("/billing/customer-portal", billingHandler.CreateCustomerPortal)

		}
	}

	return r
}
