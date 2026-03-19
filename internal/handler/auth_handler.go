package handler

import (
	"time"

	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/params/request"
	"github.com/glennprays/letpai-backend/internal/params/response"
	"github.com/glennprays/letpai-backend/internal/usecase/auth"
	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	registerUser *auth.RegisterUserUseCase
	verifyOTP    *auth.VerifyOTPUseCase
	loginUser    *auth.LoginUserUseCase
	logoutUser   *auth.LogoutUserUseCase
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	registerUser *auth.RegisterUserUseCase,
	verifyOTP *auth.VerifyOTPUseCase,
	loginUser *auth.LoginUserUseCase,
	logoutUser *auth.LogoutUserUseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUser: registerUser,
		verifyOTP:    verifyOTP,
		loginUser:    loginUser,
		logoutUser:   logoutUser,
	}
}

// Register handles user registration
// @Summary Register new user
// @Description Register new user with OTP verification
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body request.RegisterRequest true "Registration details"
// @Success 201 {object} response.RegisterResponse
// @Failure 400 {object} httperror.APIError
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req request.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
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
// @Summary Verify OTP
// @Description Verify OTP and complete registration
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body request.VerifyOTPRequest true "OTP verification details"
// @Success 200 {object} response.VerifyOTPResponse
// @Failure 400 {object} httperror.APIError
// @Router /auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req request.VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
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
// @Summary Login
// @Description Login with WhatsApp number and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "Login details"
// @Success 200 {object} response.LoginResponse
// @Failure 401 {object} httperror.APIError
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req request.LoginRequest
	if err := c.BodyParser(&req); err != nil {
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
// @Summary Logout
// @Description Logout and invalidate token
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.LogoutResponse
// @Router /auth/logout [post]
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
