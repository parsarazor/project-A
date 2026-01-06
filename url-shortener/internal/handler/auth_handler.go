package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/service"
	"github.com/url-shortener/internal/utils"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "Registration details"
// @Success 201 {object} utils.Response{data=model.UserResponse}
// @Failure 400 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Failure 422 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	// Register user
	user, err := h.authService.Register(&req)
	if err != nil {
		// Check for specific errors
		if err == service.ErrUserAlreadyExists {
			return utils.Conflict(c, "User with this email already exists")
		}

		// Check for validation errors
		if validationErrors, ok := err.(utils.ValidationErrors); ok {
			return utils.ValidationErrorResponse(c, validationErrors.Error())
		}

		return utils.InternalError(c, "Failed to register user")
	}

	return utils.Created(c, "User registered successfully", user.ToResponse())
}

// Login handles user login
// @Summary Login user
// @Description Authenticate user and return access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "Login credentials"
// @Success 200 {object} utils.Response{data=model.LoginResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 422 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	// Authenticate user
	response, err := h.authService.Login(&req)
	if err != nil {
		// Check for invalid credentials
		if err == service.ErrInvalidCredentials {
			return utils.Unauthorized(c, "Invalid email or password")
		}

		// Check for validation errors
		if validationErrors, ok := err.(utils.ValidationErrors); ok {
			return utils.ValidationErrorResponse(c, validationErrors.Error())
		}

		return utils.InternalError(c, "Failed to login")
	}

	return utils.OK(c, "Login successful", response)
}

// GetMe returns the currently authenticated user's profile
// @Summary Get current user profile
// @Description Get the profile of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=model.UserResponse}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(interface{ String() string })
	if !ok {
		return utils.Unauthorized(c, "User not authenticated")
	}

	// Parse UUID
	claims, ok := c.Locals("user").(*service.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "Invalid user session")
	}

	// Get user from database
	user, err := h.authService.GetUserByID(claims.UserID)
	if err != nil {
		if err == service.ErrUserNotFound {
			return utils.NotFound(c, "User not found")
		}
		return utils.InternalError(c, "Failed to get user profile")
	}

	_ = userID // Suppress unused variable warning

	return utils.OK(c, "", user.ToResponse())
}

// ChangePassword handles password change requests
// @Summary Change password
// @Description Change the password of the currently authenticated user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ChangePasswordRequest true "Password change details"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 422 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	// Get user ID from context
	claims, ok := c.Locals("user").(*service.JWTClaims)
	if !ok {
		return utils.Unauthorized(c, "User not authenticated")
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	// Change password
	err := h.authService.ChangePassword(claims.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return utils.Unauthorized(c, "Current password is incorrect")
		}

		if validationErrors, ok := err.(utils.ValidationErrors); ok {
			return utils.ValidationErrorResponse(c, validationErrors.Error())
		}

		return utils.InternalError(c, "Failed to change password")
	}

	return utils.OK(c, "Password changed successfully", nil)
}

// ChangePasswordRequest represents the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
