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

// EventRepository handles event database operations
type EventRepository struct {
	pool *pgxpool.Pool
}

// NewEventRepository creates a new event repository
func NewEventRepository(pool *pgxpool.Pool) *EventRepository {
	return &EventRepository{pool: pool}
}

// Create inserts a new event
func (r *EventRepository) Create(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (id, club_id, name, short_description, description, event_date, registration_deadline,
			venue, participation_type, min_group_size, max_group_size, max_participants,
			registration_fee, has_fee, fee_instructions, rules, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	event.ID = uuid.New()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		event.ID, event.ClubID, event.Name, event.ShortDescription, event.Description,
		event.EventDate, event.RegistrationDeadline, event.Venue,
		event.ParticipationType, event.MinGroupSize, event.MaxGroupSize, event.MaxParticipants,
		event.RegistrationFee, event.HasFee, event.FeeInstructions, event.Rules,
		event.Status, event.CreatedAt, event.UpdatedAt)
	if err != nil {
		return fmt.Errorf("error creating event: %w", err)
	}
	return nil
}

// GetByID retrieves an event with club name and registration count
func (r *EventRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Event, error) {
	query := `
		SELECT e.id, e.club_id, e.name, e.short_description, e.description, e.event_date,
			e.registration_deadline, e.venue, e.participation_type, e.min_group_size, e.max_group_size,
			e.max_participants, e.registration_fee, e.has_fee, e.fee_instructions, e.rules,
			e.status, e.created_at, e.updated_at,
			c.name as club_name,
			COALESCE((SELECT COUNT(*) FROM registrations reg WHERE reg.event_id = e.id AND reg.status != 'cancelled'), 0) as registration_count,
			COALESCE((SELECT COUNT(*) FROM registration_groups rg WHERE rg.event_id = e.id AND rg.status != 'cancelled'), 0) as group_count
		FROM events e
		JOIN clubs c ON c.id = e.club_id
		WHERE e.id = $1`

	event := &models.Event{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID, &event.ClubID, &event.Name, &event.ShortDescription, &event.Description,
		&event.EventDate, &event.RegistrationDeadline, &event.Venue,
		&event.ParticipationType, &event.MinGroupSize, &event.MaxGroupSize,
		&event.MaxParticipants, &event.RegistrationFee, &event.HasFee, &event.FeeInstructions,
		&event.Rules, &event.Status, &event.CreatedAt, &event.UpdatedAt,
		&event.ClubName, &event.RegistrationCount, &event.GroupCount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting event: %w", err)
	}
	return event, nil
}

