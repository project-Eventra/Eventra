package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/evenrtra/backend/internal/middleware"
	"github.com/evenrtra/backend/internal/models"
	"github.com/evenrtra/backend/internal/repositories"
	"github.com/evenrtra/backend/internal/validation"
)

// ClubAdminHandler handles club admin endpoints
type ClubAdminHandler struct {
	eventRepo *repositories.EventRepository
	clubRepo  *repositories.ClubRepository
	regRepo   *repositories.RegistrationRepository
	auditRepo *repositories.AuditLogRepository
}

// NewClubAdminHandler creates a new club admin handler
func NewClubAdminHandler(
	eventRepo *repositories.EventRepository,
	clubRepo *repositories.ClubRepository,
	regRepo *repositories.RegistrationRepository,
	auditRepo *repositories.AuditLogRepository,
) *ClubAdminHandler {
	return &ClubAdminHandler{
		eventRepo: eventRepo,
		clubRepo:  clubRepo,
		regRepo:   regRepo,
		auditRepo: auditRepo,
	}
}

// getClubID extracts and verifies the club admin's club ID from claims
func (h *ClubAdminHandler) getClubID(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	claims := middleware.GetClaims(r.Context())
	userID, _ := uuid.Parse(claims.UserID)

	profile, err := h.clubRepo.GetAdminProfile(r.Context(), userID)
	if err != nil || profile == nil {
		return uuid.Nil, uuid.Nil, err
	}

	return profile.ClubID, userID, nil
}

// GetMe handles GET /api/v1/club-admin/me
func (h *ClubAdminHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	clubID, _, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	club, err := h.clubRepo.GetByID(r.Context(), clubID)
	if err != nil || club == nil {
		writeError(w, http.StatusNotFound, "Club not found")
		return
	}

	writeSuccess(w, "", club)
}

// ListEvents handles GET /api/v1/club-admin/events
func (h *ClubAdminHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	clubID, _, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	page, pageSize := getPagination(r)
	status := r.URL.Query().Get("status")

	events, total, err := h.eventRepo.List(r.Context(), &clubID, status, "", page, pageSize)
	if err != nil {
		log.Printf("Error listing club events: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load events")
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	writeSuccess(w, "", models.PaginatedResponse{
		Items:      events,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// CreateEvent handles POST /api/v1/club-admin/events
func (h *ClubAdminHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	clubID, userID, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	// Check club is approved
	club, _ := h.clubRepo.GetByID(r.Context(), clubID)
	if club == nil || club.VerificationStatus != models.VerificationApproved {
		writeError(w, http.StatusForbidden, "Your club must be verified before creating events")
		return
	}

	var req models.CreateEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Support client field aliases
	if req.Name == "" && req.Title != "" {
		req.Name = req.Title
	}
	if req.EventDate == "" && req.StartTime != "" {
		req.EventDate = req.StartTime
	}
	if req.ParticipationType == "" && req.RegistrationType != "" {
		req.ParticipationType = req.RegistrationType
	}
	if req.ParticipationType == "" {
		req.ParticipationType = "individual"
	}
	if req.MaxParticipants == nil && req.MaxCapacity != nil {
		req.MaxParticipants = req.MaxCapacity
	}
	if req.ShortDescription == "" && req.Description != "" && len(req.Description) > 0 {
		runes := []rune(req.Description)
		if len(runes) > 120 {
			req.ShortDescription = string(runes[:117]) + "..."
		} else {
			req.ShortDescription = req.Description
		}
	}
	if (req.ParticipationType == "group" || req.ParticipationType == "both") && req.MinGroupSize < 2 {
		req.MinGroupSize = 2
	}
	if (req.ParticipationType == "group" || req.ParticipationType == "both") && req.MaxGroupSize < req.MinGroupSize {
		req.MaxGroupSize = 5
	}

	v := validation.New()
	v.Required("name", req.Name).MaxLength("name", req.Name, 300)
	v.Required("event_date", req.EventDate)
	v.Required("registration_deadline", req.RegistrationDeadline)
	v.Required("participation_type", req.ParticipationType)
	v.OneOf("participation_type", req.ParticipationType, []string{"individual", "group", "both"})

	if req.ParticipationType == "group" || req.ParticipationType == "both" {
		v.Min("min_group_size", req.MinGroupSize, 2)
		v.Min("max_group_size", req.MaxGroupSize, req.MinGroupSize)
	}
	if req.HasFee {
		v.MinFloat("registration_fee", req.RegistrationFee, 0.01)
	}

	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	eventDate, err := parseFlexibleDate(req.EventDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event date format: "+err.Error())
		return
	}
	deadline, err := parseFlexibleDate(req.RegistrationDeadline)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid deadline format: "+err.Error())
		return
	}

	if deadline.After(eventDate) {
		writeError(w, http.StatusBadRequest, "Registration deadline must be before or on the event date")
		return
	}

	// Build event model
	var shortDesc, desc, venue, feeInstr, rules *string
	if req.ShortDescription != "" {
		shortDesc = &req.ShortDescription
	}
	if req.Description != "" {
		desc = &req.Description
	}
	if req.Venue != "" {
		venue = &req.Venue
	}
	if req.FeeInstructions != "" {
		feeInstr = &req.FeeInstructions
	}
	if req.Rules != "" {
		rules = &req.Rules
	}

	minGS := 1
	maxGS := 1
	if req.ParticipationType == "group" || req.ParticipationType == "both" {
		minGS = req.MinGroupSize
		maxGS = req.MaxGroupSize
	}

	event := &models.Event{
		ClubID:               clubID,
		Name:                 req.Name,
		ShortDescription:     shortDesc,
		Description:          desc,
		EventDate:            eventDate,
		RegistrationDeadline: deadline,
		Venue:                venue,
		ParticipationType:    req.ParticipationType,
		MinGroupSize:         minGS,
		MaxGroupSize:         maxGS,
		MaxParticipants:      req.MaxParticipants,
		RegistrationFee:      req.RegistrationFee,
		HasFee:               req.HasFee,
		FeeInstructions:      feeInstr,
		Rules:                rules,
		Status:               models.EventActive,
	}

	if err := h.eventRepo.Create(r.Context(), event); err != nil {
		log.Printf("Error creating event: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create event")
		return
	}

	// Create custom fields
	for i, cf := range req.CustomFields {
		fieldName := cf.FieldName
		if fieldName == "" {
			if cf.Label != "" {
				fieldName = cf.Label
			} else if cf.Name != "" {
				fieldName = cf.Name
			}
		}
		if fieldName == "" {
			continue
		}

		fieldType := cf.FieldType
		if fieldType == "" {
			if cf.Type != "" {
				fieldType = cf.Type
			} else {
				fieldType = "text"
			}
		}

		isRequired := cf.IsRequired || cf.Required
		sortOrder := cf.SortOrder
		if sortOrder == 0 {
			sortOrder = i + 1
		}

		var opts *string
		if cf.FieldOptions != "" {
			opts = &cf.FieldOptions
		}
		field := &models.EventCustomField{
			EventID:      event.ID,
			FieldName:    fieldName,
			FieldType:    fieldType,
			FieldOptions: opts,
			IsRequired:   isRequired,
			SortOrder:    sortOrder,
		}
		h.eventRepo.CreateCustomField(r.Context(), field)
	}

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &userID,
		Action:     "create_event",
		EntityType: "event",
		EntityID:   &event.ID,
	})

	writeCreated(w, "Event created successfully", event)
}

// UpdateEvent handles PATCH /api/v1/club-admin/events/{id}
func (h *ClubAdminHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	clubID, _, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	// Verify ownership
	event, err := h.eventRepo.GetByID(r.Context(), eventID)
	if err != nil || event == nil {
		writeError(w, http.StatusNotFound, "Event not found")
		return
	}
	if event.ClubID != clubID {
		writeError(w, http.StatusForbidden, "You can only edit your own club's events")
		return
	}

	var req models.UpdateEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.eventRepo.Update(r.Context(), eventID, &req); err != nil {
		log.Printf("Error updating event: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to update event")
		return
	}

	updated, _ := h.eventRepo.GetByID(r.Context(), eventID)
	writeSuccess(w, "Event updated successfully", updated)
}

// DeleteEvent handles DELETE /api/v1/club-admin/events/{id}
func (h *ClubAdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	clubID, userID, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	event, _ := h.eventRepo.GetByID(r.Context(), eventID)
	if event == nil {
		writeError(w, http.StatusNotFound, "Event not found")
		return
	}
	if event.ClubID != clubID {
		writeError(w, http.StatusForbidden, "You can only delete your own club's events")
		return
	}

	if err := h.eventRepo.Delete(r.Context(), eventID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to cancel event")
		return
	}

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &userID,
		Action:     "cancel_event",
		EntityType: "event",
		EntityID:   &eventID,
	})

	writeSuccess(w, "Event cancelled successfully", nil)
}

