package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/url-shortener/internal/middleware"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/service"
	"github.com/url-shortener/internal/utils"
)

// URLHandler handles HTTP requests for URL operations
type URLHandler struct {
	urlService *service.URLService
}

// NewURLHandler creates a new URLHandler instance
func NewURLHandler(urlService *service.URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

// CreateURL handles POST /api/v1/urls
// Creates a new shortened URL for the authenticated user
func (h *URLHandler) CreateURL(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Parse request body
	var req model.CreateURLRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	// Validate URL
	validator := utils.NewValidator()
	validator.ValidateURL("original_url", req.OriginalURL)
	if validator.HasErrors() {
		return utils.ValidationErrorResponse(c, validator.Errors().Error())
	}

	// Create shortened URL
	url, err := h.urlService.CreateShortURL(userID, req.OriginalURL)
	if err != nil {
		if err == service.ErrInvalidURL {
			return utils.ValidationErrorResponse(c, "Invalid URL format")
		}
		return utils.InternalError(c, "Failed to create short URL")
	}

	// Return response
	return utils.Created(c, "URL shortened successfully", h.urlService.ToResponse(url))
}

// GetMyURLs handles GET /api/v1/urls/me
// Returns all URLs belonging to the authenticated user
func (h *URLHandler) GetMyURLs(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Get user's URLs
	urls, err := h.urlService.GetUserURLs(userID)
	if err != nil {
		return utils.InternalError(c, "Failed to retrieve URLs")
	}

	// Convert to response format
	responses := h.urlService.ToResponses(urls)

	return utils.SuccessWithData(c, fiber.StatusOK, responses)
}

// GetUserStats handles GET /api/v1/urls/stats
// Returns statistics for the authenticated user's URLs
func (h *URLHandler) GetUserStats(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Get stats
	stats, err := h.urlService.GetUserStats(userID)
	if err != nil {
		return utils.InternalError(c, "Failed to retrieve statistics")
	}

	return utils.SuccessWithData(c, fiber.StatusOK, stats)
}

// Redirect handles GET /{shortCode}
// Redirects to the original URL and increments click count
func (h *URLHandler) Redirect(c *fiber.Ctx) error {
	shortCode := c.Params("shortCode")
	if shortCode == "" {
		return utils.BadRequest(c, "Short code is required")
	}

	// Validate short code format
	if !utils.IsValidShortCode(shortCode) {
		return utils.NotFound(c, "URL not found")
	}

	// Get URL by short code
	url, err := h.urlService.GetURLByShortCode(shortCode)
	if err != nil {
		if err == service.ErrURLNotFound {
			return utils.NotFound(c, "URL not found")
		}
		return utils.InternalError(c, "Failed to retrieve URL")
	}

	// Increment click count asynchronously (Zero Network Overhead approach)
	// This runs in a goroutine so the redirect happens immediately
	h.urlService.IncrementClickCountAsync(shortCode)

	// Redirect to original URL (301 Moved Permanently)
	return c.Redirect(url.OriginalURL, fiber.StatusMovedPermanently)
}

// DeleteURL handles DELETE /api/v1/urls/{id}
// Deletes a URL belonging to the authenticated user
func (h *URLHandler) DeleteURL(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Parse URL ID
	urlIDParam := c.Params("id")
	urlID, err := uuid.Parse(urlIDParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID")
	}

	// Check if user is admin
	isAdmin := middleware.IsAdmin(c)

	// Delete URL
	err = h.urlService.DeleteURL(urlID, userID, isAdmin)
	if err != nil {
		switch err {
		case service.ErrURLNotFound:
			return utils.NotFound(c, "URL not found")
		case service.ErrUnauthorized:
			return utils.Forbidden(c, "You don't have permission to delete this URL")
		default:
			return utils.InternalError(c, "Failed to delete URL")
		}
	}

	return utils.OK(c, "URL deleted successfully", nil)
}

// DeactivateURL handles PATCH /api/v1/urls/{id}/deactivate
// Deactivates a URL without deleting it
func (h *URLHandler) DeactivateURL(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Parse URL ID
	urlIDParam := c.Params("id")
	urlID, err := uuid.Parse(urlIDParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID")
	}

	// Check if user is admin
	isAdmin := middleware.IsAdmin(c)

	// Deactivate URL
	err = h.urlService.DeactivateURL(urlID, userID, isAdmin)
	if err != nil {
		switch err {
		case service.ErrURLNotFound:
			return utils.NotFound(c, "URL not found")
		case service.ErrUnauthorized:
			return utils.Forbidden(c, "You don't have permission to deactivate this URL")
		default:
			return utils.InternalError(c, "Failed to deactivate URL")
		}
	}

	return utils.OK(c, "URL deactivated successfully", nil)
}

// ActivateURL handles PATCH /api/v1/urls/{id}/activate
// Reactivates a previously deactivated URL
func (h *URLHandler) ActivateURL(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Parse URL ID
	urlIDParam := c.Params("id")
	urlID, err := uuid.Parse(urlIDParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID")
	}

	// Check if user is admin
	isAdmin := middleware.IsAdmin(c)

	// Activate URL
	err = h.urlService.ActivateURL(urlID, userID, isAdmin)
	if err != nil {
		switch err {
		case service.ErrURLNotFound:
			return utils.NotFound(c, "URL not found")
		case service.ErrUnauthorized:
			return utils.Forbidden(c, "You don't have permission to activate this URL")
		default:
			return utils.InternalError(c, "Failed to activate URL")
		}
	}

	return utils.OK(c, "URL activated successfully", nil)
}

// GetURLByID handles GET /api/v1/urls/{id}
// Returns details for a specific URL
func (h *URLHandler) GetURLByID(c *fiber.Ctx) error {
	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return utils.Unauthorized(c, "Authentication required")
	}

	// Parse URL ID
	urlIDParam := c.Params("id")
	urlID, err := uuid.Parse(urlIDParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID")
	}

	// Get URL
	url, err := h.urlService.GetURLByID(urlID)
	if err != nil {
		if err == service.ErrURLNotFound {
			return utils.NotFound(c, "URL not found")
		}
		return utils.InternalError(c, "Failed to retrieve URL")
	}

	// Check if user owns this URL or is admin
	isAdmin := middleware.IsAdmin(c)
	if url.UserID != userID && !isAdmin {
		return utils.Forbidden(c, "You don't have permission to view this URL")
	}

	return utils.SuccessWithData(c, fiber.StatusOK, h.urlService.ToResponse(url))
}
