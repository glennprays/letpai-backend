package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/usecase/dashboard"
	"github.com/gofiber/fiber/v2"
)

// DashboardHandler handles dashboard requests
type DashboardHandler struct {
	getDashboard *dashboard.GetDashboardUseCase
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(getDashboard *dashboard.GetDashboardUseCase) *DashboardHandler {
	return &DashboardHandler{
		getDashboard: getDashboard,
	}
}

// GetDashboard retrieves dashboard statistics
func (h *DashboardHandler) GetDashboard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	result, err := h.getDashboard.Execute(c.Context(), userID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(result)
}
