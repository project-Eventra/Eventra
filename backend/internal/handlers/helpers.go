package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/evenrtra/backend/internal/models"
)

// Helper functions for handlers

// writeJSON sends a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeSuccess sends a success response
func writeSuccess(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// writeCreated sends a 201 created response
func writeCreated(w http.ResponseWriter, message string, data interface{}) {
	writeJSON(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// writeError sends an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.APIResponse{
		Success: false,
		Error:   message,
	})
}

// writeValidationError sends validation errors
func writeValidationError(w http.ResponseWriter, errors interface{}) {
	writeJSON(w, http.StatusBadRequest, models.APIResponse{
		Success: false,
		Error:   "Validation failed",
		Data:    errors,
	})
}

// decodeJSON reads and parses a JSON request body
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// getPagination extracts page and page_size from query parameters
func getPagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	return page, pageSize
}

// getPathParam extracts a parameter from the URL path
// For chi router: use chi.URLParam instead
// This is a fallback for manual path extraction
func getPathParam(r *http.Request, prefix string) string {
	path := r.URL.Path
	idx := strings.LastIndex(path, prefix)
	if idx == -1 {
		return ""
	}
	rest := path[idx+len(prefix):]
	// Remove trailing slash
	rest = strings.TrimSuffix(rest, "/")
	// Get the next segment
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
