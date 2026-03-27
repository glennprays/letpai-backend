package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/usecase/payment"
	"github.com/gofiber/fiber/v2"
)

// PaymentHandler handles payment requests
type PaymentHandler struct {
	submitPayment  *payment.SubmitPaymentUseCase
	approvePayment *payment.ApprovePaymentUseCase
	rejectPayment  *payment.RejectPaymentUseCase
	bulkApprove    *payment.BulkApproveUseCase
	bulkReject     *payment.BulkRejectUseCase
	getPaymentPage *payment.GetPaymentPageUseCase
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(
	submitPayment *payment.SubmitPaymentUseCase,
	approvePayment *payment.ApprovePaymentUseCase,
	rejectPayment *payment.RejectPaymentUseCase,
	bulkApprove *payment.BulkApproveUseCase,
	bulkReject *payment.BulkRejectUseCase,
	getPaymentPage *payment.GetPaymentPageUseCase,
) *PaymentHandler {
	return &PaymentHandler{
		submitPayment:  submitPayment,
		approvePayment: approvePayment,
		rejectPayment:  rejectPayment,
		bulkApprove:    bulkApprove,
		bulkReject:     bulkReject,
		getPaymentPage: getPaymentPage,
	}
}

// SubmitPayment handles payment proof submission (public endpoint, no auth required)
func (h *PaymentHandler) SubmitPayment(c *fiber.Ctx) error {
	participantID := c.Params("participant_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("participant_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	var req payment.SubmitPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.submitPayment.Execute(c.Context(), participantID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Payment proof submitted",
		"data":    result,
	})
}

// ApprovePayment handles payment approval (host only)
func (h *PaymentHandler) ApprovePayment(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	participantID := c.Params("proof_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("proof_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	var req payment.ApprovePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		// Default to empty request if body is empty
		req = payment.ApprovePaymentRequest{}
	}

	// Execute use case
	result, err := h.approvePayment.Execute(c.Context(), userID, participantID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Payment approved",
		"data":    result,
	})
}

// RejectPayment handles payment rejection (host only)
func (h *PaymentHandler) RejectPayment(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	participantID := c.Params("proof_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("proof_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	var req payment.RejectPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.rejectPayment.Execute(c.Context(), userID, participantID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Payment rejected",
		"data":    result,
	})
}

// BulkApprove handles bulk payment approval
func (h *PaymentHandler) BulkApprove(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req payment.BulkApproveRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.bulkApprove.Execute(c.Context(), userID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Payments processed",
		"data":    result,
	})
}

// BulkReject handles bulk payment rejection
func (h *PaymentHandler) BulkReject(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req payment.BulkRejectRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.bulkReject.Execute(c.Context(), userID, &req)
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

// GetPaymentPage handles getting payment page data (public endpoint, no auth required)
func (h *PaymentHandler) GetPaymentPage(c *fiber.Ctx) error {
	participantID := c.Params("participant_id")
	if participantID == "" {
		apiErr := httperror.FromError(httperror.ErrBadRequest("participant_id is required"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Execute use case
	result, err := h.getPaymentPage.Execute(c.Context(), participantID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
