package domain

import (
	"context"
	"time"
)

// User represents the central identity entity of the system.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`          // Never expose the password hash in JSON
	Role         string    `json:"role"`       // RBAC Role (admin, user, etc.)
	MFAEnabled   bool      `json:"mfa_enabled"`
	MFASecret    string    `json:"-"`          // TOTP secret key
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AuthResponse defines the payload returned after a successful login.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserRepository defines the contract for user data persistence.
// This interface will be implemented in the 'internal/repository' package.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	
	// LogSecurityEvent is used for the Audit Logs requirement
	LogSecurityEvent(ctx context.Context, userID, eventType, ip string, metadata map[string]interface{}) error
}

// TokenRepository defines how we handle opaque refresh tokens (usually in Redis).
type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID string, token string, ttl time.Duration) error
	GetUserIDByRefreshToken(ctx context.Context, token string) (string, error)
	DeleteRefreshToken(ctx context.Context, token string) error
}