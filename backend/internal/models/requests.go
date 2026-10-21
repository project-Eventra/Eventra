package models

// Request DTOs for API endpoints

// StudentRegistrationRequest for student signup
type StudentRegistrationRequest struct {
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	Name               string `json:"name"`
	RegistrationNumber string `json:"registration_number"`
	StudentID          string `json:"student_id"`
	Department         string `json:"department"`
	AcademicSession    string `json:"academic_session"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	Phone              string `json:"phone"`
	FirebaseToken      string `json:"firebase_token"`
}

// ClubRegistrationRequest for club admin signup
type ClubRegistrationRequest struct {
	ClubName        string `json:"club_name"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	ClubDescription string `json:"club_description"`
	Description     string `json:"description"`
	Email           string `json:"email"`
	ContactEmail    string `json:"contact_email"`
	Phone           string `json:"phone"`
	Password        string `json:"password"`
	AdminPassword   string `json:"admin_password"`
	AdminName       string `json:"admin_name"`
	FirebaseToken   string `json:"firebase_token"`
}

// LoginRequest for email/password login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// Alternative: login with registration number
	RegistrationNumber string `json:"registration_number"`
	Role               string `json:"role"`
}

// SuperAdminRegistrationRequest for authorized super admin signup
type SuperAdminRegistrationRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	BootstrapKey string `json:"bootstrap_key"`
}

// FirebaseAuthRequest for Firebase ID token login
type FirebaseAuthRequest struct {
	IDToken string `json:"id_token"`
	Role    string `json:"role"`
}

// CreateEventRequest for creating an event
type CreateEventRequest struct {
	Name                 string                     `json:"name"`
	ShortDescription     string                     `json:"short_description"`
	Description          string                     `json:"description"`
	EventDate            string                     `json:"event_date"`
	RegistrationDeadline string                     `json:"registration_deadline"`
	Venue                string                     `json:"venue"`
	ParticipationType    string                     `json:"participation_type"`
	MinGroupSize         int                        `json:"min_group_size"`
	MaxGroupSize         int                        `json:"max_group_size"`
	MaxParticipants      *int                       `json:"max_participants"`
	HasFee               bool                       `json:"has_fee"`
	RegistrationFee      float64                    `json:"registration_fee"`
	FeeInstructions      string                     `json:"fee_instructions"`
	Rules                string                     `json:"rules"`
	CustomFields         []CreateCustomFieldRequest `json:"custom_fields"`
}

// CreateCustomFieldRequest for event custom fields
type CreateCustomFieldRequest struct {
	FieldName    string `json:"field_name"`
	FieldType    string `json:"field_type"`
	FieldOptions string `json:"field_options"`
	IsRequired   bool   `json:"is_required"`
	SortOrder    int    `json:"sort_order"`
}

// UpdateEventRequest for editing an event
type UpdateEventRequest struct {
	Name                 *string  `json:"name"`
	ShortDescription     *string  `json:"short_description"`
	Description          *string  `json:"description"`
	EventDate            *string  `json:"event_date"`
	RegistrationDeadline *string  `json:"registration_deadline"`
	Venue                *string  `json:"venue"`
	ParticipationType    *string  `json:"participation_type"`
	MinGroupSize         *int     `json:"min_group_size"`
	MaxGroupSize         *int     `json:"max_group_size"`
	MaxParticipants      *int     `json:"max_participants"`
	HasFee               *bool    `json:"has_fee"`
	RegistrationFee      *float64 `json:"registration_fee"`
	FeeInstructions      *string  `json:"fee_instructions"`
	Rules                *string  `json:"rules"`
	Status               *string  `json:"status"`
}

// ExtendDeadlineRequest for deadline extension
type ExtendDeadlineRequest struct {
	NewDeadline string `json:"new_deadline"`
}

// CreateRegistrationRequest for individual registration
type CreateRegistrationRequest struct {
	CustomFieldsData map[string]interface{} `json:"custom_fields_data"`
}

// CreateGroupRegistrationRequest for group registration
type CreateGroupRegistrationRequest struct {
	GroupName        string                 `json:"group_name"`
	Members          []GroupMemberRequest   `json:"members"`
	CustomFieldsData map[string]interface{} `json:"custom_fields_data"`
}

// GroupMemberRequest for adding group members
type GroupMemberRequest struct {
	MemberName               string                 `json:"member_name"`
	MemberEmail              string                 `json:"member_email"`
	MemberPhone              string                 `json:"member_phone"`
	MemberRegistrationNumber string                 `json:"member_registration_number"`
	CustomFieldsData         map[string]interface{} `json:"custom_fields_data"`
}

// FeedbackRequest for submitting feedback
type FeedbackRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Category string `json:"category"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
}

// UpdateClubRequest for admin club management
type UpdateClubRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	IsActive    *bool   `json:"is_active"`
}

// UpdateStudentRequest for admin student management
type UpdateStudentRequest struct {
	FirstName       *string `json:"first_name"`
	LastName        *string `json:"last_name"`
	Department      *string `json:"department"`
	AcademicSession *string `json:"academic_session"`
	IsActive        *bool   `json:"is_active"`
}

// RejectClubRequest for club rejection
type RejectClubRequest struct {
	Reason string `json:"reason"`
}

// ForgotPasswordRequest for password reset
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// UpdateFeedbackRequest for admin feedback management
type UpdateFeedbackRequest struct {
	Status     *string `json:"status"`
	AdminNotes *string `json:"admin_notes"`
}

// AdminCreateClubRequest for super admin to add a club
type AdminCreateClubRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}
