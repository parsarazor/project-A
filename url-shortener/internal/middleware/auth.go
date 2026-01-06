package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/service"
	"github.com/url-shortener/internal/utils"
)

const (
	// AuthorizationHeader is the header key for authorization
	AuthorizationHeader = "Authorization"
	// BearerPrefix is the prefix for bearer tokens
	BearerPrefix = "Bearer "
	// UserContextKey is the key used to store user info in context
	UserContextKey = "user"
	// UserIDContextKey is the key used to store user ID in context
	UserIDContextKey = "user_id"
	// UserRoleContextKey is the key used to store user role in context
	UserRoleContextKey = "user_role"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService *service.AuthService
}

// NewAuthMiddleware creates a new AuthMiddleware instance
func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Authenticate is a middleware that validates JWT tokens
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get(AuthorizationHeader)
		if authHeader == "" {
			return utils.Unauthorized(c, "Missing authorization header")
		}

		// Check for Bearer prefix
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			return utils.Unauthorized(c, "Invalid authorization header format")
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			return utils.Unauthorized(c, "Missing token")
		}

		// Validate token
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			return utils.Unauthorized(c, "Invalid or expired token")
		}

		// Store user info in context
		c.Locals(UserIDContextKey, claims.UserID)
		c.Locals(UserRoleContextKey, claims.Role)
		c.Locals(UserContextKey, claims)

		return c.Next()
	}
}

// OptionalAuthenticate is a middleware that validates JWT tokens but doesn't require them
func (m *AuthMiddleware) OptionalAuthenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get(AuthorizationHeader)
		if authHeader == "" {
			return c.Next()
		}

		// Check for Bearer prefix
		if !strings.HasPrefix(authHeader, BearerPrefix) {
			return c.Next()
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			return c.Next()
		}

		// Validate token
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			return c.Next()
		}

		// Store user info in context
		c.Locals(UserIDContextKey, claims.UserID)
		c.Locals(UserRoleContextKey, claims.Role)
		c.Locals(UserContextKey, claims)

		return c.Next()
	}
}

// GetUserID extracts the user ID from the context
func GetUserID(c *fiber.Ctx) (uuid.UUID, bool) {
	userID, ok := c.Locals(UserIDContextKey).(uuid.UUID)
	return userID, ok
}

// GetUserRole extracts the user role from the context
func GetUserRole(c *fiber.Ctx) (model.Role, bool) {
	role, ok := c.Locals(UserRoleContextKey).(model.Role)
	return role, ok
}

// GetClaims extracts the JWT claims from the context
func GetClaims(c *fiber.Ctx) (*service.JWTClaims, bool) {
	claims, ok := c.Locals(UserContextKey).(*service.JWTClaims)
	return claims, ok
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c *fiber.Ctx) bool {
	_, ok := c.Locals(UserIDContextKey).(uuid.UUID)
	return ok
}

// IsAdmin checks if the authenticated user is an admin
func IsAdmin(c *fiber.Ctx) bool {
	role, ok := GetUserRole(c)
	if !ok {
		return false
	}
	return role == model.RoleAdmin
}
