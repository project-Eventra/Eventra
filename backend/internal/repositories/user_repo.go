package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/evenrtra/backend/internal/models"
)

// UserRepository handles user database operations
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, firebase_uid, email, password_hash, role, is_active, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		user.ID, user.FirebaseUID, user.Email, user.PasswordHash,
		user.Role, user.IsActive, user.EmailVerified, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "email") {
				return fmt.Errorf("email already registered")
			}
			if strings.Contains(err.Error(), "firebase_uid") {
				return fmt.Errorf("firebase account already linked")
			}
		}
		return fmt.Errorf("error creating user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `SELECT id, firebase_uid, email, password_hash, role, is_active, email_verified, created_at, updated_at
			  FROM users WHERE id = $1`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.FirebaseUID, &user.Email, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, firebase_uid, email, password_hash, role, is_active, email_verified, created_at, updated_at
			  FROM users WHERE email = $1`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, strings.ToLower(email)).Scan(
		&user.ID, &user.FirebaseUID, &user.Email, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}
	return user, nil
}

// GetByFirebaseUID retrieves a user by Firebase UID
func (r *UserRepository) GetByFirebaseUID(ctx context.Context, uid string) (*models.User, error) {
	query := `SELECT id, firebase_uid, email, password_hash, role, is_active, email_verified, created_at, updated_at
			  FROM users WHERE firebase_uid = $1`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, uid).Scan(
		&user.ID, &user.FirebaseUID, &user.Email, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}
	return user, nil
}

// UpdateFirebaseUID updates the firebase_uid for a user
func (r *UserRepository) UpdateFirebaseUID(ctx context.Context, userID uuid.UUID, firebaseUID string) error {
	query := `UPDATE users SET firebase_uid = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, firebaseUID, userID)
	return err
}

// UpdateActive toggles user active status
func (r *UserRepository) UpdateActive(ctx context.Context, userID uuid.UUID, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, isActive, userID)
	return err
}

// Delete removes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// List returns paginated users filtered by role
func (r *UserRepository) List(ctx context.Context, role string, page, pageSize int) ([]models.User, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int

	countQuery := `SELECT COUNT(*) FROM users WHERE ($1 = '' OR role = $1)`
	err := r.pool.QueryRow(ctx, countQuery, role).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting users: %w", err)
	}

	query := `SELECT id, firebase_uid, email, password_hash, role, is_active, email_verified, created_at, updated_at
			  FROM users WHERE ($1 = '' OR role = $1)
			  ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, role, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		err := rows.Scan(&u.ID, &u.FirebaseUID, &u.Email, &u.PasswordHash,
			&u.Role, &u.IsActive, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning user: %w", err)
		}
		users = append(users, u)
	}

	return users, totalCount, nil
}
