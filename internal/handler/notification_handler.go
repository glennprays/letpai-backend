package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/usecase/notification"
	"github.com/gofiber/fiber/v2"
)

// NotificationHandler handles notification requests
type NotificationHandler struct {
	sendNotifications *notification.SendNotificationsUseCase
	sendReminder      *notification.SendReminderUseCase
	bulkReminder      *notification.BulkReminderUseCase
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(
	sendNotifications *notification.SendNotificationsUseCase,
	sendReminder *notification.SendReminderUseCase,
	bulkReminder *notification.BulkReminderUseCase,
) *NotificationHandler {
	return &NotificationHandler{
		sendNotifications: sendNotifications,
		sendReminder:      sendReminder,
		bulkReminder:      bulkReminder,
	}
}

// SendNotifications sends WhatsApp notifications to all session participants
// @Summary Send notifications
// @Description Send WhatsApp notifications to all participants
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} notification.SendNotificationsResponse
// @Failure 400 {object} httperror.APIError
// @Router /sessions/{id}/send-notifications [post]
func (h *NotificationHandler) SendNotifications(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	if sessionID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("session_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.sendNotifications.Execute(c.Context(), userID, sessionID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": result.Message,
		"data":    result,
	})
}

// SendReminder sends reminder to specific participant
// @Summary Send reminder
// @Description Send reminder to specific participant
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param participant_id path string true "Participant ID"
// @Success 200 {object} notification.SendReminderResponse
// @Failure 400 {object} httperror.APIError
// @Failure 429 {object} httperror.APIError
// @Router /participants/{participant_id}/reminder [post]
func (h *NotificationHandler) SendReminder(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	participantID := c.Params("participant_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("participant_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.sendReminder.Execute(c.Context(), userID, participantID)
	if err != nil {
		apiErr := httperror.FromError(err)
		// Check for rate limit error
		if apiErr.Status == 400 && apiErr.Message == "rate limit exceeded" {
			// Add Retry-After header
			if result.RetryAfter > 0 {
				c.Set("Retry-After", string(rune(result.RetryAfter)))
			}
		}
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": result.Message,
		"data":    result,
	})
}

// BulkReminder sends reminders to all unpaid participants
// @Summary Send bulk reminders
// @Description Send reminder to all unpaid participants
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} notification.BulkReminderResponse
// @Failure 400 {object} httperror.APIError
// @Router /sessions/{id}/bulk-reminder [post]
func (h *NotificationHandler) BulkReminder(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	if sessionID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("session_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.bulkReminder.Execute(c.Context(), userID, sessionID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": result.Message,
		"data":    result,
	})
}
