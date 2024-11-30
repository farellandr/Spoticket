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

		eventPublic := public.Group("/events")
		{
			eventPublic.GET("", handlers.ListEvents)
			eventPublic.GET("/:id", handlers.GetEvent)
		}
	}

	protected := r.Group("/v1")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		eventProtected := protected.Group("/events")
		{
			eventProtected.POST("", handlers.CreateEvent)
			eventProtected.PUT("/:id", handlers.UpdateEvent)
			eventProtected.DELETE("/:id", handlers.DeleteEvent)
		}
	}
}
