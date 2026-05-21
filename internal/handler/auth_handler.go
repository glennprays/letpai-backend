package handler

import (
	"time"

	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/params/request"
	"github.com/glennprays/letpai-backend/internal/params/response"
	"github.com/glennprays/letpai-backend/internal/validation"
	"github.com/glennprays/letpai-backend/internal/usecase/auth"
	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	registerUser   *auth.RegisterUserUseCase
	verifyOTP      *auth.VerifyOTPUseCase
	loginUser      *auth.LoginUserUseCase
	logoutUser     *auth.LogoutUserUseCase
	updateProfile  *auth.UpdateProfileUseCase
	forgotPassword *auth.ForgotPasswordUseCase
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	registerUser *auth.RegisterUserUseCase,
	verifyOTP *auth.VerifyOTPUseCase,
	loginUser *auth.LoginUserUseCase,
	logoutUser *auth.LogoutUserUseCase,
	updateProfile *auth.UpdateProfileUseCase,
	forgotPassword *auth.ForgotPasswordUseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUser:   registerUser,
		verifyOTP:      verifyOTP,
		loginUser:      loginUser,
		logoutUser:     logoutUser,
		updateProfile:  updateProfile,
		forgotPassword: forgotPassword,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req request.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Convert domain request
	ucReq := &auth.RegisterUserRequest{
		WhatsAppNumber: req.WhatsAppNumber,
		Password:       req.Password,
	}

	// Execute use case
	result, err := h.registerUser.Execute(c.Context(), ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Parse expiry time
	expiresAt, _ := time.Parse(time.RFC3339, result.ExpiresAt)

	return c.Status(fiber.StatusCreated).JSON(response.RegisterResponse{
		Success:   true,
		Message:   "OTP sent to your WhatsApp",
		UserID:    result.UserID,
		ExpiresAt: expiresAt,
	})
}

// VerifyOTP handles OTP verification
func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req request.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Convert domain request
	ucReq := &auth.VerifyOTPRequest{
		WhatsAppNumber: req.WhatsAppNumber,
		OTPCode:        req.OTPCode,
	}

	// Execute use case
	result, err := h.verifyOTP.Execute(c.Context(), ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(response.VerifyOTPResponse{
		Success: true,
		Message: "Registration successful",
		Token:   result.Token,
		User: &response.User{
			UserID:         result.User.UserID,
			WhatsAppNumber: result.User.WhatsAppNumber,
			FullName:       result.User.FullName,
		},
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req request.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Convert domain request
	ucReq := &auth.LoginRequest{
		WhatsAppNumber: req.WhatsAppNumber,
		Password:       req.Password,
	}

	// Execute use case
	result, err := h.loginUser.Execute(c.Context(), ucReq)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(response.LoginResponse{
		Success: true,
		Token:   result.Token,
		User: &response.User{
			UserID:         result.User.UserID,
			WhatsAppNumber: result.User.WhatsAppNumber,
			FullName:       result.User.FullName,
		},
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Execute use case
	result, err := h.logoutUser.Execute(c.Context(), &auth.LogoutRequest{})
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(response.LogoutResponse{
		Success: true,
		Message: result.Message,
	})
}

// ForgotPassword starts a password-reset flow by sending a one-time code
// via WhatsApp. The response is intentionally identical whether the supplied
// number is registered or not, to avoid leaking account existence.
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req request.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.forgotPassword.Execute(c.Context(), &auth.ForgotPasswordRequest{
		WhatsAppNumber: req.WhatsAppNumber,
	})
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.Status(fiber.StatusOK).JSON(response.ForgotPasswordResponse{
		Success:   true,
		Message:   result.Message,
		ExpiresAt: result.ExpiresAt,
	})
}

// UpdateProfile handles user profile update
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req auth.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}
	if err := validation.Struct(&req); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	result, err := h.updateProfile.Execute(c.Context(), userID, &req)
	if err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	return c.JSON(result)
}
