package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/evenrtra/backend/internal/models"
)

// RegistrationRepository handles registration database operations
type RegistrationRepository struct {
	pool *pgxpool.Pool
}

// NewRegistrationRepository creates a new registration repository
func NewRegistrationRepository(pool *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{pool: pool}
}

// CreateIndividual creates an individual registration with capacity check inside a transaction
func (r *RegistrationRepository) CreateIndividual(ctx context.Context, reg *models.Registration) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the event row to prevent race conditions
	var maxParticipants *int
	var deadline time.Time
	var status string
	var participationType string
	err = tx.QueryRow(ctx,
		`SELECT max_participants, registration_deadline, status, participation_type FROM events WHERE id = $1 FOR UPDATE`,
		reg.EventID).Scan(&maxParticipants, &deadline, &status, &participationType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("event not found")
		}
		return fmt.Errorf("error locking event: %w", err)
	}

	// Check event status
	if status != "active" {
		return fmt.Errorf("event is not accepting registrations")
	}

	// Check deadline
	if time.Now().After(deadline) {
		return fmt.Errorf("registration deadline has passed")
	}

	// Check participation type
	if participationType == "group" {
		return fmt.Errorf("this event only accepts group registrations")
	}

	// Check for duplicate registration
	var existing int
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM registrations WHERE event_id = $1 AND user_id = $2 AND registration_type = 'individual' AND status != 'cancelled'`,
		reg.EventID, reg.UserID).Scan(&existing)
	if err != nil {
		return fmt.Errorf("error checking existing registration: %w", err)
	}
	if existing > 0 {
		return fmt.Errorf("you have already registered for this event")
	}

	// Check capacity
	if maxParticipants != nil {
		var currentCount int
		err = tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM registrations WHERE event_id = $1 AND status != 'cancelled'`,
			reg.EventID).Scan(&currentCount)
		if err != nil {
			return fmt.Errorf("error checking capacity: %w", err)
		}
		if currentCount >= *maxParticipants {
			return fmt.Errorf("event has reached maximum capacity")
		}
	}

	// Insert registration
	reg.ID = uuid.New()
	reg.RegistrationType = "individual"
	reg.Status = "confirmed"
	reg.CreatedAt = time.Now()
	reg.UpdatedAt = time.Now()

	_, err = tx.Exec(ctx,
		`INSERT INTO registrations (id, event_id, user_id, registration_type, status, custom_fields_data, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		reg.ID, reg.EventID, reg.UserID, reg.RegistrationType, reg.Status,
		reg.CustomFieldsData, reg.CreatedAt, reg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating registration: %w", err)
	}

	return tx.Commit(ctx)
}

// CreateGroup creates a group registration with all members in a transaction
func (r *RegistrationRepository) CreateGroup(ctx context.Context, group *models.RegistrationGroup, members []models.GroupMember, leaderReg *models.Registration) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock event row
	var maxParticipants *int
	var deadline time.Time
	var status, participationType string
	var minGroupSize, maxGroupSize int
	err = tx.QueryRow(ctx,
		`SELECT max_participants, registration_deadline, status, participation_type, min_group_size, max_group_size FROM events WHERE id = $1 FOR UPDATE`,
		group.EventID).Scan(&maxParticipants, &deadline, &status, &participationType, &minGroupSize, &maxGroupSize)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("event not found")
		}
		return fmt.Errorf("error locking event: %w", err)
	}

	if status != "active" {
		return fmt.Errorf("event is not accepting registrations")
	}
	if time.Now().After(deadline) {
		return fmt.Errorf("registration deadline has passed")
	}
	if participationType == "individual" {
		return fmt.Errorf("this event only accepts individual registrations")
	}

	// Validate group size
	memberCount := len(members)
	if memberCount < minGroupSize {
		return fmt.Errorf("group must have at least %d members", minGroupSize)
	}
	if memberCount > maxGroupSize {
		return fmt.Errorf("group must have at most %d members", maxGroupSize)
	}

	// Check for duplicate group by leader
	var existingGroup int
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM registration_groups WHERE event_id = $1 AND leader_user_id = $2 AND status != 'cancelled'`,
		group.EventID, group.LeaderUserID).Scan(&existingGroup)
	if err != nil {
		return fmt.Errorf("error checking existing group: %w", err)
	}
	if existingGroup > 0 {
		return fmt.Errorf("you have already created a group for this event")
	}

	// Check capacity (count groups)
	if maxParticipants != nil {
		var currentCount int
		err = tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM registration_groups WHERE event_id = $1 AND status != 'cancelled'`,
			group.EventID).Scan(&currentCount)
		if err != nil {
			return fmt.Errorf("error checking capacity: %w", err)
		}
		if currentCount >= *maxParticipants {
			return fmt.Errorf("event has reached maximum team capacity")
		}
	}

	// Insert group
	group.ID = uuid.New()
	group.Status = "confirmed"
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	_, err = tx.Exec(ctx,
		`INSERT INTO registration_groups (id, event_id, group_name, leader_user_id, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		group.ID, group.EventID, group.GroupName, group.LeaderUserID,
		group.Status, group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating group: %w", err)
	}

	// Insert leader registration
	leaderReg.ID = uuid.New()
	leaderReg.EventID = group.EventID
	leaderReg.RegistrationType = "group"
	leaderReg.GroupID = &group.ID
	leaderReg.Status = "confirmed"
	leaderReg.CreatedAt = time.Now()
	leaderReg.UpdatedAt = time.Now()

	_, err = tx.Exec(ctx,
		`INSERT INTO registrations (id, event_id, user_id, registration_type, group_id, status, custom_fields_data, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		leaderReg.ID, leaderReg.EventID, leaderReg.UserID, leaderReg.RegistrationType,
		leaderReg.GroupID, leaderReg.Status, leaderReg.CustomFieldsData,
		leaderReg.CreatedAt, leaderReg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating leader registration: %w", err)
	}

	// Insert members
	for i := range members {
		members[i].ID = uuid.New()
		members[i].GroupID = group.ID
		members[i].CreatedAt = time.Now()

		var customDataJSON *string
		if members[i].CustomFieldsData != nil {
			customDataJSON = members[i].CustomFieldsData
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO group_members (id, group_id, member_name, member_email, member_phone, member_registration_number, user_id, custom_fields_data, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			members[i].ID, members[i].GroupID, members[i].MemberName,
			members[i].MemberEmail, members[i].MemberPhone, members[i].MemberRegistrationNumber,
			members[i].UserID, customDataJSON, members[i].CreatedAt)
		if err != nil {
			return fmt.Errorf("error adding group member: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetByID retrieves a registration by ID
func (r *RegistrationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Registration, error) {
	query := `
		SELECT r.id, r.event_id, r.user_id, r.registration_type, r.group_id, r.status, r.custom_fields_data, r.created_at, r.updated_at,
			e.name as event_name, c.name as club_name, e.event_date, e.registration_deadline,
			COALESCE(e.venue, 'Campus') as venue,
			COALESCE(NULLIF(TRIM(sp.first_name || ' ' || sp.last_name), ''), u.email) as student_name,
			COALESCE(sp.registration_number, '') as student_reg_no,
			u.email as student_email,
			COALESCE(rg.group_name, '') as team_name,
			COALESCE(sp.department, '') as department,
			COALESCE(sp.phone, '') as phone
		FROM registrations r
		JOIN events e ON e.id = r.event_id
		JOIN clubs c ON c.id = e.club_id
		JOIN users u ON u.id = r.user_id
		LEFT JOIN student_profiles sp ON sp.user_id = r.user_id
		LEFT JOIN registration_groups rg ON rg.id = r.group_id
		WHERE r.id = $1`

	reg := &models.Registration{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&reg.ID, &reg.EventID, &reg.UserID, &reg.RegistrationType, &reg.GroupID,
		&reg.Status, &reg.CustomFieldsData, &reg.CreatedAt, &reg.UpdatedAt,
		&reg.EventName, &reg.ClubName, &reg.EventDate, &reg.Deadline,
		&reg.Venue, &reg.StudentName, &reg.StudentRegNo, &reg.StudentEmail,
		&reg.TeamName, &reg.Department, &reg.Phone)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting registration: %w", err)
	}
	reg.StudentID = reg.StudentRegNo
	reg.Email = reg.StudentEmail
	return reg, nil
}

// GetByUserID retrieves all registrations for a user
func (r *RegistrationRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Registration, error) {
	query := `
		SELECT r.id, r.event_id, r.user_id, r.registration_type, r.group_id, r.status, r.custom_fields_data, r.created_at, r.updated_at,
			e.name as event_name, c.name as club_name, e.event_date, e.registration_deadline,
			COALESCE(e.venue, 'Campus') as venue,
			COALESCE(NULLIF(TRIM(sp.first_name || ' ' || sp.last_name), ''), u.email) as student_name,
			COALESCE(sp.registration_number, '') as student_reg_no,
			u.email as student_email,
			COALESCE(rg.group_name, '') as team_name
		FROM registrations r
		JOIN events e ON e.id = r.event_id
		JOIN clubs c ON c.id = e.club_id
		JOIN users u ON u.id = r.user_id
		LEFT JOIN student_profiles sp ON sp.user_id = r.user_id
		LEFT JOIN registration_groups rg ON rg.id = r.group_id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting registrations: %w", err)
	}
	defer rows.Close()

	var regs []models.Registration
	for rows.Next() {
		var reg models.Registration
		err := rows.Scan(
			&reg.ID, &reg.EventID, &reg.UserID, &reg.RegistrationType, &reg.GroupID,
			&reg.Status, &reg.CustomFieldsData, &reg.CreatedAt, &reg.UpdatedAt,
			&reg.EventName, &reg.ClubName, &reg.EventDate, &reg.Deadline,
			&reg.Venue, &reg.StudentName, &reg.StudentRegNo, &reg.StudentEmail,
			&reg.TeamName)
		if err != nil {
			return nil, fmt.Errorf("error scanning registration: %w", err)
		}
		reg.StudentID = reg.StudentRegNo
		reg.Email = reg.StudentEmail
		regs = append(regs, reg)
	}

	return regs, nil
}

