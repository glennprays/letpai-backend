package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/params/request"
	"github.com/glennprays/letpai-backend/internal/params/response"
	"github.com/glennprays/letpai-backend/internal/usecase/contactgroup"
	"github.com/gofiber/fiber/v2"
)

// ContactGroupHandler handles contact group requests
type ContactGroupHandler struct {
	createGroup *contactgroup.CreateGroupUseCase
	getGroups   *contactgroup.GetGroupsUseCase
	updateGroup *contactgroup.UpdateGroupUseCase
	deleteGroup *contactgroup.DeleteGroupUseCase
}

// NewContactGroupHandler creates a new contact group handler
func NewContactGroupHandler(
	createGroup *contactgroup.CreateGroupUseCase,
	getGroups *contactgroup.GetGroupsUseCase,
	updateGroup *contactgroup.UpdateGroupUseCase,
	deleteGroup *contactgroup.DeleteGroupUseCase,
) *ContactGroupHandler {
	return &ContactGroupHandler{
		createGroup: createGroup,
		getGroups:   getGroups,
		updateGroup: updateGroup,
		deleteGroup: deleteGroup,
	}
}

// Create creates a new contact group
// @Summary Create contact group
// @Description Create a new contact group
// @Tags Contact Groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CreateContactGroupRequest true "Group details"
// @Success 201 {object} response.ContactGroupResponse
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 409 {object} httperror.APIError
// @Router /contact-groups [post]
func (h *ContactGroupHandler) Create(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req request.CreateContactGroupRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &contactgroup.CreateGroupRequest{
		Name:      req.Name,
		Color:     req.Color,
		SortOrder: req.SortOrder,
	}

	result, err := h.createGroup.Execute(c.Context(), userID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": response.ContactGroupResponse{
			GroupID:   result.GroupID.String(),
			Name:      result.Name,
			Color:     result.Color,
			SortOrder: result.SortOrder,
		},
	})
}

// GetGroups retrieves all contact groups
// @Summary Get contact groups
// @Description Get all contact groups for the authenticated user
// @Tags Contact Groups
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} httperror.APIError
// @Router /contact-groups [get]
func (h *ContactGroupHandler) GetGroups(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	result, err := h.getGroups.Execute(c.Context(), userID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	groups := make([]*response.ContactGroupResponse, 0, len(result.Groups))
	for _, g := range result.Groups {
		groups = append(groups, &response.ContactGroupResponse{
			GroupID:   g.GroupID,
			Name:      g.Name,
			Color:     g.Color,
			SortOrder: g.SortOrder,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"groups": groups,
		},
	})
}

// Update updates a contact group
// @Summary Update contact group
// @Description Update a contact group
// @Tags Contact Groups
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Param request body request.UpdateContactGroupRequest true "Group details"
// @Success 200 {object} response.ContactGroupResponse
// @Failure 400 {object} httperror.APIError
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /contact-groups/{id} [put]
func (h *ContactGroupHandler) Update(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	groupID := c.Params("id")

	var req request.UpdateContactGroupRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	ucReq := &contactgroup.UpdateGroupRequest{
		Name:      req.Name,
		Color:     req.Color,
		SortOrder: req.SortOrder,
	}

	result, err := h.updateGroup.Execute(c.Context(), userID, groupID, ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": response.ContactGroupResponse{
			GroupID:   result.GroupID,
			Name:      result.Name,
			Color:     result.Color,
			SortOrder: result.SortOrder,
		},
	})
}

// Delete deletes a contact group
// @Summary Delete contact group
// @Description Delete a contact group
// @Tags Contact Groups
// @Produce json
// @Security BearerAuth
// @Param id path string true "Group ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} httperror.APIError
// @Failure 404 {object} httperror.APIError
// @Router /contact-groups/{id} [delete]
func (h *ContactGroupHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	groupID := c.Params("id")

	result, err := h.deleteGroup.Execute(c.Context(), userID, groupID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}
