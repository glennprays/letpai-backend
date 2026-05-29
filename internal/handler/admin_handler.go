package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/validation"
	adminuc "github.com/glennprays/letpai-backend/internal/usecase/admin"
	"github.com/gofiber/fiber/v2"
)

// AdminHandler handles admin requests
type AdminHandler struct {
	initiateLoginUseCase    *adminuc.InitiateLoginUseCase
	verifyOTPUseCase        *adminuc.VerifyOTPUseCase
	loginUseCase            *adminuc.LoginUseCase
	getProfileUseCase       *adminuc.GetProfileUseCase
	updateProfileUseCase    *adminuc.UpdateProfileUseCase
	setupPasswordUseCase    *adminuc.SetupPasswordUseCase
	listAdminsUseCase       *adminuc.ListAdminsUseCase
	createAdminUseCase      *adminuc.CreateAdminUseCase
	updateAdminUseCase      *adminuc.UpdateAdminUseCase
	deleteAdminUseCase      *adminuc.DeleteAdminUseCase
	getStatusUseCase        *adminuc.GetStatusUseCase
	getQRCodeUseCase        *adminuc.GetQRCodeUseCase
	logoutUseCase           *adminuc.LogoutUseCase
	updateConfigUseCase     *adminuc.UpdateConfigUseCase
	disconnectDeviceUseCase *adminuc.DisconnectDeviceUseCase
	needsSetupUseCase       *adminuc.NeedsSetupUseCase
	bootstrapUseCase        *adminuc.BootstrapUseCase
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	initiateLoginUseCase *adminuc.InitiateLoginUseCase,
	verifyOTPUseCase *adminuc.VerifyOTPUseCase,
	loginUseCase *adminuc.LoginUseCase,
	getProfileUseCase *adminuc.GetProfileUseCase,
	updateProfileUseCase *adminuc.UpdateProfileUseCase,
	setupPasswordUseCase *adminuc.SetupPasswordUseCase,
	listAdminsUseCase *adminuc.ListAdminsUseCase,
	createAdminUseCase *adminuc.CreateAdminUseCase,
	updateAdminUseCase *adminuc.UpdateAdminUseCase,
	deleteAdminUseCase *adminuc.DeleteAdminUseCase,
	getStatusUseCase *adminuc.GetStatusUseCase,
	getQRCodeUseCase *adminuc.GetQRCodeUseCase,
	logoutUseCase *adminuc.LogoutUseCase,
	updateConfigUseCase *adminuc.UpdateConfigUseCase,
	disconnectDeviceUseCase *adminuc.DisconnectDeviceUseCase,
	needsSetupUseCase *adminuc.NeedsSetupUseCase,
	bootstrapUseCase *adminuc.BootstrapUseCase,
) *AdminHandler {
	return &AdminHandler{
		initiateLoginUseCase:    initiateLoginUseCase,
		verifyOTPUseCase:        verifyOTPUseCase,
		loginUseCase:            loginUseCase,
		getProfileUseCase:       getProfileUseCase,
		updateProfileUseCase:    updateProfileUseCase,
		setupPasswordUseCase:    setupPasswordUseCase,
		listAdminsUseCase:       listAdminsUseCase,
		createAdminUseCase:      createAdminUseCase,
		updateAdminUseCase:      updateAdminUseCase,
		deleteAdminUseCase:      deleteAdminUseCase,
		getStatusUseCase:        getStatusUseCase,
		getQRCodeUseCase:        getQRCodeUseCase,
		logoutUseCase:           logoutUseCase,
		updateConfigUseCase:     updateConfigUseCase,
		disconnectDeviceUseCase: disconnectDeviceUseCase,
		needsSetupUseCase:       needsSetupUseCase,
		bootstrapUseCase:        bootstrapUseCase,
	}
}

// NeedsSetup tells the FE whether the first-boot wizard at
// /admin/setup should be reachable. Public, unauthenticated — the
// wizard needs it to decide where to route an unauthenticated user
// landing on /admin/login.
func (h *AdminHandler) NeedsSetup(c *fiber.Ctx) error {
	result, err := h.needsSetupUseCase.Execute(c.Context())
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// Bootstrap claims the first-super-admin slot. Public, unauthenticated;
// the use case + repository reject the request once any real super
// admin already exists, so this endpoint can't be used to escalate
// privileges on a live deployment.
func (h *AdminHandler) Bootstrap(c *fiber.Ctx) error {
	var req adminuc.BootstrapRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.bootstrapUseCase.Execute(c.Context(), &req, true)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// InitiateLogin initiates admin login flow
func (h *AdminHandler) InitiateLogin(c *fiber.Ctx) error {
	var req adminuc.InitiateLoginRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
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
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.loginUseCase.Execute(c.Context(), &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// UpdateProfile lets the calling admin rename themselves. The path
// has no admin_id — adminID is sourced from the bearer token's
// claims via middleware.GetUserID so a caller can only edit their
// own row.
func (h *AdminHandler) UpdateProfile(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)
	if adminID == "" {
		apiErr := httperror.FromError(httperror.ErrUnauthorized("not authenticated"))
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	var req adminuc.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.updateProfileUseCase.Execute(c.Context(), adminID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// DisconnectDevice releases the WhatsApp gateway pairing. Distinct
// from POST /admin/logout, which ends the admin's panel session.
func (h *AdminHandler) DisconnectDevice(c *fiber.Ctx) error {
	result, err := h.disconnectDeviceUseCase.Execute(c.Context())
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	return c.Status(fiber.StatusOK).JSON(result)
}

// VerifyOTP handles OTP verification
func (h *AdminHandler) VerifyOTP(c *fiber.Ctx) error {
	var req adminuc.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
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
	if err := validation.Struct(&req); err != nil {
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
	if err := validation.Struct(&req); err != nil {
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
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	callerID := middleware.GetUserID(c)
	result, err := h.updateAdminUseCase.Execute(c.Context(), callerID, adminID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

// DeleteAdmin deletes an admin (super admin only)
func (h *AdminHandler) DeleteAdmin(c *fiber.Ctx) error {
	adminID := c.Params("id")

	callerID := middleware.GetUserID(c)
	err := h.deleteAdminUseCase.Execute(c.Context(), callerID, adminID)
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
	if err := validation.Struct(&req); err != nil {
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
	if err := validation.Struct(&req); err != nil {
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
