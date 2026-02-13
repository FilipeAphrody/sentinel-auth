package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/FilipeAphrody/sentinel-auth/internal/domain"
	"github.com/FilipeAphrody/sentinel-auth/pkg/security"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrMFARequired        = errors.New("mfa_challenge_required")
	ErrInvalidMFACode     = errors.New("invalid mfa code")
)

type AuthUsecase struct {
	userRepo  domain.UserRepository
	tokenRepo domain.TokenRepository
	jwtSecret string
}

func NewAuthUsecase(u domain.UserRepository, t domain.TokenRepository, secret string) *AuthUsecase {
	return &AuthUsecase{
		userRepo:  u,
		tokenRepo: t,
		jwtSecret: secret,
	}
}

// Login handles the first step of authentication: validating credentials.
func (u *AuthUsecase) Login(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 1. Verify Password using Argon2id
	match, err := security.ComparePassword(password, user.PasswordHash)
	if err != nil || !match {
		// Log failed attempt if necessary
		_ = u.userRepo.LogSecurityEvent(ctx, user.ID, "LOGIN_FAILED", "", nil)
		return nil, ErrInvalidCredentials
	}

	// 2. Check if Multi-Factor Authentication is required
	if user.MFAEnabled {
		return nil, ErrMFARequired
	}

	// 3. If no MFA, generate the session immediately
	return u.generateSession(ctx, user)
}

// VerifyMFA handles the second step: validating the TOTP code.
func (u *AuthUsecase) VerifyMFA(ctx context.Context, email, code string) (*domain.AuthResponse, error) {
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Validate TOTP Code
	if !security.VerifyMFACode(code, user.MFASecret) {
		_ = u.userRepo.LogSecurityEvent(ctx, user.ID, "MFA_FAILED", "", nil)
		return nil, ErrInvalidMFACode
	}

	return u.generateSession(ctx, user)
}

// generateSession creates the JWT Access Token and the Opaque Refresh Token.
func (u *AuthUsecase) generateSession(ctx context.Context, user *domain.User) (*domain.AuthResponse, error) {
	// 1. Generate Access Token (JWT) - valid for 15 minutes
	accessToken, err := security.GenerateAccessToken(user.ID, user.Role, u.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	// 2. Generate Refresh Token (Opaque)
	// We use a cryptographically secure random string
	refreshToken, _ := security.GenerateMFASecret() 
	
	// 3. Store Refresh Token in Redis (valid for 24 hours)
	err = u.tokenRepo.StoreRefreshToken(ctx, user.ID, refreshToken, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	// 4. Log successful login
	_ = u.userRepo.LogSecurityEvent(ctx, user.ID, "LOGIN_SUCCESS", "", nil)

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	}, nil
}