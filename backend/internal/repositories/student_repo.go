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

// StudentRepository handles student profile database operations
type StudentRepository struct {
	pool *pgxpool.Pool
}

// NewStudentRepository creates a new student repository
func NewStudentRepository(pool *pgxpool.Pool) *StudentRepository {
	return &StudentRepository{pool: pool}
}

// Create inserts a new student profile
func (r *StudentRepository) Create(ctx context.Context, profile *models.StudentProfile) error {
	query := `
		INSERT INTO student_profiles (id, user_id, first_name, last_name, registration_number, department, academic_session, phone, phone_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	profile.ID = uuid.New()
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		profile.ID, profile.UserID, profile.FirstName, profile.LastName,
		profile.RegistrationNumber, profile.Department, profile.AcademicSession,
		profile.Phone, profile.PhoneVerified, profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "registration_number") {
				return fmt.Errorf("registration number already exists")
			}
			if strings.Contains(err.Error(), "user_id") {
				return fmt.Errorf("student profile already exists for this user")
			}
		}
		return fmt.Errorf("error creating student profile: %w", err)
	}
	return nil
}

// GetByUserID retrieves a student profile by user ID
func (r *StudentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.StudentProfile, error) {
	query := `SELECT id, user_id, first_name, last_name, registration_number, department, academic_session, phone, phone_verified, created_at, updated_at
			  FROM student_profiles WHERE user_id = $1`

	sp := &models.StudentProfile{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&sp.ID, &sp.UserID, &sp.FirstName, &sp.LastName,
		&sp.RegistrationNumber, &sp.Department, &sp.AcademicSession,
		&sp.Phone, &sp.PhoneVerified, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting student profile: %w", err)
	}
	return sp, nil
}

// GetByRegistrationNumber retrieves a student profile by registration number
func (r *StudentRepository) GetByRegistrationNumber(ctx context.Context, regNo string) (*models.StudentProfile, error) {
	query := `SELECT id, user_id, first_name, last_name, registration_number, department, academic_session, phone, phone_verified, created_at, updated_at
			  FROM student_profiles WHERE registration_number = $1`

	sp := &models.StudentProfile{}
	err := r.pool.QueryRow(ctx, query, regNo).Scan(
		&sp.ID, &sp.UserID, &sp.FirstName, &sp.LastName,
		&sp.RegistrationNumber, &sp.Department, &sp.AcademicSession,
		&sp.Phone, &sp.PhoneVerified, &sp.CreatedAt, &sp.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting student profile: %w", err)
	}
	return sp, nil
}

// Update modifies a student profile
func (r *StudentRepository) Update(ctx context.Context, userID uuid.UUID, req *models.UpdateStudentRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argIdx))
		args = append(args, *req.FirstName)
		argIdx++
	}
	if req.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argIdx))
		args = append(args, *req.LastName)
		argIdx++
	}
	if req.Department != nil {
		setClauses = append(setClauses, fmt.Sprintf("department = $%d", argIdx))
		args = append(args, *req.Department)
		argIdx++
	}
	if req.AcademicSession != nil {
		setClauses = append(setClauses, fmt.Sprintf("academic_session = $%d", argIdx))
		args = append(args, *req.AcademicSession)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE student_profiles SET %s WHERE user_id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, userID)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

// ListStudents returns paginated student profiles with user info
func (r *StudentRepository) ListStudents(ctx context.Context, search string, page, pageSize int) ([]models.StudentProfile, []models.User, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int

	searchPattern := "%" + strings.ToLower(search) + "%"

	countQuery := `SELECT COUNT(*) FROM student_profiles sp
		JOIN users u ON u.id = sp.user_id
		WHERE ($1 = '%%' OR LOWER(sp.first_name) LIKE $1 OR LOWER(sp.last_name) LIKE $1 OR LOWER(sp.registration_number) LIKE $1 OR LOWER(u.email) LIKE $1)`
	err := r.pool.QueryRow(ctx, countQuery, searchPattern).Scan(&totalCount)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("error counting students: %w", err)
	}

	query := `SELECT sp.id, sp.user_id, sp.first_name, sp.last_name, sp.registration_number, sp.department, sp.academic_session, sp.phone, sp.phone_verified, sp.created_at, sp.updated_at,
			  u.id, u.email, u.role, u.is_active, u.email_verified, u.created_at
		FROM student_profiles sp
		JOIN users u ON u.id = sp.user_id
		WHERE ($1 = '%%' OR LOWER(sp.first_name) LIKE $1 OR LOWER(sp.last_name) LIKE $1 OR LOWER(sp.registration_number) LIKE $1 OR LOWER(u.email) LIKE $1)
		ORDER BY sp.created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, searchPattern, pageSize, offset)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("error listing students: %w", err)
	}
	defer rows.Close()

	var profiles []models.StudentProfile
	var users []models.User
	for rows.Next() {
		var sp models.StudentProfile
		var u models.User
		err := rows.Scan(
			&sp.ID, &sp.UserID, &sp.FirstName, &sp.LastName, &sp.RegistrationNumber,
			&sp.Department, &sp.AcademicSession, &sp.Phone, &sp.PhoneVerified,
			&sp.CreatedAt, &sp.UpdatedAt,
			&u.ID, &u.Email, &u.Role, &u.IsActive, &u.EmailVerified, &u.CreatedAt)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("error scanning student: %w", err)
		}
		profiles = append(profiles, sp)
		users = append(users, u)
	}

	return profiles, users, totalCount, nil
}
