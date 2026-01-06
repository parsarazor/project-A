package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/database"
	"github.com/url-shortener/internal/handler"
	"github.com/url-shortener/internal/middleware"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(database.GetDB())
	urlRepo := repository.NewURLRepository(database.GetDB())

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg)
	urlService := service.NewURLService(urlRepo, cfg.App.BaseURL)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	urlHandler := handler.NewURLHandler(urlService)
	adminHandler := handler.NewAdminHandler(urlService, authService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "URL Shortener API",
		ErrorHandler: customErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// Auth routes (protected)
	authProtected := auth.Group("", authMiddleware.Authenticate())
	authProtected.Get("/me", authHandler.GetMe)
	authProtected.Post("/change-password", authHandler.ChangePassword)

	// URL routes (protected)
	urls := api.Group("/urls", authMiddleware.Authenticate())
	urls.Post("", urlHandler.CreateURL)
	urls.Get("/me", urlHandler.GetMyURLs)
	urls.Get("/stats", urlHandler.GetUserStats)
	urls.Get("/:id", urlHandler.GetURLByID)
	urls.Delete("/:id", urlHandler.DeleteURL)
	urls.Patch("/:id/deactivate", urlHandler.DeactivateURL)
	urls.Patch("/:id/activate", urlHandler.ActivateURL)

	// Admin routes (protected + admin only)
	admin := api.Group("/admin", authMiddleware.Authenticate(), middleware.RequireAdmin())
	admin.Get("/urls", adminHandler.GetAllURLs)
	admin.Get("/urls/:id", adminHandler.GetURLByID)
	admin.Delete("/urls/:id", adminHandler.DeleteURL)
	admin.Patch("/urls/:id/deactivate", adminHandler.DeactivateURL)
	admin.Patch("/urls/:id/activate", adminHandler.ActivateURL)

	// Redirect route (public) - must be last to avoid conflicts
	app.Get("/:shortCode", urlHandler.Redirect)

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.App.Port)
		log.Printf("Server starting on %s", addr)
		log.Printf("Environment: %s", cfg.App.Env)
		log.Printf("Base URL: %s", cfg.App.BaseURL)

		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	// Default to 500 Internal Server Error
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Return JSON error response
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error": fiber.Map{
			"code":    code,
			"message": message,
		},
	})
}