// GetByEventID retrieves registrations for an event with student info
func (r *RegistrationRepository) GetByEventID(ctx context.Context, eventID uuid.UUID, page, pageSize int) ([]models.Registration, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int

	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM registrations WHERE event_id = $1 AND status != 'cancelled'`,
		eventID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting registrations: %w", err)
	}

	query := `
		SELECT r.id, r.event_id, r.user_id, r.registration_type, r.group_id, r.status, r.custom_fields_data, r.created_at, r.updated_at,
			COALESCE(NULLIF(TRIM(sp.first_name || ' ' || sp.last_name), ''), u.email) as student_name,
			COALESCE(sp.registration_number, '') as student_reg_no,
			u.email as student_email,
			COALESCE(rg.group_name, '') as team_name,
			COALESCE(sp.department, '') as department,
			COALESCE(sp.phone, '') as phone
		FROM registrations r
		JOIN users u ON u.id = r.user_id
		LEFT JOIN student_profiles sp ON sp.user_id = r.user_id
		LEFT JOIN registration_groups rg ON rg.id = r.group_id
		WHERE r.event_id = $1 AND r.status != 'cancelled'
		ORDER BY r.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, eventID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting event registrations: %w", err)
	}
	defer rows.Close()

	var regs []models.Registration
	for rows.Next() {
		var reg models.Registration
		err := rows.Scan(
			&reg.ID, &reg.EventID, &reg.UserID, &reg.RegistrationType, &reg.GroupID,
			&reg.Status, &reg.CustomFieldsData, &reg.CreatedAt, &reg.UpdatedAt,
			&reg.StudentName, &reg.StudentRegNo, &reg.StudentEmail,
			&reg.TeamName, &reg.Department, &reg.Phone)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning registration: %w", err)
		}
		reg.StudentID = reg.StudentRegNo
		reg.Email = reg.StudentEmail
		regs = append(regs, reg)
	}

	return regs, totalCount, nil
}

