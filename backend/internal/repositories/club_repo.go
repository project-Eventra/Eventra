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

// ClubRepository handles club database operations
type ClubRepository struct {
	pool *pgxpool.Pool
}

// NewClubRepository creates a new club repository
func NewClubRepository(pool *pgxpool.Pool) *ClubRepository {
	return &ClubRepository{pool: pool}
}

// Create inserts a new club
func (r *ClubRepository) Create(ctx context.Context, club *models.Club) error {
	query := `
		INSERT INTO clubs (id, name, description, email, phone, verification_status, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	club.ID = uuid.New()
	club.CreatedAt = time.Now()
	club.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		club.ID, club.Name, club.Description, club.Email, club.Phone,
		club.VerificationStatus, club.IsActive, club.CreatedAt, club.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("club email already registered")
		}
		return fmt.Errorf("error creating club: %w", err)
	}
	return nil
}

// GetByID retrieves a club by ID
func (r *ClubRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Club, error) {
	query := `SELECT id, name, description, email, phone, verification_status, is_active, verified_at, verified_by, rejection_reason, created_at, updated_at
			  FROM clubs WHERE id = $1`

	club := &models.Club{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&club.ID, &club.Name, &club.Description, &club.Email, &club.Phone,
		&club.VerificationStatus, &club.IsActive, &club.VerifiedAt, &club.VerifiedBy,
		&club.RejectionReason, &club.CreatedAt, &club.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting club: %w", err)
	}
	return club, nil
}

// GetByEmail retrieves a club by email
func (r *ClubRepository) GetByEmail(ctx context.Context, email string) (*models.Club, error) {
	query := `SELECT id, name, description, email, phone, verification_status, is_active, verified_at, verified_by, rejection_reason, created_at, updated_at
			  FROM clubs WHERE email = $1`

	club := &models.Club{}
	err := r.pool.QueryRow(ctx, query, strings.ToLower(email)).Scan(
		&club.ID, &club.Name, &club.Description, &club.Email, &club.Phone,
		&club.VerificationStatus, &club.IsActive, &club.VerifiedAt, &club.VerifiedBy,
		&club.RejectionReason, &club.CreatedAt, &club.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting club: %w", err)
	}
	return club, nil
}

// List returns paginated clubs with optional status filter
func (r *ClubRepository) List(ctx context.Context, status string, page, pageSize int) ([]models.Club, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int

	countQuery := `SELECT COUNT(*) FROM clubs WHERE ($1 = '' OR verification_status = $1)`
	err := r.pool.QueryRow(ctx, countQuery, status).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting clubs: %w", err)
	}

	query := `SELECT id, name, description, email, phone, verification_status, is_active, verified_at, verified_by, rejection_reason, created_at, updated_at
			  FROM clubs WHERE ($1 = '' OR verification_status = $1)
			  ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing clubs: %w", err)
	}
	defer rows.Close()

	var clubs []models.Club
	for rows.Next() {
		var c models.Club
		err := rows.Scan(
			&c.ID, &c.Name, &c.Description, &c.Email, &c.Phone,
			&c.VerificationStatus, &c.IsActive, &c.VerifiedAt, &c.VerifiedBy,
			&c.RejectionReason, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning club: %w", err)
		}
		clubs = append(clubs, c)
	}

	return clubs, totalCount, nil
}

// Approve marks a club as approved
func (r *ClubRepository) Approve(ctx context.Context, clubID, approverID uuid.UUID) error {
	query := `UPDATE clubs SET verification_status = 'approved', verified_at = NOW(), verified_by = $1, rejection_reason = NULL, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, approverID, clubID)
	return err
}

// Reject marks a club as rejected
func (r *ClubRepository) Reject(ctx context.Context, clubID uuid.UUID, reason string) error {
	query := `UPDATE clubs SET verification_status = 'rejected', rejection_reason = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, reason, clubID)
	return err
}

// Update modifies club fields
func (r *ClubRepository) Update(ctx context.Context, clubID uuid.UUID, req *models.UpdateClubRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *req.Email)
		argIdx++
	}
	if req.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *req.Phone)
		argIdx++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE clubs SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, clubID)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

// Delete removes a club
func (r *ClubRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE clubs SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// CreateAdminProfile creates a club admin profile
func (r *ClubRepository) CreateAdminProfile(ctx context.Context, profile *models.ClubAdminProfile) error {
	query := `
		INSERT INTO club_admin_profiles (id, user_id, club_id, is_primary, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	profile.ID = uuid.New()
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		profile.ID, profile.UserID, profile.ClubID, profile.IsPrimary,
		profile.CreatedAt, profile.UpdatedAt)
	return err
}

// GetAdminProfile retrieves a club admin profile by user ID
func (r *ClubRepository) GetAdminProfile(ctx context.Context, userID uuid.UUID) (*models.ClubAdminProfile, error) {
	query := `SELECT id, user_id, club_id, is_primary, created_at, updated_at
			  FROM club_admin_profiles WHERE user_id = $1`

	p := &models.ClubAdminProfile{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&p.ID, &p.UserID, &p.ClubID, &p.IsPrimary, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting admin profile: %w", err)
	}
	return p, nil
}
