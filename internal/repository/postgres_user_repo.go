package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/FilipeAphrody/sentinel-auth/internal/domain"
)

// PostgresUserRepo implements domain.UserRepository using PostgreSQL.
type PostgresUserRepo struct {
	db *sql.DB
}

// NewPostgresUserRepo creates a new repository instance.
func NewPostgresUserRepo(db *sql.DB) *PostgresUserRepo {
	return &PostgresUserRepo{db: db}
}

// GetByEmail retrieves a user by their email address, joining with the roles table.
func (r *PostgresUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// We join with 'roles' to get the role name directly, avoiding N+1 queries.
	query := `
		SELECT u.id, u.email, u.password_hash, r.name, u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.MFAEnabled,
		&user.MFASecret,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by their UUID.
func (r *PostgresUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, r.name, u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.MFAEnabled,
		&user.MFASecret,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	return user, nil
}

// Create inserts a new user into the database.
func (r *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	// 1. Resolve Role Name to ID
	var roleID string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM roles WHERE name = $1", user.Role).Scan(&roleID)
	if err != nil {
		return fmt.Errorf("role '%s' not found: %w", user.Role, err)
	}

	// 2. Insert User
	query := `
		INSERT INTO users (email, password_hash, role_id, mfa_enabled, mfa_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	// Use COALESCE logic in Go: if secret is empty, send NULL to DB if configured, 
	// or empty string depending on DB constraint. Here we send string.
	var mfaSecret sql.NullString
	if user.MFASecret != "" {
		mfaSecret.String = user.MFASecret
		mfaSecret.Valid = true
	}

	err = r.db.QueryRowContext(ctx, query,
		user.Email,
		user.PasswordHash,
		roleID,
		user.MFAEnabled,
		mfaSecret,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update modifies an existing user's MFA status and secret.
func (r *PostgresUserRepo) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users 
		SET mfa_enabled = $1, mfa_secret = $2, updated_at = $3
		WHERE id = $4
	`
	
	user.UpdatedAt = time.Now()
	
	var mfaSecret sql.NullString
	if user.MFASecret != "" {
		mfaSecret.String = user.MFASecret
		mfaSecret.Valid = true
	}

	result, err := r.db.ExecContext(ctx, query, user.MFAEnabled, mfaSecret, user.UpdatedAt, user.ID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("user not found")
	}

	return nil
}

// LogSecurityEvent inserts an immutable record into the audit_logs table.
func (r *PostgresUserRepo) LogSecurityEvent(ctx context.Context, userID, eventType, ip string, metadata map[string]interface{}) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (user_id, event_type, ip_address, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	// Handle case where userID is empty (e.g. anonymous failed login)
	// The schema allows user_id to be NULL.
	var uid sql.NullString
	if userID != "" {
		uid.String = userID
		uid.Valid = true
	}

	_, err = r.db.ExecContext(ctx, query, uid, eventType, ip, metaJSON, time.Now())
	return err
}