package server

import (
	"fmt"
	"os"

	"github.com/farellandr/spoticket/config"
	"github.com/farellandr/spoticket/internal/handlers"
	"github.com/farellandr/spoticket/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Start() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	db, err := config.InitDatabase(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}

	r := gin.Default()

	setupRoutes(r, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return r.Run(":" + port)
}

func setupRoutes(r *gin.Engine, db *gorm.DB) {
	r.Use(middleware.DatabaseMiddleware(db))

	public := r.Group("/v1")
	{
		public.POST("/register", handlers.Register)
		public.POST("/login", handlers.Login)

		categoryPublic := public.Group("/categories")
		{
			categoryPublic.GET("", handlers.ListCategories)
		}

		eventPublic := public.Group("/events")
		{
			eventPublic.GET("", handlers.ListEvents)
			eventPublic.GET("/:id", handlers.GetEvent)
			eventPublic.GET("/:id/banner", handlers.StreamEventBanner)
		}

		ticketPublic := public.Group("/tickets")
		{
			ticketPublic.GET("/:id", handlers.GetTicket)
		}

		couponPublic := public.Group("/coupons")
		{
			couponPublic.GET("", handlers.ListCoupons)
			couponPublic.GET("/:id", handlers.GetCoupon)
		}

		paymentPublic := public.Group("/payments")
		{
			paymentPublic.POST("/notification", handlers.PaymentNotification)
		}
	}

	protected := r.Group("/v1")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		profileProtected := protected.Group("/profile")
		{
			profileProtected.GET("", handlers.GetProfile)
			profileProtected.PUT("/change-password", handlers.ChangePassword)
		}

		categoryProtected := protected.Group("/categories")
		{
			categoryProtected.POST("", handlers.CreateCategory)
			categoryProtected.PUT("/:id", handlers.UpdateCategory)
			categoryProtected.DELETE("/:id", handlers.DeleteCategory)
		}

		eventProtected := protected.Group("/events")
		{
			eventProtected.POST("", handlers.CreateEvent)
			eventProtected.PUT("/:id", handlers.UpdateEvent)
			eventProtected.DELETE("/:id", handlers.DeleteEvent)
		}

		ticketProtected := protected.Group("/tickets")
		{
			ticketProtected.POST("", handlers.CreateTicket)
			ticketProtected.PUT("/:id", handlers.UpdateTicket)
			ticketProtected.DELETE("/:id", handlers.DeleteTicket)
			ticketProtected.POST("/validate", handlers.ValidateTicket)
		}

		couponProtected := protected.Group("/coupons")
		{
			couponProtected.POST("", handlers.CreateCoupon)
			couponProtected.POST("/claim", handlers.ClaimCoupon)
			couponProtected.PUT("/:id", handlers.UpdateCoupon)
			couponProtected.DELETE("/:id", handlers.DeleteCoupon)
		}

		paymentProtected := protected.Group("/payments")
		{
			paymentProtected.POST("", handlers.CreatePaymentLink)
		}

		purchaseProtected := protected.Group("/purchases")
		{
			purchaseProtected.GET(":purchaseId/qr", handlers.GenerateTicketQR)
		}
	}
}
