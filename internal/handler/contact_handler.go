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
	importContacts *contact.ImportContactsUseCase
}

// NewContactHandler creates a new contact handler
func NewContactHandler(
	createContact *contact.CreateContactUseCase,
	getContacts *contact.GetContactsUseCase,
	getContactByID *contact.GetContactByIDUseCase,
	updateContact *contact.UpdateContactUseCase,
	deleteContact *contact.DeleteContactUseCase,
	bulkOperations *contact.BulkOperationsUseCase,
	importContacts *contact.ImportContactsUseCase,
) *ContactHandler {
	return &ContactHandler{
		createContact:  createContact,
		getContacts:    getContacts,
		getContactByID: getContactByID,
		updateContact:  updateContact,
		deleteContact:  deleteContact,
		bulkOperations: bulkOperations,
		importContacts: importContacts,
	}
}

// Create creates a new contact
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
		AvatarURL:      req.AvatarURL,
		Notes:          req.Notes,
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
		AvatarURL:      req.AvatarURL,
		Notes:          req.Notes,
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

// ImportContacts imports contacts from JSON
func (h *ContactHandler) ImportContacts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req contact.ImportContactsRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.importContacts.Execute(c.Context(), userID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