// ExtendDeadline handles PATCH /api/v1/club-admin/events/{id}/deadline
func (h *ClubAdminHandler) ExtendDeadline(w http.ResponseWriter, r *http.Request) {
	clubID, _, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	event, _ := h.eventRepo.GetByID(r.Context(), eventID)
	if event == nil {
		writeError(w, http.StatusNotFound, "Event not found")
		return
	}
	if event.ClubID != clubID {
		writeError(w, http.StatusForbidden, "You can only modify your own club's events")
		return
	}

	var req models.ExtendDeadlineRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	newDeadline, err := parseFlexibleDate(req.NewDeadline)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid deadline format: "+err.Error())
		return
	}

	if err := h.eventRepo.ExtendDeadline(r.Context(), eventID, newDeadline); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeSuccess(w, "Deadline extended successfully", nil)
}

// GetEventRegistrations handles GET /api/v1/club-admin/events/{id}/registrations
func (h *ClubAdminHandler) GetEventRegistrations(w http.ResponseWriter, r *http.Request) {
	clubID, _, err := h.getClubID(r)
	if err != nil {
		writeError(w, http.StatusForbidden, "Club admin profile not found")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	event, _ := h.eventRepo.GetByID(r.Context(), eventID)
	if event == nil {
		writeError(w, http.StatusNotFound, "Event not found")
		return
	}
	if event.ClubID != clubID {
		writeError(w, http.StatusForbidden, "You can only view registrations for your own club's events")
		return
	}

	page, pageSize := getPagination(r)
	regs, total, err := h.regRepo.GetByEventID(r.Context(), eventID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load registrations")
		return
	}

	// Also get group registrations
	groups, _ := h.regRepo.GetGroupsByEventID(r.Context(), eventID)

	totalPages := (total + pageSize - 1) / pageSize
	writeSuccess(w, "", map[string]interface{}{
		"registrations": models.PaginatedResponse{
			Items:      regs,
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
		"groups": groups,
	})
}

func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date %q (supported: RFC3339, ISO8601)", s)
}
