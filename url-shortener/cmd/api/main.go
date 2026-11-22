package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/database"
	"github.com/url-shortener/internal/handler"
	"github.com/url-shortener/internal/middleware"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/service"
)

// version is the application version, injected at build time with
// -ldflags "-X main.version=..." (dev otherwise).
var version = "dev"

func main() {
	// Load configuration (fails fast on unsafe settings outside development)
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
	fiberConfig := fiber.Config{
		AppName:      "URL Shortener API",
		ErrorHandler: customErrorHandler,
		// Cap slowloris-style abuse: fail clients that cannot send or accept
		// data in a reasonable time, and reject oversized bodies.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  75 * time.Second,
		BodyLimit:    1 << 20, // 1 MiB; requests are tiny JSON documents
	}
	// Only honor spoofable forwarded headers when the direct peer is an
	// explicitly configured proxy; otherwise c.IP() stays the peer address.
	// The rate limiters below additionally walk X-Forwarded-For right-to-left
	// so a spoofed header cannot rotate the limit key.
	trustedNets := trustedProxyNets(cfg.App.TrustedProxies)
	if len(trustedNets) > 0 {
		fiberConfig.EnableTrustedProxyCheck = true
		fiberConfig.TrustedProxies = cfg.App.TrustedProxies
		fiberConfig.ProxyHeader = fiber.HeaderXForwardedFor
		fiberConfig.EnableIPValidation = true
	}
	app := fiber.New(fiberConfig)

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(cfg.App.AllowedOrigins, ","),
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Global rate limit (per IP, per minute) - protects the public redirect
	// and every other route from abusive traffic
	app.Use(limiter.New(limiter.Config{
		Max:               cfg.RateLimit.GlobalPerMinute,
		Expiration:        time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator:      rateLimitKey(trustedNets),
	}))

	// Health endpoints: /health is a static liveness probe, /ready pings the
	// database with a short timeout and reports 503 while it is unreachable.
	handler.RegisterHealthHandlers(app, database.GetDB(), version)

	// API v1 routes
	api := app.Group("/api/v1")

	// Strict rate limit for the credential endpoints (per IP, per minute) -
	// slows down brute forcing without throttling the authenticated /auth
	// routes
	authLimiter := limiter.New(limiter.Config{
		Max:               cfg.RateLimit.AuthPerMinute,
		Expiration:        time.Minute,
		LimiterMiddleware: limiter.SlidingWindow{},
		KeyGenerator:      rateLimitKey(trustedNets),
	})

	// Auth routes (public) - only the credential endpoints are strictly
	// rate limited
	authPublic := api.Group("/auth")
	authPublic.Post("/register", authLimiter, authHandler.Register)
	authPublic.Post("/login", authLimiter, authHandler.Login)

	// Auth routes (protected) - not under the strict credential limiter
	authProtected := api.Group("/auth", authMiddleware.Authenticate())
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

	// Stop the click counter and wait for queued writes to drain
	urlService.Close()

	log.Println("Server exited properly")
}

// trustedProxyNets parses configured proxies (IPs or CIDRs) into networks.
func trustedProxyNets(proxies []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(proxies))
	for _, p := range proxies {
		if _, ipnet, err := net.ParseCIDR(p); err == nil {
			nets = append(nets, ipnet)
			continue
		}
		if ip := net.ParseIP(p); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}
	return nets
}

func ipInTrusted(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// rateLimitKey returns a limiter KeyGenerator that resolves the real client
// IP. Requests not coming from a trusted proxy are keyed by the direct peer
// address. For requests from a trusted proxy, the client is the rightmost
// non-trusted IP in X-Forwarded-For, so a client-supplied header can neither
// forge nor rotate the key.
func rateLimitKey(trusted []*net.IPNet) func(*fiber.Ctx) string {
	return func(c *fiber.Ctx) string {
		peer := c.Context().RemoteIP()
		if len(trusted) == 0 || !ipInTrusted(peer, trusted) {
			return peer.String()
		}
		parts := strings.Split(c.Get(fiber.HeaderXForwardedFor), ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				break
			}
			if ipInTrusted(ip, trusted) {
				continue
			}
			return ip.String()
		}
		return peer.String()
	}
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	// Default to 500 Internal Server Error
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it is a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Rate limit responses should hint when to retry; the limiter middleware
	// already sets the header from its own expiration, so only fill it in
	// when it is missing.
	if code == fiber.StatusTooManyRequests && c.GetRespHeader(fiber.HeaderRetryAfter) == "" {
		c.Set(fiber.HeaderRetryAfter, "60")
	}

	// Log unexpected failures (non-Fiber errors and 5xx) so production
	// incidents are visible; expected client errors (4xx) stay off the log.
	if _, isFiberErr := err.(*fiber.Error); !isFiberErr || code >= 500 {
		log.Printf("request error: method=%s path=%s status=%d err=%v", c.Method(), c.Path(), code, err)
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