// GetGroupsByEventID retrieves group registrations for an event
func (r *RegistrationRepository) GetGroupsByEventID(ctx context.Context, eventID uuid.UUID) ([]models.RegistrationGroup, error) {
	query := `
		SELECT rg.id, rg.event_id, rg.group_name, rg.leader_user_id, rg.status, rg.created_at, rg.updated_at,
			COALESCE(sp.first_name || ' ' || sp.last_name, u.email) as leader_name
		FROM registration_groups rg
		JOIN users u ON u.id = rg.leader_user_id
		LEFT JOIN student_profiles sp ON sp.user_id = rg.leader_user_id
		WHERE rg.event_id = $1 AND rg.status != 'cancelled'
		ORDER BY rg.created_at DESC`

	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("error getting groups: %w", err)
	}
	defer rows.Close()

	var groups []models.RegistrationGroup
	for rows.Next() {
		var g models.RegistrationGroup
		err := rows.Scan(&g.ID, &g.EventID, &g.GroupName, &g.LeaderUserID, &g.Status,
			&g.CreatedAt, &g.UpdatedAt, &g.LeaderName)
		if err != nil {
			return nil, fmt.Errorf("error scanning group: %w", err)
		}

		// Get members
		memberRows, err := r.pool.Query(ctx,
			`SELECT id, group_id, member_name, member_email, member_phone, member_registration_number, user_id, custom_fields_data, created_at
			 FROM group_members WHERE group_id = $1`,
			g.ID)
		if err != nil {
			return nil, fmt.Errorf("error getting group members: %w", err)
		}

		for memberRows.Next() {
			var m models.GroupMember
			err := memberRows.Scan(&m.ID, &m.GroupID, &m.MemberName, &m.MemberEmail,
				&m.MemberPhone, &m.MemberRegistrationNumber, &m.UserID,
				&m.CustomFieldsData, &m.CreatedAt)
			if err != nil {
				memberRows.Close()
				return nil, fmt.Errorf("error scanning member: %w", err)
			}
			g.Members = append(g.Members, m)
		}
		memberRows.Close()

		groups = append(groups, g)
	}

	return groups, nil
}

// CheckUserRegistered checks if a user is already registered for an event
func (r *RegistrationRepository) CheckUserRegistered(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM registrations WHERE event_id = $1 AND user_id = $2 AND status != 'cancelled'`,
		eventID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// marshalJSON is a helper to convert map to JSON string
func marshalJSON(data map[string]interface{}) *string {
	if data == nil || len(data) == 0 {
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}
