package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/usecase/admintemplates"
	"github.com/glennprays/letpai-backend/internal/validation"
	"github.com/gofiber/fiber/v2"
)

// AdminTemplatesHandler exposes the admin CRUD for message templates.
// All routes mount under /admin/templates and require admin auth.
type AdminTemplatesHandler struct {
	listTemplates  *admintemplates.ListTemplatesUseCase
	updateTemplate *admintemplates.UpdateTemplateUseCase
	testSend       *admintemplates.TestSendUseCase
}

func NewAdminTemplatesHandler(
	listTemplates *admintemplates.ListTemplatesUseCase,
	updateTemplate *admintemplates.UpdateTemplateUseCase,
	testSend *admintemplates.TestSendUseCase,
) *AdminTemplatesHandler {
	return &AdminTemplatesHandler{
		listTemplates:  listTemplates,
		updateTemplate: updateTemplate,
		testSend:       testSend,
	}
}

// List returns every message template. No pagination — the set is tiny.
func (h *AdminTemplatesHandler) List(c *fiber.Ctx) error {
	result, err := h.listTemplates.Execute(c.Context())
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    fiber.Map{"templates": result},
	})
}

// Update edits one template's name/description/body/variables. The
// `:key` path param is the stable identifier (e.g. "session_notification").
func (h *AdminTemplatesHandler) Update(c *fiber.Ctx) error {
	key := c.Params("key")

	var req admintemplates.UpdateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.updateTemplate.Execute(c.Context(), key, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// TestSend renders a template with sample data and sends it to the
// admin-provided phone number so they can preview the message on a
// real WhatsApp device.
func (h *AdminTemplatesHandler) TestSend(c *fiber.Ctx) error {
	key := c.Params("key")

	var req admintemplates.TestSendRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.testSend.Execute(c.Context(), key, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
