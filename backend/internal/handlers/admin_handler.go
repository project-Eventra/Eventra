package handlers

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/evenrtra/backend/internal/middleware"
	"github.com/evenrtra/backend/internal/models"
	"github.com/evenrtra/backend/internal/repositories"
	"github.com/evenrtra/backend/internal/validation"
)

// AdminHandler handles super admin endpoints
type AdminHandler struct {
	userRepo     *repositories.UserRepository
	studentRepo  *repositories.StudentRepository
	clubRepo     *repositories.ClubRepository
	eventRepo    *repositories.EventRepository
	regRepo      *repositories.RegistrationRepository
	feedbackRepo *repositories.FeedbackRepository
	auditRepo    *repositories.AuditLogRepository
	pool         *pgxpool.Pool
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	userRepo *repositories.UserRepository,
	studentRepo *repositories.StudentRepository,
	clubRepo *repositories.ClubRepository,
	eventRepo *repositories.EventRepository,
	regRepo *repositories.RegistrationRepository,
	feedbackRepo *repositories.FeedbackRepository,
	auditRepo *repositories.AuditLogRepository,
	pool *pgxpool.Pool,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		studentRepo:  studentRepo,
		clubRepo:     clubRepo,
		eventRepo:    eventRepo,
		regRepo:      regRepo,
		feedbackRepo: feedbackRepo,
		auditRepo:    auditRepo,
		pool:         pool,
	}
}

// GetDashboardStats handles GET /api/v1/admin/stats
func (h *AdminHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := repositories.GetDashboardStats(r.Context(), h.pool)
	if err != nil {
		log.Printf("Error getting dashboard stats: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load statistics")
		return
	}
	writeSuccess(w, "", stats)
}

// ListClubs handles GET /api/v1/admin/clubs
func (h *AdminHandler) ListClubs(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPagination(r)
	status := r.URL.Query().Get("status")

	clubs, total, err := h.clubRepo.List(r.Context(), status, page, pageSize)
	if err != nil {
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

// GetClub handles GET /api/v1/admin/clubs/{id}
func (h *AdminHandler) GetClub(w http.ResponseWriter, r *http.Request) {
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

// CreateClub handles POST /api/v1/admin/clubs
func (h *AdminHandler) CreateClub(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	adminID, _ := uuid.Parse(claims.UserID)

	var req models.AdminCreateClubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validation.New()
	v.Required("name", req.Name)
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("admin_email", req.AdminEmail).Email("admin_email", req.AdminEmail)
	v.Required("admin_password", req.AdminPassword).Password("admin_password", req.AdminPassword)

	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	// Create admin user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), bcrypt.DefaultCost)
	adminUser := &models.User{
		Email:        req.AdminEmail,
		PasswordHash: string(hashedPassword),
		Role:         models.RoleClubAdmin,
		IsActive:     true,
	}
	if err := h.userRepo.Create(r.Context(), adminUser); err != nil {
		writeError(w, http.StatusConflict, "Admin email already exists")
		return
	}

	// Create club (auto-approved by super admin)
	var desc, phone *string
	if req.Description != "" {
		desc = &req.Description
	}
	if req.Phone != "" {
		phone = &req.Phone
	}

	club := &models.Club{
		Name:               req.Name,
		Description:        desc,
		Email:              req.Email,
		Phone:              phone,
		VerificationStatus: models.VerificationApproved,
		IsActive:           true,
		VerifiedBy:         &adminID,
	}
	if err := h.clubRepo.Create(r.Context(), club); err != nil {
		h.userRepo.Delete(r.Context(), adminUser.ID)
		writeError(w, http.StatusConflict, "Club email already exists")
		return
	}

	// Link admin to club
	profile := &models.ClubAdminProfile{
		UserID:    adminUser.ID,
		ClubID:    club.ID,
		IsPrimary: true,
	}
	h.clubRepo.CreateAdminProfile(r.Context(), profile)

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &adminID,
		Action:     "admin_create_club",
		EntityType: "club",
		EntityID:   &club.ID,
	})

	writeCreated(w, "Club created successfully", club)
}

// UpdateClub handles PATCH /api/v1/admin/clubs/{id}
func (h *AdminHandler) UpdateClub(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	var req models.UpdateClubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.clubRepo.Update(r.Context(), clubID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update club")
		return
	}

	updated, _ := h.clubRepo.GetByID(r.Context(), clubID)
	writeSuccess(w, "Club updated successfully", updated)
}

// DeleteClub handles DELETE /api/v1/admin/clubs/{id}
func (h *AdminHandler) DeleteClub(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	claims := middleware.GetClaims(r.Context())
	adminID, _ := uuid.Parse(claims.UserID)

	if err := h.clubRepo.Delete(r.Context(), clubID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to deactivate club")
		return
	}

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &adminID,
		Action:     "deactivate_club",
		EntityType: "club",
		EntityID:   &clubID,
	})

	writeSuccess(w, "Club deactivated successfully", nil)
}

// ApproveClub handles POST /api/v1/admin/clubs/{id}/approve
func (h *AdminHandler) ApproveClub(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	claims := middleware.GetClaims(r.Context())
	adminID, _ := uuid.Parse(claims.UserID)

	if err := h.clubRepo.Approve(r.Context(), clubID, adminID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to approve club")
		return
	}

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &adminID,
		Action:     "approve_club",
		EntityType: "club",
		EntityID:   &clubID,
	})

	writeSuccess(w, "Club approved successfully", nil)
}

