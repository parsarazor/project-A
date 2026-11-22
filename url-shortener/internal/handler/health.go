package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// dbPingTimeout bounds the readiness probe so a hung database fails /ready
// fast instead of pinning the health-check connection.
const dbPingTimeout = 2 * time.Second

// RegisterHealthHandlers registers the health endpoints on the given app:
// /health is a static liveness endpoint, /ready reports database readiness.
func RegisterHealthHandlers(app *fiber.App, db *gorm.DB, version string) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": version,
		})
	})

	app.Get("/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), dbPingTimeout)
		defer cancel()

		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(ctx)
		}
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unavailable",
				"error":  "database unreachable",
			})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
}
