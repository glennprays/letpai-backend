package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	adminuc "github.com/glennprays/letpai-backend/internal/usecase/admin"
	"github.com/gofiber/fiber/v2"
)

// AdminHandler handles admin requests
type AdminHandler struct {
	initiateLoginUseCase *adminuc.InitiateLoginUseCase
	verifyOTPUseCase     *adminuc.VerifyOTPUseCase
	getProfileUseCase    *adminuc.GetProfileUseCase
	setupPasswordUseCase *adminuc.SetupPasswordUseCase
	listAdminsUseCase    *adminuc.ListAdminsUseCase
	createAdminUseCase   *adminuc.CreateAdminUseCase
	updateAdminUseCase   *adminuc.UpdateAdminUseCase
	deleteAdminUseCase   *adminuc.DeleteAdminUseCase
	getStatusUseCase     *adminuc.GetStatusUseCase
	getQRCodeUseCase     *adminuc.GetQRCodeUseCase
	logoutUseCase        *adminuc.LogoutUseCase
	updateConfigUseCase  *adminuc.UpdateConfigUseCase
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	initiateLoginUseCase *adminuc.InitiateLoginUseCase,
	verifyOTPUseCase *adminuc.VerifyOTPUseCase,
	getProfileUseCase *adminuc.GetProfileUseCase,
	setupPasswordUseCase *adminuc.SetupPasswordUseCase,
	listAdminsUseCase *adminuc.ListAdminsUseCase,
	createAdminUseCase *adminuc.CreateAdminUseCase,
	updateAdminUseCase *adminuc.UpdateAdminUseCase,
	deleteAdminUseCase *adminuc.DeleteAdminUseCase,
	getStatusUseCase *adminuc.GetStatusUseCase,
	getQRCodeUseCase *adminuc.GetQRCodeUseCase,
	logoutUseCase *adminuc.LogoutUseCase,
	updateConfigUseCase *adminuc.UpdateConfigUseCase,
) *AdminHandler {
	return &AdminHandler{
		initiateLoginUseCase: initiateLoginUseCase,
		verifyOTPUseCase:     verifyOTPUseCase,
		getProfileUseCase:    getProfileUseCase,
		setupPasswordUseCase: setupPasswordUseCase,
		listAdminsUseCase:    listAdminsUseCase,
		createAdminUseCase:   createAdminUseCase,
		updateAdminUseCase:   updateAdminUseCase,
		deleteAdminUseCase:   deleteAdminUseCase,
		getStatusUseCase:     getStatusUseCase,
		getQRCodeUseCase:     getQRCodeUseCase,
		logoutUseCase:        logoutUseCase,
		updateConfigUseCase:  updateConfigUseCase,
	}
}

// InitiateLogin initiates admin login flow
func (h *AdminHandler) InitiateLogin(c *fiber.Ctx) error {
	var req adminuc.InitiateLoginRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.initiateLoginUseCase.Execute(c.Context(), &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// Login handles admin login with OTP
func (h *AdminHandler) Login(c *fiber.Ctx) error {
	var req adminuc.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// TODO: Implement password-based login
	// For now, this is essentially a no-op as login is done via OTP
	return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
		"success": false,
		"error":   "Password login not implemented yet. Please use OTP login.",
	})
}

// VerifyOTP handles OTP verification
func (h *AdminHandler) VerifyOTP(c *fiber.Ctx) error {
	var req adminuc.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.verifyOTPUseCase.Execute(c.Context(), &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// GetProfile retrieves admin profile
func (h *AdminHandler) GetProfile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	result, err := h.getProfileUseCase.Execute(c.Context(), userID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// SetupPassword sets initial password for admin
func (h *AdminHandler) SetupPassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req adminuc.SetupPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	err := h.setupPasswordUseCase.Execute(c.Context(), userID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}

// ListAdmins lists all admins (super admin only)
func (h *AdminHandler) ListAdmins(c *fiber.Ctx) error {
	result, err := h.listAdminsUseCase.Execute(c.Context())
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// CreateAdmin creates a new admin (super admin only)
func (h *AdminHandler) CreateAdmin(c *fiber.Ctx) error {
	var req adminuc.CreateAdminRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.createAdminUseCase.Execute(c.Context(), &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

// UpdateAdmin updates admin details
func (h *AdminHandler) UpdateAdmin(c *fiber.Ctx) error {
	adminID := c.Params("id")

	var req adminuc.UpdateAdminRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.updateAdminUseCase.Execute(c.Context(), adminID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// DeleteAdmin deletes an admin (super admin only)
func (h *AdminHandler) DeleteAdmin(c *fiber.Ctx) error {
	adminID := c.Params("id")

	err := h.deleteAdminUseCase.Execute(c.Context(), adminID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"admin_id": adminID,
	})
}

// GetStatus retrieves WhatsApp gateway status
func (h *AdminHandler) GetStatus(c *fiber.Ctx) error {
	result, err := h.getStatusUseCase.Execute(c.Context())
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// GetQRCode generates QR code for WhatsApp pairing
func (h *AdminHandler) GetQRCode(c *fiber.Ctx) error {
	var req adminuc.GetQRCodeRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.getQRCodeUseCase.Execute(c.Context(), &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// Logout handles admin logout
func (h *AdminHandler) Logout(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	err := h.logoutUseCase.Execute(c.Context(), userID)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}

// UpdateConfig updates WhatsApp API configuration
func (h *AdminHandler) UpdateConfig(c *fiber.Ctx) error {
	var req adminuc.UpdateConfigRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	err := h.updateConfigUseCase.Execute(c.Context(), &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}
