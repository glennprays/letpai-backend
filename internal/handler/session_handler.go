package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/params/request"
	"github.com/glennprays/letpai-backend/internal/usecase/billing"
	"github.com/glennprays/letpai-backend/internal/usecase/participant"
	"github.com/glennprays/letpai-backend/internal/usecase/session"
	"github.com/gofiber/fiber/v2"
)

// SessionHandler handles session requests
type SessionHandler struct {
	createSession     *session.CreateSessionUseCase
	getSessions       *session.GetSessionsUseCase
	getSessionDetail  *session.GetSessionDetailUseCase
	updateSession     *session.UpdateSessionUseCase
	cancelSession     *session.CancelSessionUseCase
	addParticipants   *participant.AddParticipantsUseCase
	removeParticipant *participant.RemoveParticipantUseCase
	updateParticipant *participant.UpdateParticipantUseCase
	addBillItem       *billing.AddBillItemUseCase
	calculateSplits   *billing.CalculateSplitsUseCase
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
	calculateSplits *billing.CalculateSplitsUseCase,
) *SessionHandler {
	return &SessionHandler{
		createSession:     createSession,
		getSessions:       getSessions,
		getSessionDetail:  getSessionDetail,
		updateSession:     updateSession,
		cancelSession:     cancelSession,
		addParticipants:   addParticipants,
		removeParticipant: removeParticipant,
		updateParticipant: updateParticipant,
		addBillItem:       addBillItem,
		calculateSplits:   calculateSplits,
	}
}

// Create creates a new session
// @Summary Create session
// @Description Create a new bill splitting session
// @Tags Sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreateSessionRequest true "Session details"
// @Success 201 {object} response.SessionResponse
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Router /sessions [post]
func (h *SessionHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req request.CreateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &session.CreateSessionRequest{
		Title:       req.Title,
		Description: req.Description,
		Currency:    req.Currency,
		SessionDate: req.SessionDate,
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
// @Summary Get sessions
// @Description Get sessions with pagination and filters
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status (active, completed, cancelled)"
// @Param search query string false "Search in title and description"
// @Param sort_by query string false "Sort by field (created_at, session_date, title, total_amount)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.SessionListResponse
// @Failure 401 {object} httperror.APIError
// @Router /sessions [get]
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
// @Summary Get session by ID
// @Description Get a session by ID with participants and bills
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} response.SessionDetailResponse
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id} [get]
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
// @Summary Update session
// @Description Update a session
// @Tags Sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Param request body request.UpdateSessionRequest true "Session details"
// @Success 200 {object} response.SessionResponse
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id} [put]
func (h *SessionHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req request.UpdateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &session.UpdateSessionRequest{
		Title:       req.Title,
		Description: req.Description,
		Currency:    req.Currency,
		SessionDate: req.SessionDate,
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
// @Summary Cancel session
// @Description Cancel a session
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id} [delete]
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
// @Summary Add participants
// @Description Add participants to a session
// @Tags Sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Param request body request.AddParticipantsRequest true "Participants"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id}/participants [post]
func (h *SessionHandler) AddParticipants(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req request.AddParticipantsRequest
	if err := c.BodyParser(&req); err != nil {
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
// @Summary Remove participant
// @Description Remove a participant from a session
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Param participant_id path string true "Participant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id}/participants/{participant_id} [delete]
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
// @Summary Update participant
// @Description Update a custom participant
// @Tags Sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Param participant_id path string true "Participant ID"
// @Param request body request.UpdateParticipantRequest true "Participant details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id}/participants/{participant_id} [put]
func (h *SessionHandler) UpdateParticipant(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")
	participantID := c.Params("participant_id")

	var req request.UpdateParticipantRequest
	if err := c.BodyParser(&req); err != nil {
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
// @Summary Add bill item
// @Description Add a bill item to a session
// @Tags Sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Param request body request.AddBillItemRequest true "Bill item details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id}/bills [post]
func (h *SessionHandler) AddBillItem(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	sessionID := c.Params("id")

	var req request.AddBillItemRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &billing.AddBillItemRequest{
		Description: req.Description,
		Amount:      req.Amount,
		Category:    req.Category,
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
// @Summary Calculate splits
// @Description Calculate equal splits for all participants
// @Tags Sessions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /sessions/{id}/calculate-splits [put]
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
