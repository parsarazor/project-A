package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/utils"
)

// RequireAdmin is a middleware that requires admin role
// Must be used after AuthMiddleware.Authenticate()
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user role from context (set by AuthMiddleware)
		role, ok := c.Locals(UserRoleContextKey).(model.Role)
		if !ok {
			return utils.Unauthorized(c, "Authentication required")
		}

		// Check if user is admin
		if role != model.RoleAdmin {
			return utils.Forbidden(c, "Admin access required")
		}

		return c.Next()
	}
}

// RequireRoles creates a middleware that requires one of the specified roles
// Must be used after AuthMiddleware.Authenticate()
func RequireRoles(roles ...model.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user role from context
		userRole, ok := c.Locals(UserRoleContextKey).(model.Role)
		if !ok {
			return utils.Unauthorized(c, "Authentication required")
		}

		// Check if user has one of the required roles
		for _, role := range roles {
			if userRole == role {
				return c.Next()
			}
		}

		return utils.Forbidden(c, "Insufficient permissions")
	}
}
