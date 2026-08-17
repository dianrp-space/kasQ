package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/models"
)

func (r *Repository) CreateUserVerified(ctx context.Context, u *models.User, verified bool) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO users (team_id, name, email, password_hash, role, email_verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		u.TeamID, u.Name, u.Email, u.PasswordHash, u.Role, verified,
	).Scan(&u.ID, &u.CreatedAt)
}

func (r *Repository) SetVerificationToken(ctx context.Context, userID uuid.UUID, token string, expires time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET verification_token=$2, verification_expires=$3 WHERE id=$1`,
		userID, token, expires)
	return err
}

func (r *Repository) GetUserByVerificationToken(ctx context.Context, token string) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, team_id, name, email, password_hash, role, email_verified, avatar_file, created_at
		FROM users
		WHERE verification_token = $1 AND verification_expires > NOW()`, token)
	return scanUser(row)
}

func (r *Repository) VerifyUserEmail(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET email_verified=TRUE, verification_token=NULL, verification_expires=NULL
		WHERE id=$1`, userID)
	return err
}

func (r *Repository) SetResetToken(ctx context.Context, userID uuid.UUID, token string, expires time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET reset_token=$2, reset_expires=$3 WHERE id=$1`,
		userID, token, expires)
	return err
}

func (r *Repository) GetUserByResetToken(ctx context.Context, token string) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, team_id, name, email, password_hash, role, email_verified, avatar_file, created_at
		FROM users
		WHERE reset_token = $1 AND reset_expires > NOW()`, token)
	return scanUser(row)
}

func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET password_hash=$2, reset_token=NULL, reset_expires=NULL
		WHERE id=$1`, userID, passwordHash)
	return err
}

func (r *Repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, email).Scan(&exists)
	return exists, err
}
