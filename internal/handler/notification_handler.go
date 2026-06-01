package handler

import (
	"errors"
	"strconv"

	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/usecase/notification"
	"github.com/gofiber/fiber/v2"
)

// NotificationHandler handles notification requests
type NotificationHandler struct {
	sendNotifications  *notification.SendNotificationsUseCase
	sendReminder       *notification.SendReminderUseCase
	bulkReminder       *notification.BulkReminderUseCase
	reminderStatus     *notification.ReminderStatusUseCase
	retryNotification  *notification.RetryNotificationUseCase
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(
	sendNotifications *notification.SendNotificationsUseCase,
	sendReminder *notification.SendReminderUseCase,
	bulkReminder *notification.BulkReminderUseCase,
	reminderStatus *notification.ReminderStatusUseCase,
	retryNotification *notification.RetryNotificationUseCase,
) *NotificationHandler {
	return &NotificationHandler{
		sendNotifications: sendNotifications,
		sendReminder:      sendReminder,
		bulkReminder:      bulkReminder,
		reminderStatus:    reminderStatus,
		retryNotification: retryNotification,
	}
}

// RetryNotification re-dispatches the most recent notification for a
// participant. Used by the per-participant chip's Retry button.
func (h *NotificationHandler) RetryNotification(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	participantID := c.Params("participant_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("participant_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.retryNotification.Execute(c.Context(), userID, participantID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// ReminderStatus returns whether the host can fire a reminder right now
// and, if not, when the cooldown next opens. Lets the FE render the
// Remind button as disabled + countdown on first paint instead of
// surfacing the 429 only after the user clicks.
func (h *NotificationHandler) ReminderStatus(c *fiber.Ctx) error {
	participantID := c.Params("participant_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("participant_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.reminderStatus.Execute(c.Context(), participantID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// SendNotifications sends WhatsApp notifications to all session participants.
//
// `force=true` is set on the /resend escape-hatch route; everything
// else takes the dirty-gated default path. On a 409 SESSION_NOT_DIRTY
// the response body includes last_notified_at + a stable error code
// so the FE can render its "send again anyway?" confirm modal
// without parsing the message string.
func (h *NotificationHandler) sendNotificationsImpl(c *fiber.Ctx, force bool) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	if sessionID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("session_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.sendNotifications.Execute(c.Context(), userID, sessionID, force)
	if err != nil {
		// Surface SESSION_NOT_DIRTY with the timestamp the FE needs.
		// errors.As digs through domain.Error (it implements Unwrap)
		// to find the embedded NotDirtyError we stamped in the use
		// case.
		var nde *notification.NotDirtyError
		if errors.As(err, &nde) {
			body := fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "SESSION_NOT_DIRTY",
					"message": "Notifications already sent — make a change or use Resend to send again.",
				},
			}
			if !nde.LastNotifiedAt.IsZero() {
				body["last_notified_at"] = nde.LastNotifiedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
			return c.Status(fiber.StatusConflict).JSON(body)
		}
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": result.Message,
		"data":    result,
	})
}

// SendNotifications gates on the dirty predicate.
func (h *NotificationHandler) SendNotifications(c *fiber.Ctx) error {
	return h.sendNotificationsImpl(c, false)
}

// ResendNotifications skips the dirty predicate. Same auth scope.
func (h *NotificationHandler) ResendNotifications(c *fiber.Ctx) error {
	return h.sendNotificationsImpl(c, true)
}

// SendReminder sends reminder to specific participant
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
		// Check for rate limit error and add Retry-After header if available
		if apiErr.Status == 400 && apiErr.Message == "rate limit exceeded" && result != nil && result.RetryAfter > 0 {
			c.Set("Retry-After", strconv.Itoa(int(result.RetryAfter)))
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
