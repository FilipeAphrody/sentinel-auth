package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/FilipeAphrody/sentinel-auth/internal/usecase"
)

// AuthHandler represents the HTTP delivery layer for authentication.
type AuthHandler struct {
	usecase *usecase.AuthUsecase
}

// NewAuthHandler registers the authentication routes to the provided echo group.
func NewAuthHandler(e *echo.Group, u *usecase.AuthUsecase) {
	handler := &AuthHandler{usecase: u}

	e.POST("/login", handler.Login)
	e.POST("/mfa/verify", handler.VerifyMFA)
}

// loginRequest defines the expected JSON payload for the login endpoint.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// mfaRequest defines the expected JSON payload for the MFA verification endpoint.
type mfaRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
}

// Login handles the initial authentication request.
func (h *AuthHandler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	ctx := c.Request().Context()
	resp, err := h.usecase.Login(ctx, req.Email, req.Password)

	if err != nil {
		// Handle the specific MFA required case
		if err == usecase.ErrMFARequired {
			return c.JSON(http.StatusAccepted, echo.Map{
				"message": "mfa_required",
				"email":   req.Email,
			})
		}

		// Handle invalid credentials
		if err == usecase.ErrInvalidCredentials {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
		}

		// Generic internal error
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, resp)
}

// VerifyMFA handles the second step of authentication for users with MFA enabled.
func (h *AuthHandler) VerifyMFA(c echo.Context) error {
	var req mfaRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	ctx := c.Request().Context()
	resp, err := h.usecase.VerifyMFA(ctx, req.Email, req.Code)

	if err != nil {
		if err == usecase.ErrInvalidMFACode || err == usecase.ErrInvalidCredentials {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, resp)
}