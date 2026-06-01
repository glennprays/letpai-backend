package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/validation"
	"github.com/glennprays/letpai-backend/internal/params/request"
	"github.com/glennprays/letpai-backend/internal/usecase/billing"
	"github.com/glennprays/letpai-backend/internal/usecase/participant"
	"github.com/glennprays/letpai-backend/internal/usecase/session"
	"github.com/gofiber/fiber/v2"
)

// SessionHandler handles session requests
type SessionHandler struct {
	createSession         *session.CreateSessionUseCase
	getSessions           *session.GetSessionsUseCase
	getSessionDetail      *session.GetSessionDetailUseCase
	updateSession         *session.UpdateSessionUseCase
	cancelSession         *session.CancelSessionUseCase
	addParticipants       *participant.AddParticipantsUseCase
	removeParticipant     *participant.RemoveParticipantUseCase
	updateParticipant     *participant.UpdateParticipantUseCase
	addBillItem           *billing.AddBillItemUseCase
	updateBillItem        *billing.UpdateBillItemUseCase
	deleteBillItem        *billing.DeleteBillItemUseCase
	calculateSplits       *billing.CalculateSplitsUseCase
	replaceBankAccounts   *session.ReplaceBankAccountsUseCase
	uploadBillImage       *billing.UploadBillImageUseCase
	getBillImages         *billing.GetBillImagesUseCase
	getBillImageSignedUrl *billing.GetBillImageSignedUrlUseCase
	deleteBillImage       *billing.DeleteBillImageUseCase
	updateFeeConfig       *billing.UpdateFeeConfigUseCase
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(
	createSession *session.CreateSessionUseCase,
	getSessions *session.GetSessionsUseCase,
	getSessionDetail *session.GetSessionDetailUseCase,
	updateSession *session.UpdateSessionUseCase,
	cancelSession *session.CancelSessionUseCase,
	addParticipants *participant.AddParticipantsUseCase,
	removeParticipant *participant.RemoveParticipantUseCase,
	updateParticipant *participant.UpdateParticipantUseCase,
	addBillItem *billing.AddBillItemUseCase,
	updateBillItem *billing.UpdateBillItemUseCase,
	deleteBillItem *billing.DeleteBillItemUseCase,
	calculateSplits *billing.CalculateSplitsUseCase,
	replaceBankAccounts *session.ReplaceBankAccountsUseCase,
	uploadBillImage *billing.UploadBillImageUseCase,
	getBillImages *billing.GetBillImagesUseCase,
	getBillImageSignedUrl *billing.GetBillImageSignedUrlUseCase,
	deleteBillImage *billing.DeleteBillImageUseCase,
	updateFeeConfig *billing.UpdateFeeConfigUseCase,
) *SessionHandler {
	return &SessionHandler{
		createSession:         createSession,
		getSessions:           getSessions,
		getSessionDetail:      getSessionDetail,
		updateSession:         updateSession,
		cancelSession:         cancelSession,
		addParticipants:       addParticipants,
		removeParticipant:     removeParticipant,
		updateParticipant:     updateParticipant,
		addBillItem:           addBillItem,
		updateBillItem:        updateBillItem,
		deleteBillItem:        deleteBillItem,
		calculateSplits:       calculateSplits,
		replaceBankAccounts:   replaceBankAccounts,
		uploadBillImage:       uploadBillImage,
		getBillImages:         getBillImages,
		getBillImageSignedUrl: getBillImageSignedUrl,
		deleteBillImage:       deleteBillImage,
		updateFeeConfig:       updateFeeConfig,
	}
}

// ReplaceBankAccounts swaps the host's transfer destinations.
func (h *SessionHandler) ReplaceBankAccounts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req session.ReplaceBankAccountsRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.replaceBankAccounts.Execute(c.Context(), userID, sessionID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// Create creates a new session
func (h *SessionHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req request.CreateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &session.CreateSessionRequest{
		Title:             req.Title,
		Description:       req.Description,
		Currency:          req.Currency,
		SessionDate:       req.SessionDate,
		BankName:          req.BankName,
		BankAccountNumber: req.BankAccountNumber,
		BankAccountHolder: req.BankAccountHolder,
	}

	result, err := h.createSession.Execute(c.Context(), userID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetSessions retrieves sessions with pagination and filters
func (h *SessionHandler) GetSessions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var query request.GetSessionsQuery
	if err := c.QueryParser(&query); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Set defaults
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.SortBy == "" {
		query.SortBy = "created_at"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}

	ucReq := &session.GetSessionsRequest{
		Status:    query.Status,
		Search:    query.Search,
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
		Page:      query.Page,
		Limit:     query.Limit,
	}

	result, err := h.getSessions.Execute(c.Context(), userID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetByID retrieves a session by ID with full details
func (h *SessionHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	result, err := h.getSessionDetail.Execute(c.Context(), userID, sessionID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// Update updates a session
func (h *SessionHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req request.UpdateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &session.UpdateSessionRequest{
		Title:             req.Title,
		Description:       req.Description,
		Currency:          req.Currency,
		SessionDate:       req.SessionDate,
		BankName:          req.BankName,
		BankAccountNumber: req.BankAccountNumber,
		BankAccountHolder: req.BankAccountHolder,
	}

	result, err := h.updateSession.Execute(c.Context(), userID, sessionID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// Delete cancels a session
func (h *SessionHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	result, err := h.cancelSession.Execute(c.Context(), userID, sessionID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// AddParticipants adds participants to a session
func (h *SessionHandler) AddParticipants(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req request.AddParticipantsRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Convert request types
	participants := make([]participant.AddParticipantRequest, 0, len(req.Participants))
	for _, p := range req.Participants {
		participants = append(participants, participant.AddParticipantRequest{
			ContactID:      p.ContactID,
			CustomName:     p.CustomName,
			CustomWhatsApp: p.CustomWhatsApp,
		})
	}

	ucReq := &participant.AddParticipantsRequest{
		Participants: participants,
	}

	result, err := h.addParticipants.Execute(c.Context(), userID, sessionID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// RemoveParticipant removes a participant from a session
func (h *SessionHandler) RemoveParticipant(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	participantID := c.Params("participant_id")

	result, err := h.removeParticipant.Execute(c.Context(), userID, sessionID, participantID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// UpdateParticipant updates a participant
func (h *SessionHandler) UpdateParticipant(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	participantID := c.Params("participant_id")

	var req request.UpdateParticipantRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &participant.UpdateParticipantRequest{
		CustomName:     req.CustomName,
		CustomWhatsApp: req.CustomWhatsApp,
	}

	result, err := h.updateParticipant.Execute(c.Context(), userID, sessionID, participantID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// AddBillItem adds a bill item to a session
func (h *SessionHandler) AddBillItem(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req request.AddBillItemRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &billing.AddBillItemRequest{
		Description:    req.Description,
		Amount:         req.Amount,
		Category:       req.Category,
		ParticipantIDs: req.ParticipantIDs,
		IncludesServiceCharge: req.IncludesServiceCharge,
		IncludesTax:           req.IncludesTax,
	}

	result, err := h.addBillItem.Execute(c.Context(), userID, sessionID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// CalculateSplits calculates equal splits for all participants
func (h *SessionHandler) CalculateSplits(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	result, err := h.calculateSplits.Execute(c.Context(), userID, sessionID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// UpdateBillItem updates a bill item
func (h *SessionHandler) UpdateBillItem(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	billItemID := c.Params("bill_item_id")

	var req request.UpdateBillItemRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	billingReq := &billing.UpdateBillItemRequest{
		Description:           req.Description,
		Amount:                req.Amount,
		Category:              req.Category,
		ParticipantIDs:        req.ParticipantIDs,
		IncludesServiceCharge: req.IncludesServiceCharge,
		IncludesTax:           req.IncludesTax,
	}
	result, err := h.updateBillItem.Execute(c.Context(), userID, sessionID, billItemID, billingReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// DeleteBillItem deletes a bill item
func (h *SessionHandler) DeleteBillItem(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	billItemID := c.Params("bill_item_id")

	result, err := h.deleteBillItem.Execute(c.Context(), userID, sessionID, billItemID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// UploadBillImage uploads a bill/receipt image to a session.
func (h *SessionHandler) UploadBillImage(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req billing.UploadBillImageRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.uploadBillImage.Execute(c.Context(), userID, sessionID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetBillImages lists all bill images for a session.
func (h *SessionHandler) GetBillImages(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	result, err := h.getBillImages.Execute(c.Context(), userID, sessionID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetBillImageSignedUrl generates a presigned GET URL for a bill image.
func (h *SessionHandler) GetBillImageSignedUrl(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	imageID := c.Params("image_id")

	result, err := h.getBillImageSignedUrl.Execute(c.Context(), userID, sessionID, imageID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// DeleteBillImage deletes a bill image from a session.
func (h *SessionHandler) DeleteBillImage(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	imageID := c.Params("image_id")

	result, err := h.deleteBillImage.Execute(c.Context(), userID, sessionID, imageID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// UpdateFeeConfig updates the fee configuration for a session.
func (h *SessionHandler) UpdateFeeConfig(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req billing.UpdateFeeConfigRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.updateFeeConfig.Execute(c.Context(), userID, sessionID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
