package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents the base authentication record
type User struct {
	ID            uuid.UUID `json:"id" db:"id"`
	FirebaseUID   *string   `json:"firebase_uid,omitempty" db:"firebase_uid"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Role          string    `json:"role" db:"role"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// StudentProfile holds student-specific data
type StudentProfile struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	FirstName          string    `json:"first_name" db:"first_name"`
	LastName           string    `json:"last_name" db:"last_name"`
	RegistrationNumber string    `json:"registration_number" db:"registration_number"`
	Department         string    `json:"department" db:"department"`
	AcademicSession    string    `json:"academic_session" db:"academic_session"`
	Phone              *string   `json:"phone,omitempty" db:"phone"`
	PhoneVerified      bool      `json:"phone_verified" db:"phone_verified"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// Club represents a university club
type Club struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	Name               string     `json:"name" db:"name"`
	Category           string     `json:"category,omitempty"`
	Description        *string    `json:"description,omitempty" db:"description"`
	Email              string     `json:"email" db:"email"`
	Phone              *string    `json:"phone,omitempty" db:"phone"`
	VerificationStatus string     `json:"verification_status" db:"verification_status"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedBy         *uuid.UUID `json:"verified_by,omitempty" db:"verified_by"`
	RejectionReason    *string    `json:"rejection_reason,omitempty" db:"rejection_reason"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// ClubAdminProfile links a user to a club as admin
type ClubAdminProfile struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ClubID    uuid.UUID `json:"club_id" db:"club_id"`
	IsPrimary bool      `json:"is_primary" db:"is_primary"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Event represents a club event
type Event struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	ClubID               uuid.UUID `json:"club_id" db:"club_id"`
	Name                 string    `json:"name" db:"name"`
	ShortDescription     *string   `json:"short_description,omitempty" db:"short_description"`
	Description          *string   `json:"description,omitempty" db:"description"`
	EventDate            time.Time `json:"event_date" db:"event_date"`
	RegistrationDeadline time.Time `json:"registration_deadline" db:"registration_deadline"`
	Venue                *string   `json:"venue,omitempty" db:"venue"`
	ParticipationType    string    `json:"participation_type" db:"participation_type"`
	MinGroupSize         int       `json:"min_group_size" db:"min_group_size"`
	MaxGroupSize         int       `json:"max_group_size" db:"max_group_size"`
	MaxParticipants      *int      `json:"max_participants,omitempty" db:"max_participants"`
	RegistrationFee      float64   `json:"registration_fee" db:"registration_fee"`
	HasFee               bool      `json:"has_fee" db:"has_fee"`
	FeeInstructions      *string   `json:"fee_instructions,omitempty" db:"fee_instructions"`
	Rules                *string   `json:"rules,omitempty" db:"rules"`
	Status               string    `json:"status" db:"status"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
	// Computed/joined fields
	ClubName          string `json:"club_name,omitempty" db:"club_name"`
	Category          string `json:"category,omitempty"`
	RegistrationCount int    `json:"registration_count" db:"registration_count"`
	GroupCount        int    `json:"group_count,omitempty" db:"group_count"`
	IsRegistered      bool   `json:"is_registered,omitempty"`
}

// EventCustomField represents a custom registration question
type EventCustomField struct {
	ID           uuid.UUID `json:"id" db:"id"`
	EventID      uuid.UUID `json:"event_id" db:"event_id"`
	FieldName    string    `json:"field_name" db:"field_name"`
	FieldType    string    `json:"field_type" db:"field_type"`
	FieldOptions *string   `json:"field_options,omitempty" db:"field_options"`
	IsRequired   bool      `json:"is_required" db:"is_required"`
	SortOrder    int       `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Registration represents an event registration
type Registration struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	EventID          uuid.UUID  `json:"event_id" db:"event_id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	RegistrationType string     `json:"registration_type" db:"registration_type"`
	GroupID          *uuid.UUID `json:"group_id,omitempty" db:"group_id"`
	Status           string     `json:"status" db:"status"`
	CustomFieldsData *string    `json:"custom_fields_data,omitempty" db:"custom_fields_data"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	// Computed/joined fields
	EventName    string     `json:"event_name,omitempty" db:"event_name"`
	ClubName     string     `json:"club_name,omitempty" db:"club_name"`
	Venue        string     `json:"venue,omitempty"`
	EventDate    *time.Time `json:"event_date,omitempty" db:"event_date"`
	Deadline     *time.Time `json:"deadline,omitempty" db:"registration_deadline"`
	StudentName  string     `json:"student_name,omitempty"`
	StudentRegNo string     `json:"student_reg_no,omitempty"`
	StudentEmail string     `json:"student_email,omitempty"`
}