// RejectClub handles POST /api/v1/admin/clubs/{id}/reject
func (h *AdminHandler) RejectClub(w http.ResponseWriter, r *http.Request) {
	clubID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid club ID")
		return
	}

	var req models.RejectClubRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	claims := middleware.GetClaims(r.Context())
	adminID, _ := uuid.Parse(claims.UserID)

	if err := h.clubRepo.Reject(r.Context(), clubID, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reject club")
		return
	}

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &adminID,
		Action:     "reject_club",
		EntityType: "club",
		EntityID:   &clubID,
	})

	writeSuccess(w, "Club rejected", nil)
}

// ListStudents handles GET /api/v1/admin/students
func (h *AdminHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPagination(r)
	search := r.URL.Query().Get("search")

	profiles, users, total, err := h.studentRepo.ListStudents(r.Context(), search, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load students")
		return
	}

	type studentItem struct {
		models.StudentProfile
		Email         string `json:"email"`
		IsActive      bool   `json:"is_active"`
		EmailVerified bool   `json:"email_verified"`
	}

	var items []studentItem
	for i, p := range profiles {
		items = append(items, studentItem{
			StudentProfile: p,
			Email:          users[i].Email,
			IsActive:       users[i].IsActive,
			EmailVerified:  users[i].EmailVerified,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	writeSuccess(w, "", models.PaginatedResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// GetStudent handles GET /api/v1/admin/students/{id}
func (h *AdminHandler) GetStudent(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	user, _ := h.userRepo.GetByID(r.Context(), userID)
	if user == nil {
		writeError(w, http.StatusNotFound, "Student not found")
		return
	}

	profile, _ := h.studentRepo.GetByUserID(r.Context(), userID)

	writeSuccess(w, "", map[string]interface{}{
		"user":    user,
		"profile": profile,
	})
}

// UpdateStudent handles PATCH /api/v1/admin/students/{id}
func (h *AdminHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	var req models.UpdateStudentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update active status on user record if provided
	if req.IsActive != nil {
		h.userRepo.UpdateActive(r.Context(), userID, *req.IsActive)
	}

	// Update profile fields
	if err := h.studentRepo.Update(r.Context(), userID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update student")
		return
	}

	writeSuccess(w, "Student updated successfully", nil)
}

// DeleteStudent handles DELETE /api/v1/admin/students/{id}
func (h *AdminHandler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	claims := middleware.GetClaims(r.Context())
	adminID, _ := uuid.Parse(claims.UserID)

	// Deactivate instead of hard delete to preserve referential integrity
	h.userRepo.UpdateActive(r.Context(), userID, false)

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &adminID,
		Action:     "deactivate_student",
		EntityType: "student",
		EntityID:   &userID,
	})

	writeSuccess(w, "Student account deactivated", nil)
}

// ListAllEvents handles GET /api/v1/admin/events
func (h *AdminHandler) ListAllEvents(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPagination(r)
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	events, total, err := h.eventRepo.List(r.Context(), nil, status, search, page, pageSize)
	if err != nil {
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

// UpdateEvent handles PATCH /api/v1/admin/events/{id}
func (h *AdminHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.UpdateEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.eventRepo.Update(r.Context(), eventID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update event")
		return
	}

	writeSuccess(w, "Event updated successfully", nil)
}

// DeleteEvent handles DELETE /api/v1/admin/events/{id}
func (h *AdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	claims := middleware.GetClaims(r.Context())
	adminID, _ := uuid.Parse(claims.UserID)

	if err := h.eventRepo.Delete(r.Context(), eventID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to cancel event")
		return
	}

	h.auditRepo.Create(r.Context(), &models.AuditLog{
		ActorID:    &adminID,
		Action:     "admin_cancel_event",
		EntityType: "event",
		EntityID:   &eventID,
	})

	writeSuccess(w, "Event cancelled successfully", nil)
}

// GetEventRegistrations handles GET /api/v1/admin/events/{id}/registrations
func (h *AdminHandler) GetEventRegistrations(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	page, pageSize := getPagination(r)
	regs, total, err := h.regRepo.GetByEventID(r.Context(), eventID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load registrations")
		return
	}

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
		"items":  regs,
		"groups": groups,
	})
}

// ListFeedback handles GET /api/v1/admin/feedback
func (h *AdminHandler) ListFeedback(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPagination(r)
	status := r.URL.Query().Get("status")

	feedbacks, total, err := h.feedbackRepo.List(r.Context(), status, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load feedback")
		return
	}

	totalPages := (total + pageSize - 1) / pageSize
	writeSuccess(w, "", models.PaginatedResponse{
		Items:      feedbacks,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

// UpdateFeedback handles PATCH /api/v1/admin/feedback/{id}
func (h *AdminHandler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	feedbackID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid feedback ID")
		return
	}

	var req models.UpdateFeedbackRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.feedbackRepo.Update(r.Context(), feedbackID, &req); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update feedback")
		return
	}

	writeSuccess(w, "Feedback updated successfully", nil)
}