// List returns paginated events with optional filters
func (r *EventRepository) List(ctx context.Context, clubID *uuid.UUID, status, search string, page, pageSize int) ([]models.Event, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int
	searchPattern := "%" + strings.ToLower(search) + "%"

	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if clubID != nil {
		conditions = append(conditions, fmt.Sprintf("e.club_id = $%d", argIdx))
		args = append(args, *clubID)
		argIdx++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(e.name) LIKE $%d OR LOWER(c.name) LIKE $%d)", argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events e JOIN clubs c ON c.id = e.club_id WHERE %s", whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting events: %w", err)
	}

	listArgs := make([]interface{}, len(args))
	copy(listArgs, args)
	listArgs = append(listArgs, pageSize, offset)

	query := fmt.Sprintf(`
		SELECT e.id, e.club_id, e.name, e.short_description, e.description, e.event_date,
			e.registration_deadline, e.venue, e.participation_type, e.min_group_size, e.max_group_size,
			e.max_participants, e.registration_fee, e.has_fee, e.fee_instructions, e.rules,
			e.status, e.created_at, e.updated_at,
			c.name as club_name,
			COALESCE((SELECT COUNT(*) FROM registrations reg WHERE reg.event_id = e.id AND reg.status != 'cancelled'), 0) as registration_count,
			COALESCE((SELECT COUNT(*) FROM registration_groups rg WHERE rg.event_id = e.id AND rg.status != 'cancelled'), 0) as group_count
		FROM events e
		JOIN clubs c ON c.id = e.club_id
		WHERE %s
		ORDER BY e.event_date ASC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		err := rows.Scan(
			&e.ID, &e.ClubID, &e.Name, &e.ShortDescription, &e.Description,
			&e.EventDate, &e.RegistrationDeadline, &e.Venue,
			&e.ParticipationType, &e.MinGroupSize, &e.MaxGroupSize,
			&e.MaxParticipants, &e.RegistrationFee, &e.HasFee, &e.FeeInstructions,
			&e.Rules, &e.Status, &e.CreatedAt, &e.UpdatedAt,
			&e.ClubName, &e.RegistrationCount, &e.GroupCount)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning event: %w", err)
		}
		events = append(events, e)
	}

	return events, totalCount, nil
}

// ListPublicActive returns active events from approved clubs for public display
func (r *EventRepository) ListPublicActive(ctx context.Context, search string, page, pageSize int) ([]models.Event, int, error) {
	offset := (page - 1) * pageSize
	var totalCount int
	searchPattern := "%" + strings.ToLower(search) + "%"

	baseWhere := "e.status = 'active' AND c.verification_status = 'approved' AND c.is_active = true"
	if search != "" {
		baseWhere += " AND (LOWER(e.name) LIKE $1 OR LOWER(c.name) LIKE $1)"
	}

	var err error
	if search != "" {
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events e JOIN clubs c ON c.id = e.club_id WHERE %s", baseWhere)
		err = r.pool.QueryRow(ctx, countQuery, searchPattern).Scan(&totalCount)
	} else {
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events e JOIN clubs c ON c.id = e.club_id WHERE %s", baseWhere)
		err = r.pool.QueryRow(ctx, countQuery).Scan(&totalCount)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("error counting events: %w", err)
	}

	var rows pgx.Rows
	selectFields := `
		SELECT e.id, e.club_id, e.name, e.short_description, e.description, e.event_date,
			e.registration_deadline, e.venue, e.participation_type, e.min_group_size, e.max_group_size,
			e.max_participants, e.registration_fee, e.has_fee, e.fee_instructions, e.rules,
			e.status, e.created_at, e.updated_at,
			c.name as club_name,
			COALESCE((SELECT COUNT(*) FROM registrations reg WHERE reg.event_id = e.id AND reg.status != 'cancelled'), 0) as registration_count,
			COALESCE((SELECT COUNT(*) FROM registration_groups rg WHERE rg.event_id = e.id AND rg.status != 'cancelled'), 0) as group_count
		FROM events e
		JOIN clubs c ON c.id = e.club_id`

	if search != "" {
		query := fmt.Sprintf("%s WHERE %s ORDER BY e.event_date ASC LIMIT $2 OFFSET $3", selectFields, baseWhere)
		rows, err = r.pool.Query(ctx, query, searchPattern, pageSize, offset)
	} else {
		query := fmt.Sprintf("%s WHERE %s ORDER BY e.event_date ASC LIMIT $1 OFFSET $2", selectFields, baseWhere)
		rows, err = r.pool.Query(ctx, query, pageSize, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("error listing events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		err := rows.Scan(
			&e.ID, &e.ClubID, &e.Name, &e.ShortDescription, &e.Description,
			&e.EventDate, &e.RegistrationDeadline, &e.Venue,
			&e.ParticipationType, &e.MinGroupSize, &e.MaxGroupSize,
			&e.MaxParticipants, &e.RegistrationFee, &e.HasFee, &e.FeeInstructions,
			&e.Rules, &e.Status, &e.CreatedAt, &e.UpdatedAt,
			&e.ClubName, &e.RegistrationCount, &e.GroupCount)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning event: %w", err)
		}
		events = append(events, e)
	}

	return events, totalCount, nil
}

// GetTimeline returns events in the next 7 days
func (r *EventRepository) GetTimeline(ctx context.Context) ([]models.Event, error) {
	query := `
		SELECT e.id, e.club_id, e.name, e.short_description, e.description, e.event_date,
			e.registration_deadline, e.venue, e.participation_type, e.min_group_size, e.max_group_size,
			e.max_participants, e.registration_fee, e.has_fee, e.fee_instructions, e.rules,
			e.status, e.created_at, e.updated_at,
			c.name as club_name,
			COALESCE((SELECT COUNT(*) FROM registrations reg WHERE reg.event_id = e.id AND reg.status != 'cancelled'), 0) as registration_count,
			0 as group_count
		FROM events e
		JOIN clubs c ON c.id = e.club_id
		WHERE e.status = 'active' AND c.verification_status = 'approved' AND c.is_active = true
			AND e.event_date >= NOW()
			AND e.event_date <= NOW() + INTERVAL '7 days'
		ORDER BY e.event_date ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error getting timeline: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		err := rows.Scan(
			&e.ID, &e.ClubID, &e.Name, &e.ShortDescription, &e.Description,
			&e.EventDate, &e.RegistrationDeadline, &e.Venue,
			&e.ParticipationType, &e.MinGroupSize, &e.MaxGroupSize,
			&e.MaxParticipants, &e.RegistrationFee, &e.HasFee, &e.FeeInstructions,
			&e.Rules, &e.Status, &e.CreatedAt, &e.UpdatedAt,
			&e.ClubName, &e.RegistrationCount, &e.GroupCount)
		if err != nil {
			return nil, fmt.Errorf("error scanning timeline event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// Update modifies event fields
func (r *EventRepository) Update(ctx context.Context, eventID uuid.UUID, req *models.UpdateEventRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.ShortDescription != nil {
		setClauses = append(setClauses, fmt.Sprintf("short_description = $%d", argIdx))
		args = append(args, *req.ShortDescription)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.EventDate != nil {
		t, err := time.Parse(time.RFC3339, *req.EventDate)
		if err != nil {
			return fmt.Errorf("invalid event_date: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("event_date = $%d", argIdx))
		args = append(args, t)
		argIdx++
	}
	if req.RegistrationDeadline != nil {
		t, err := time.Parse(time.RFC3339, *req.RegistrationDeadline)
		if err != nil {
			return fmt.Errorf("invalid registration_deadline: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("registration_deadline = $%d", argIdx))
		args = append(args, t)
		argIdx++
	}
	if req.Venue != nil {
		setClauses = append(setClauses, fmt.Sprintf("venue = $%d", argIdx))
		args = append(args, *req.Venue)
		argIdx++
	}
	if req.ParticipationType != nil {
		setClauses = append(setClauses, fmt.Sprintf("participation_type = $%d", argIdx))
		args = append(args, *req.ParticipationType)
		argIdx++
	}
	if req.MinGroupSize != nil {
		setClauses = append(setClauses, fmt.Sprintf("min_group_size = $%d", argIdx))
		args = append(args, *req.MinGroupSize)
		argIdx++
	}
	if req.MaxGroupSize != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_group_size = $%d", argIdx))
		args = append(args, *req.MaxGroupSize)
		argIdx++
	}
	if req.MaxParticipants != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_participants = $%d", argIdx))
		args = append(args, *req.MaxParticipants)
		argIdx++
	}
	if req.HasFee != nil {
		setClauses = append(setClauses, fmt.Sprintf("has_fee = $%d", argIdx))
		args = append(args, *req.HasFee)
		argIdx++
	}
	if req.RegistrationFee != nil {
		setClauses = append(setClauses, fmt.Sprintf("registration_fee = $%d", argIdx))
		args = append(args, *req.RegistrationFee)
		argIdx++
	}
	if req.FeeInstructions != nil {
		setClauses = append(setClauses, fmt.Sprintf("fee_instructions = $%d", argIdx))
		args = append(args, *req.FeeInstructions)
		argIdx++
	}
	if req.Rules != nil {
		setClauses = append(setClauses, fmt.Sprintf("rules = $%d", argIdx))
		args = append(args, *req.Rules)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE events SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)
	args = append(args, eventID)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

// ExtendDeadline extends a registration deadline
func (r *EventRepository) ExtendDeadline(ctx context.Context, eventID uuid.UUID, newDeadline time.Time) error {
	query := `UPDATE events SET registration_deadline = $1, updated_at = NOW() WHERE id = $2 AND registration_deadline < $1`
	result, err := r.pool.Exec(ctx, query, newDeadline, eventID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("new deadline must be after the current deadline")
	}
	return nil
}

// Delete soft-deletes an event by setting status to cancelled
func (r *EventRepository) Delete(ctx context.Context, eventID uuid.UUID) error {
	query := `UPDATE events SET status = 'cancelled', updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, eventID)
	return err
}

// GetCustomFields returns custom fields for an event
func (r *EventRepository) GetCustomFields(ctx context.Context, eventID uuid.UUID) ([]models.EventCustomField, error) {
	query := `SELECT id, event_id, field_name, field_type, field_options, is_required, sort_order, created_at
			  FROM event_custom_fields WHERE event_id = $1 ORDER BY sort_order`

	rows, err := r.pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("error getting custom fields: %w", err)
	}
	defer rows.Close()

	var fields []models.EventCustomField
	for rows.Next() {
		var f models.EventCustomField
		err := rows.Scan(&f.ID, &f.EventID, &f.FieldName, &f.FieldType, &f.FieldOptions,
			&f.IsRequired, &f.SortOrder, &f.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("error scanning custom field: %w", err)
		}
		fields = append(fields, f)
	}

	return fields, nil
}

// CreateCustomField inserts a custom field for an event
func (r *EventRepository) CreateCustomField(ctx context.Context, field *models.EventCustomField) error {
	query := `INSERT INTO event_custom_fields (id, event_id, field_name, field_type, field_options, is_required, sort_order, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	field.ID = uuid.New()
	field.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		field.ID, field.EventID, field.FieldName, field.FieldType, field.FieldOptions,
		field.IsRequired, field.SortOrder, field.CreatedAt)
	return err
}

// DeleteCustomFields removes all custom fields for an event
func (r *EventRepository) DeleteCustomFields(ctx context.Context, eventID uuid.UUID) error {
	query := `DELETE FROM event_custom_fields WHERE event_id = $1`
	_, err := r.pool.Exec(ctx, query, eventID)
	return err
}
