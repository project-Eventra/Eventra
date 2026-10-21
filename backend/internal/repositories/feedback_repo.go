package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/evenrtra/backend/internal/models"
)

// FeedbackRepository handles feedback database operations
type FeedbackRepository struct {
	pool *pgxpool.Pool
}

// NewFeedbackRepository creates a new feedback repository
func NewFeedbackRepository(pool *pgxpool.Pool) *FeedbackRepository {
	return &FeedbackRepository{pool: pool}
}

// Create inserts feedback
func (r *FeedbackRepository) Create(ctx context.Context, fb *models.Feedback) error {
	query := `
		INSERT INTO feedback (id, user_id, name, email, category, subject, message, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	fb.ID = uuid.New()
	fb.Status = "new"
	fb.CreatedAt = time.Now()
	fb.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		fb.ID, fb.UserID, fb.Name, fb.Email, fb.Category, fb.Subject,
		fb.Message, fb.Status, fb.CreatedAt, fb.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating feedback: %w", err)
	}
	return nil
}

// List returns paginated feedback
func (r *FeedbackRepository) List(ctx context.Context, status string, page, pageSize int) ([]models.Feedback, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int

	countQuery := `SELECT COUNT(*) FROM feedback WHERE ($1 = '' OR status = $1)`
	err := r.pool.QueryRow(ctx, countQuery, status).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting feedback: %w", err)
	}

	query := `SELECT id, user_id, name, email, category, subject, message, status, admin_notes, created_at, updated_at
			  FROM feedback WHERE ($1 = '' OR status = $1)
			  ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing feedback: %w", err)
	}
	defer rows.Close()

	var feedbacks []models.Feedback
	for rows.Next() {
		var f models.Feedback
		err := rows.Scan(&f.ID, &f.UserID, &f.Name, &f.Email, &f.Category, &f.Subject,
			&f.Message, &f.Status, &f.AdminNotes, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning feedback: %w", err)
		}
		feedbacks = append(feedbacks, f)
	}

	return feedbacks, totalCount, nil
}

// Update modifies feedback status/notes
func (r *FeedbackRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateFeedbackRequest) error {
	if req.Status != nil {
		_, err := r.pool.Exec(ctx, `UPDATE feedback SET status = $1, updated_at = NOW() WHERE id = $2`, *req.Status, id)
		if err != nil {
			return err
		}
	}
	if req.AdminNotes != nil {
		_, err := r.pool.Exec(ctx, `UPDATE feedback SET admin_notes = $1, updated_at = NOW() WHERE id = $2`, *req.AdminNotes, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetByID retrieves feedback by ID
func (r *FeedbackRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Feedback, error) {
	query := `SELECT id, user_id, name, email, category, subject, message, status, admin_notes, created_at, updated_at
			  FROM feedback WHERE id = $1`

	f := &models.Feedback{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.UserID, &f.Name, &f.Email, &f.Category, &f.Subject,
		&f.Message, &f.Status, &f.AdminNotes, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting feedback: %w", err)
	}
	return f, nil
}

// AuditLogRepository handles audit log operations
type AuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(pool *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{pool: pool}
}

// Create inserts an audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *models.AuditLog) error {
	query := `INSERT INTO audit_logs (id, actor_id, action, entity_type, entity_id, details, ip_address, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	log.ID = uuid.New()
	log.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		log.ID, log.ActorID, log.Action, log.EntityType, log.EntityID,
		log.Details, log.IPAddress, log.CreatedAt)
	return err
}

// DashboardStats returns platform statistics
func GetDashboardStats(ctx context.Context, pool *pgxpool.Pool) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	queries := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM users WHERE role = 'student'`, &stats.TotalStudents},
		{`SELECT COUNT(*) FROM clubs WHERE verification_status = 'approved'`, &stats.TotalClubs},
		{`SELECT COUNT(*) FROM events WHERE status = 'active'`, &stats.TotalEvents},
		{`SELECT COUNT(*) FROM registrations WHERE status != 'cancelled'`, &stats.TotalRegistrations},
		{`SELECT COUNT(*) FROM clubs WHERE verification_status = 'pending'`, &stats.PendingClubs},
		{`SELECT COUNT(*) FROM events WHERE status = 'active'`, &stats.ActiveEvents},
		{`SELECT COUNT(*) FROM feedback WHERE status = 'new'`, &stats.NewFeedback},
	}

	for _, q := range queries {
		err := pool.QueryRow(ctx, q.query).Scan(q.dest)
		if err != nil {
			return nil, fmt.Errorf("error getting stats: %w", err)
		}
	}

	return stats, nil
}