// RegistrationGroup represents a group registration
type RegistrationGroup struct {
	ID           uuid.UUID `json:"id" db:"id"`
	EventID      uuid.UUID `json:"event_id" db:"event_id"`
	GroupName    string    `json:"group_name" db:"group_name"`
	LeaderUserID uuid.UUID `json:"leader_user_id" db:"leader_user_id"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	// Joined fields
	Members    []GroupMember `json:"members,omitempty"`
	LeaderName string        `json:"leader_name,omitempty"`
}

// GroupMember represents a member of a registration group
type GroupMember struct {
	ID                       uuid.UUID  `json:"id" db:"id"`
	GroupID                  uuid.UUID  `json:"group_id" db:"group_id"`
	MemberName               string     `json:"member_name" db:"member_name"`
	MemberEmail              *string    `json:"member_email,omitempty" db:"member_email"`
	MemberPhone              *string    `json:"member_phone,omitempty" db:"member_phone"`
	MemberRegistrationNumber *string    `json:"member_registration_number,omitempty" db:"member_registration_number"`
	UserID                   *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	CustomFieldsData         *string    `json:"custom_fields_data,omitempty" db:"custom_fields_data"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
}

// Payment represents a payment record
type Payment struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	RegistrationID   *uuid.UUID `json:"registration_id,omitempty" db:"registration_id"`
	GroupID          *uuid.UUID `json:"group_id,omitempty" db:"group_id"`
	EventID          uuid.UUID  `json:"event_id" db:"event_id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	Amount           float64    `json:"amount" db:"amount"`
	Status           string     `json:"status" db:"status"`
	PaymentMethod    *string    `json:"payment_method,omitempty" db:"payment_method"`
	PaymentReference *string    `json:"payment_reference,omitempty" db:"payment_reference"`
	Notes            *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// Feedback represents user feedback
type Feedback struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Name       *string    `json:"name,omitempty" db:"name"`
	Email      *string    `json:"email,omitempty" db:"email"`
	Category   string     `json:"category" db:"category"`
	Subject    *string    `json:"subject,omitempty" db:"subject"`
	Message    string     `json:"message" db:"message"`
	Status     string     `json:"status" db:"status"`
	AdminNotes *string    `json:"admin_notes,omitempty" db:"admin_notes"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// AuditLog records administrative operations
type AuditLog struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ActorID    *uuid.UUID `json:"actor_id,omitempty" db:"actor_id"`
	Action     string     `json:"action" db:"action"`
	EntityType string     `json:"entity_type" db:"entity_type"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty" db:"entity_id"`
	Details    *string    `json:"details,omitempty" db:"details"`
	IPAddress  *string    `json:"ip_address,omitempty" db:"ip_address"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// Role constants
const (
	RoleStudent    = "student"
	RoleClubAdmin  = "club_admin"
	RoleSuperAdmin = "super_admin"
)

// Verification status constants
const (
	VerificationPending  = "pending"
	VerificationApproved = "approved"
	VerificationRejected = "rejected"
)

// Event status constants
const (
	EventDraft     = "draft"
	EventActive    = "active"
	EventCancelled = "cancelled"
	EventCompleted = "completed"
)

// Participation type constants
const (
	ParticipationIndividual = "individual"
	ParticipationGroup      = "group"
	ParticipationBoth       = "both"
)

// Registration status constants
const (
	RegistrationPending    = "pending"
	RegistrationConfirmed  = "confirmed"
	RegistrationCancelled  = "cancelled"
	RegistrationWaitlisted = "waitlisted"
)

// Payment status constants
const (
	PaymentPending   = "pending"
	PaymentCompleted = "completed"
	PaymentFailed    = "failed"
	PaymentRefunded  = "refunded"
)
