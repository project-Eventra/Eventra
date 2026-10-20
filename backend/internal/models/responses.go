package models

// Response DTOs for API endpoints

// APIResponse is the standard JSON response envelope
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// AuthResponse returned after successful authentication
type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token,omitempty"` // Only if not using cookie
}

// UserResponse is a safe user representation (no password hash)
type UserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Role          string  `json:"role"`
	IsActive      bool    `json:"is_active"`
	EmailVerified bool    `json:"email_verified"`
	// Student fields (populated when role is student)
	FirstName          string `json:"first_name,omitempty"`
	LastName           string `json:"last_name,omitempty"`
	RegistrationNumber string `json:"registration_number,omitempty"`
	Department         string `json:"department,omitempty"`
	AcademicSession    string `json:"academic_session,omitempty"`
	Phone              string `json:"phone,omitempty"`
	// Club admin fields
	ClubID             string `json:"club_id,omitempty"`
	ClubName           string `json:"club_name,omitempty"`
	ClubStatus         string `json:"club_status,omitempty"`
}

// EventListResponse for event listing
type EventListResponse struct {
	ID                   string  `json:"id"`
	ClubID               string  `json:"club_id"`
	ClubName             string  `json:"club_name"`
	Name                 string  `json:"name"`
	ShortDescription     string  `json:"short_description"`
	EventDate            string  `json:"event_date"`
	RegistrationDeadline string  `json:"registration_deadline"`
	Venue                string  `json:"venue"`
	ParticipationType    string  `json:"participation_type"`
	HasFee               bool    `json:"has_fee"`
	RegistrationFee      float64 `json:"registration_fee"`
	Status               string  `json:"status"`
	RegistrationCount    int     `json:"registration_count"`
	MaxParticipants      *int    `json:"max_participants"`
	IsRegistered         bool    `json:"is_registered"`
}

// DashboardStats for admin dashboards
type DashboardStats struct {
	TotalStudents      int `json:"total_students"`
	TotalClubs         int `json:"total_clubs"`
	TotalEvents        int `json:"total_events"`
	TotalRegistrations int `json:"total_registrations"`
	PendingClubs       int `json:"pending_clubs"`
	ActiveEvents       int `json:"active_events"`
	NewFeedback        int `json:"new_feedback"`
}
