package handlers

import (
	"log"
	"net/http"

	"github.com/google/uuid"

	"github.com/evenrtra/backend/internal/middleware"
	"github.com/evenrtra/backend/internal/models"
	"github.com/evenrtra/backend/internal/repositories"
	"github.com/evenrtra/backend/internal/validation"
)

// FeedbackHandler handles feedback endpoints
type FeedbackHandler struct {
	feedbackRepo *repositories.FeedbackRepository
}

// NewFeedbackHandler creates a new feedback handler
func NewFeedbackHandler(feedbackRepo *repositories.FeedbackRepository) *FeedbackHandler {
	return &FeedbackHandler{feedbackRepo: feedbackRepo}
}

// SubmitFeedback handles POST /api/v1/feedback
func (h *FeedbackHandler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	var req models.FeedbackRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validation.New()
	v.Required("message", req.Message).MaxLength("message", req.Message, 5000)
	if req.Category != "" {
		v.OneOf("category", req.Category, []string{"complaint", "suggestion", "general", "bug"})
	}
	if req.Email != "" {
		v.Email("email", req.Email)
	}

	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	category := req.Category
	if category == "" {
		category = "general"
	}

	var userID *uuid.UUID
	claims := middleware.GetClaims(r.Context())
	if claims != nil {
		uid, err := uuid.Parse(claims.UserID)
		if err == nil {
			userID = &uid
		}
	}

	var name, email, subject *string
	if req.Name != "" {
		name = &req.Name
	}
	if req.Email != "" {
		email = &req.Email
	}
	if req.Subject != "" {
		subject = &req.Subject
	}

	fb := &models.Feedback{
		UserID:   userID,
		Name:     name,
		Email:    email,
		Category: category,
		Subject:  subject,
		Message:  req.Message,
	}

	if err := h.feedbackRepo.Create(r.Context(), fb); err != nil {
		log.Printf("Error creating feedback: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to submit feedback")
		return
	}

	writeCreated(w, "Feedback submitted successfully. Thank you!", fb)
}
