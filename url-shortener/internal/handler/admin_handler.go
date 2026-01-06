package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/url-shortener/internal/service"
	"github.com/url-shortener/internal/utils"
)

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	urlService  *service.URLService
	authService *service.AuthService
}

// NewAdminHandler creates a new AdminHandler instance
func NewAdminHandler(urlService *service.URLService, authService *service.AuthService) *AdminHandler {
	return &AdminHandler{
		urlService:  urlService,
		authService: authService,
	}
}

// GetAllURLs returns all URLs in the system (admin only)
// @Summary Get all URLs
// @Description Retrieves all shortened URLs in the system (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/admin/urls [get]
func (h *AdminHandler) GetAllURLs(c *fiber.Ctx) error {
	// Get pagination parameters
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 20)

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Get all URLs with user info
	urls, total, err := h.urlService.GetAllURLs(page, pageSize)
	if err != nil {
		return utils.InternalError(c, "Failed to retrieve URLs")
	}

	// Convert to response format
	urlResponses := h.urlService.ToResponses(urls)

	return utils.OK(c, "", fiber.Map{
		"urls":      urlResponses,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// DeleteURL deletes a URL by ID (admin only)
// @Summary Delete a URL
// @Description Deletes a shortened URL by its ID (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/admin/urls/{id} [delete]
func (h *AdminHandler) DeleteURL(c *fiber.Ctx) error {
	// Parse URL ID from path
	idParam := c.Params("id")
	if idParam == "" {
		return utils.BadRequest(c, "URL ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID format")
	}

	// Delete URL (admin has permission to delete any URL)
	if err := h.urlService.DeleteURLByAdmin(id); err != nil {
		if err == service.ErrURLNotFound {
			return utils.NotFound(c, "URL not found")
		}
		return utils.InternalError(c, "Failed to delete URL")
	}

	return utils.OK(c, "URL deleted successfully", nil)
}

// GetURLByID retrieves a specific URL by ID (admin only)
// @Summary Get URL by ID
// @Description Retrieves a specific URL by its ID (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/admin/urls/{id} [get]
func (h *AdminHandler) GetURLByID(c *fiber.Ctx) error {
	// Parse URL ID from path
	idParam := c.Params("id")
	if idParam == "" {
		return utils.BadRequest(c, "URL ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID format")
	}

	// Get URL
	url, err := h.urlService.GetURLByID(id)
	if err != nil {
		if err == service.ErrURLNotFound {
			return utils.NotFound(c, "URL not found")
		}
		return utils.InternalError(c, "Failed to retrieve URL")
	}

	return utils.OK(c, "", h.urlService.ToResponse(url))
}

// DeactivateURL deactivates a URL without deleting it (admin only)
// @Summary Deactivate a URL
// @Description Deactivates a URL so it no longer redirects (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/admin/urls/{id}/deactivate [patch]
func (h *AdminHandler) DeactivateURL(c *fiber.Ctx) error {
	// Parse URL ID from path
	idParam := c.Params("id")
	if idParam == "" {
		return utils.BadRequest(c, "URL ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID format")
	}

	// Deactivate URL (admin can deactivate any URL)
	if err := h.urlService.DeactivateURL(id, uuid.Nil, true); err != nil {
		if err == service.ErrURLNotFound {
			return utils.NotFound(c, "URL not found")
		}
		return utils.InternalError(c, "Failed to deactivate URL")
	}

	return utils.OK(c, "URL deactivated successfully", nil)
}

// ActivateURL activates a previously deactivated URL (admin only)
// @Summary Activate a URL
// @Description Reactivates a previously deactivated URL (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "URL ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/v1/admin/urls/{id}/activate [patch]
func (h *AdminHandler) ActivateURL(c *fiber.Ctx) error {
	// Parse URL ID from path
	idParam := c.Params("id")
	if idParam == "" {
		return utils.BadRequest(c, "URL ID is required")
	}

	id, err := uuid.Parse(idParam)
	if err != nil {
		return utils.BadRequest(c, "Invalid URL ID format")
	}

	// Activate URL (admin can activate any URL)
	if err := h.urlService.ActivateURL(id, uuid.Nil, true); err != nil {
		if err == service.ErrURLNotFound {
			return utils.NotFound(c, "URL not found")
		}
		return utils.InternalError(c, "Failed to activate URL")
	}

	return utils.OK(c, "URL activated successfully", nil)
}
