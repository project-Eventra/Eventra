package handlers

import (
	"encoding/json"
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

// EventHandler handles event endpoints
type EventHandler struct {
	eventRepo *repositories.EventRepository
	regRepo   *repositories.RegistrationRepository
	clubRepo  *repositories.ClubRepository
}

// NewEventHandler creates a new event handler
func NewEventHandler(eventRepo *repositories.EventRepository, regRepo *repositories.RegistrationRepository, clubRepo *repositories.ClubRepository) *EventHandler {
	return &EventHandler{eventRepo: eventRepo, regRepo: regRepo, clubRepo: clubRepo}
}

// ListPublicEvents handles GET /api/v1/events
func (h *EventHandler) ListPublicEvents(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page, pageSize := getPagination(r)

	events, total, err := h.eventRepo.ListPublicActive(r.Context(), search, page, pageSize)
	if err != nil {
		log.Printf("Error listing events: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load events")
		return
	}

	// Check registration status per user
	claims := middleware.GetClaims(r.Context())
	if claims != nil {
		userID, err := uuid.Parse(claims.UserID)
		if err == nil {
			for i := range events {
				registered, _ := h.regRepo.CheckUserRegistered(r.Context(), events[i].ID, userID)
				events[i].IsRegistered = registered
			}
		}
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

// GetEvent handles GET /api/v1/events/{id}
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	event, err := h.eventRepo.GetByID(r.Context(), eventID)
	if err != nil {
		log.Printf("Error getting event: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load event")
		return
	}
	if event == nil {
		writeError(w, http.StatusNotFound, "Event not found")
		return
	}

	claims := middleware.GetClaims(r.Context())
	if claims != nil {
		userID, err := uuid.Parse(claims.UserID)
		if err == nil {
			registered, _ := h.regRepo.CheckUserRegistered(r.Context(), eventID, userID)
			event.IsRegistered = registered
		}
	}

	customFields, _ := h.eventRepo.GetCustomFields(r.Context(), eventID)
	if customFields == nil {
		customFields = []models.EventCustomField{}
	}
	for i := range customFields {
		customFields[i].Label = customFields[i].FieldName
		customFields[i].Name = customFields[i].FieldName
		customFields[i].Type = customFields[i].FieldType
		customFields[i].Required = customFields[i].IsRequired
	}
	event.CustomFields = customFields

	writeSuccess(w, "", map[string]interface{}{
		"event":                 event,
		"custom_fields":         customFields,
		"id":                    event.ID,
		"name":                  event.Name,
		"title":                 event.Name,
		"club_id":               event.ClubID,
		"club_name":             event.ClubName,
		"category":              event.Category,
		"description":           event.Description,
		"short_description":     event.ShortDescription,
		"event_date":            event.EventDate,
		"start_time":            event.EventDate,
		"end_time":              event.EventDate,
		"registration_deadline": event.RegistrationDeadline,
		"venue":                 event.Venue,
		"participation_type":    event.ParticipationType,
		"registration_type":     event.ParticipationType,
		"min_group_size":        event.MinGroupSize,
		"max_group_size":        event.MaxGroupSize,
		"max_participants":      event.MaxParticipants,
		"max_capacity":          event.MaxParticipants,
		"registration_count":    event.RegistrationCount,
		"status":                event.Status,
		"is_registered":         event.IsRegistered,
	})
}

// GetTimeline handles GET /api/v1/events/timeline
func (h *EventHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	events, err := h.eventRepo.GetTimeline(r.Context())
	if err != nil {
		log.Printf("Error getting timeline: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load timeline")
		return
	}
	if events == nil {
		events = []models.Event{}
	}
	writeSuccess(w, "", events)
}

// ListClubs handles GET /api/v1/clubs
func (h *EventHandler) ListClubs(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPagination(r)
	clubs, total, err := h.clubRepo.List(r.Context(), models.VerificationApproved, page, pageSize)
	if err != nil {
		log.Printf("Error listing clubs: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load clubs")
		return
	}
	totalPages := (total + pageSize - 1) / pageSize
	writeSuccess(w, "", models.PaginatedResponse{
		Items:      clubs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// GetClub handles GET /api/v1/clubs/{id}
func (h *EventHandler) GetClub(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}
	club, err := h.clubRepo.GetByID(r.Context(), clubID)
	if err != nil || club == nil {
		writeError(w, http.StatusNotFound, "Club not found")
		return
	}
	writeSuccess(w, "", club)
}

// RegisterForEvent handles POST /api/v1/events/{id}/registrations
func (h *EventHandler) RegisterForEvent(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid session")
		return
	}

	var req models.CreateRegistrationRequest
	if err := decodeJSON(r, &req); err != nil {
		req = models.CreateRegistrationRequest{}
	}

	fields := req.CustomFieldsData
	if fields == nil {
		fields = req.CustomData
	}

	var customData *string
	if fields != nil && len(fields) > 0 {
		b, _ := json.Marshal(fields)
		s := string(b)
		customData = &s
	}

	reg := &models.Registration{
		EventID:          eventID,
		UserID:           userID,
		CustomFieldsData: customData,
	}

	if err := h.regRepo.CreateIndividual(r.Context(), reg); err != nil {
		log.Printf("Error creating registration: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeCreated(w, "Registration completed", reg)
}

// RegisterGroup handles POST /api/v1/events/{id}/group-registrations
func (h *EventHandler) RegisterGroup(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid session")
		return
	}

	var req models.CreateGroupRegistrationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GroupName == "" && req.TeamName != "" {
		req.GroupName = req.TeamName
	}

	v := validation.New()
	v.Required("group_name", req.GroupName)
	v.Min("members", len(req.Members), 1)
	for i, m := range req.Members {
		v.Required(fmt.Sprintf("members[%d].member_name", i), m.MemberName)
	}
	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	group := &models.RegistrationGroup{
		EventID:      eventID,
		GroupName:    req.GroupName,
		LeaderUserID: userID,
	}

	var members []models.GroupMember
	for _, m := range req.Members {
		member := models.GroupMember{MemberName: m.MemberName}
		if m.MemberEmail != "" {
			member.MemberEmail = &m.MemberEmail
		}
		if m.MemberPhone != "" {
			member.MemberPhone = &m.MemberPhone
		}
		if m.MemberRegistrationNumber != "" {
			member.MemberRegistrationNumber = &m.MemberRegistrationNumber
		}
		if m.CustomFieldsData != nil {
			b, _ := json.Marshal(m.CustomFieldsData)
			s := string(b)
			member.CustomFieldsData = &s
		}
		members = append(members, member)
	}

	groupFields := req.CustomFieldsData
	if groupFields == nil {
		groupFields = req.CustomData
	}
	var customData *string
	if groupFields != nil && len(groupFields) > 0 {
		b, _ := json.Marshal(groupFields)
		s := string(b)
		customData = &s
	}

	leaderReg := &models.Registration{
		UserID:           userID,
		CustomFieldsData: customData,
	}

	if err := h.regRepo.CreateGroup(r.Context(), group, members, leaderReg); err != nil {
		log.Printf("Error creating group registration: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeCreated(w, "Registration completed", map[string]interface{}{
		"group":        group,
		"registration": leaderReg,
	})
}

// GetMyRegistrations handles GET /api/v1/students/me/registrations
func (h *EventHandler) GetMyRegistrations(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid session")
		return
	}

	regs, err := h.regRepo.GetByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("Error getting registrations: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load registrations")
		return
	}
	if regs == nil {
		regs = []models.Registration{}
	}

	writeSuccess(w, "", regs)
}

// GetRegistration handles GET /api/v1/registrations/{id}
func (h *EventHandler) GetRegistration(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	regID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid registration ID")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid session")
		return
	}

	reg, err := h.regRepo.GetByID(r.Context(), regID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load registration")
		return
	}
	if reg == nil {
		writeError(w, http.StatusNotFound, "Registration not found")
		return
	}

	if reg.UserID != userID && claims.Role != models.RoleSuperAdmin {
		writeError(w, http.StatusForbidden, "Access denied")
		return
	}

	writeSuccess(w, "", reg)
}

// Ensure imports are used
var _ = time.Now
