package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/params/request"
	"github.com/glennprays/letpai-backend/internal/usecase/contact"
	"github.com/gofiber/fiber/v2"
)

// ContactHandler handles contact requests
type ContactHandler struct {
	createContact  *contact.CreateContactUseCase
	getContacts    *contact.GetContactsUseCase
	getContactByID *contact.GetContactByIDUseCase
	updateContact  *contact.UpdateContactUseCase
	deleteContact  *contact.DeleteContactUseCase
	bulkOperations *contact.BulkOperationsUseCase
}

// NewContactHandler creates a new contact handler
func NewContactHandler(
	createContact *contact.CreateContactUseCase,
	getContacts *contact.GetContactsUseCase,
	getContactByID *contact.GetContactByIDUseCase,
	updateContact *contact.UpdateContactUseCase,
	deleteContact *contact.DeleteContactUseCase,
	bulkOperations *contact.BulkOperationsUseCase,
) *ContactHandler {
	return &ContactHandler{
		createContact:  createContact,
		getContacts:    getContacts,
		getContactByID: getContactByID,
		updateContact:  updateContact,
		deleteContact:  deleteContact,
		bulkOperations: bulkOperations,
	}
}

// Create creates a new contact
// @Summary Create contact
// @Description Create a new contact
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreateContactRequest true "Contact details"
// @Success 201 {object} response.ContactResponse
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 409 {object} httperror.APIError
// @Router /contacts [post]
func (h *ContactHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req request.CreateContactRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &contact.CreateContactRequest{
		Name:           req.Name,
		WhatsAppNumber: req.WhatsAppNumber,
		GroupID:        req.GroupID,
	}

	result, err := h.createContact.Execute(c.Context(), userID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetContacts retrieves contacts with pagination and filters
// @Summary Get contacts
// @Description Get contacts with pagination and filters
// @Tags Contacts
// @Produce json
// @Security BearerAuth
// @Param group_id query string false "Filter by group ID"
// @Param is_favorite query boolean false "Filter by favorite status"
// @Param search query string false "Search in name and WhatsApp number"
// @Param sort_by query string false "Sort by field (name, created_at, group_name, whatsapp_number)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.ContactListResponse
// @Failure 401 {object} httperror.APIError
// @Router /contacts [get]
func (h *ContactHandler) GetContacts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var query request.GetContactsQuery
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

	ucReq := &contact.GetContactsRequest{
		GroupID:    query.GroupID,
		IsFavorite: query.IsFavorite,
		Search:     query.Search,
		SortBy:     query.SortBy,
		SortOrder:  query.SortOrder,
		Page:       query.Page,
		Limit:      query.Limit,
	}

	result, err := h.getContacts.Execute(c.Context(), userID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetByID retrieves a contact by ID
// @Summary Get contact by ID
// @Description Get a contact by ID
// @Tags Contacts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Contact ID"
// @Success 200 {object} response.ContactResponse
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /contacts/{id} [get]
func (h *ContactHandler) GetByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	contactID := c.Params("id")

	result, err := h.getContactByID.Execute(c.Context(), userID, contactID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// Update updates a contact
// @Summary Update contact
// @Description Update a contact
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Contact ID"
// @Param request body request.UpdateContactRequest true "Contact details"
// @Success 200 {object} response.ContactResponse
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /contacts/{id} [put]
func (h *ContactHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	contactID := c.Params("id")

	var req request.UpdateContactRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &contact.UpdateContactRequest{
		Name:           req.Name,
		WhatsAppNumber: req.WhatsAppNumber,
		GroupID:        req.GroupID,
		IsFavorite:     req.IsFavorite,
	}

	result, err := h.updateContact.Execute(c.Context(), userID, contactID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// Delete deletes a contact
// @Summary Delete contact
// @Description Delete a contact
// @Tags Contacts
// @Produce json
// @Security BearerAuth
// @Param id path string true "Contact ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /contacts/{id} [delete]
func (h *ContactHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	contactID := c.Params("id")

	result, err := h.deleteContact.Execute(c.Context(), userID, contactID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// BulkOperations performs bulk operations on contacts
// @Summary Bulk operations on contacts
// @Description Perform bulk operations on contacts (add_to_group, delete)
// @Tags Contacts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.BulkContactsRequest true "Bulk operation details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Router /contacts/bulk [post]
func (h *ContactHandler) BulkOperations(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req request.BulkContactsRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &contact.BulkOperationsRequest{
		Operation:  req.Operation,
		ContactIDs: req.ContactIDs,
		GroupID:    req.GroupID,
	}

	result, err := h.bulkOperations.Execute(c.Context(), userID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
