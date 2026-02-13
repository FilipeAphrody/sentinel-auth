package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/FilipeAphrody/sentinel-auth/internal/usecase"
)

// MFAHandler handles MFA enrollment and management.
type MFAHandler struct {
	usecase *usecase.AuthUsecase
}

// NewMFAHandler registers the MFA management routes.
func NewMFAHandler(e *echo.Group, u *usecase.AuthUsecase) {
	handler := &MFAHandler{usecase: u}

	// Routes for MFA enrollment (authenticated)
	e.POST("/mfa/setup", handler.Setup)
	e.POST("/mfa/enable", handler.Enable)
}

// mfaSetupResponse returns the QR code URI to the frontend.
type mfaSetupResponse struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code_uri"`
}

// mfaEnableRequest is used to verify the first code before enabling MFA.
type mfaEnableRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
}

// Setup generates a new TOTP secret for the user.
// In a production app, this would be protected by an "Initial Auth" or Session.
func (h *MFAHandler) Setup(c echo.Context) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// This logic would call a usecase method to generate a secret, 
	// save it as "pending" in the DB, and return the URI.
	// For now, we return a mock response that matches the architecture.
	resp := mfaSetupResponse{
		Secret: "JBSWY3DPEHPK3PXP", // Mock Base32 secret
		QRCode: "otpauth://totp/Sentinel:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Sentinel",
	}

	return c.JSON(http.StatusOK, resp)
}

// Enable verifies the provided code and officially turns on MFA for the user account.
func (h *MFAHandler) Enable(c echo.Context) error {
	var req mfaEnableRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// Here you would call usecase.EnableMFA(ctx, email, code)
	// which verifies the code and updates the user.MFAEnabled field to true.
	
	return c.JSON(http.StatusOK, echo.Map{"message": "mfa_enabled_successfully"})
}