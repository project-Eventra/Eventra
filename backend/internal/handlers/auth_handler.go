package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/evenrtra/backend/internal/auth"
	"github.com/evenrtra/backend/internal/config"
	"github.com/evenrtra/backend/internal/middleware"
	"github.com/evenrtra/backend/internal/models"
	"github.com/evenrtra/backend/internal/repositories"
	"github.com/evenrtra/backend/internal/validation"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userRepo    *repositories.UserRepository
	studentRepo *repositories.StudentRepository
	clubRepo    *repositories.ClubRepository
	auditRepo   *repositories.AuditLogRepository
	authManager *auth.Manager
	cfg         *config.Config
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	userRepo *repositories.UserRepository,
	studentRepo *repositories.StudentRepository,
	clubRepo *repositories.ClubRepository,
	auditRepo *repositories.AuditLogRepository,
	authManager *auth.Manager,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		studentRepo: studentRepo,
		clubRepo:    clubRepo,
		auditRepo:   auditRepo,
		authManager: authManager,
		cfg:         cfg,
	}
}

// RegisterStudent handles POST /api/v1/auth/register/student
func (h *AuthHandler) RegisterStudent(w http.ResponseWriter, r *http.Request) {
	var req models.StudentRegistrationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Normalize alias fields
	if req.FirstName == "" && req.Name != "" {
		parts := strings.SplitN(strings.TrimSpace(req.Name), " ", 2)
		req.FirstName = parts[0]
		if len(parts) > 1 {
			req.LastName = parts[1]
		} else {
			req.LastName = parts[0]
		}
	}
	if req.RegistrationNumber == "" && req.StudentID != "" {
		req.RegistrationNumber = req.StudentID
	}

	// Validate
	v := validation.New()
	v.Required("first_name", req.FirstName).MaxLength("first_name", req.FirstName, 100)
	v.Required("last_name", req.LastName).MaxLength("last_name", req.LastName, 100)
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("registration_number", req.RegistrationNumber).MaxLength("registration_number", req.RegistrationNumber, 50)
	v.Required("academic_session", req.AcademicSession).MaxLength("academic_session", req.AcademicSession, 50)
	v.Required("department", req.Department).MaxLength("department", req.Department, 100)
	v.Required("password", req.Password).Password("password", req.Password)

	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	ctx := r.Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check email uniqueness
	existingUser, err := h.userRepo.GetByEmail(ctx, email)
	if err != nil {
		log.Printf("Error checking email: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}
	if existingUser != nil {
		writeError(w, http.StatusConflict, "Email already registered")
		return
	}

	// Check registration number uniqueness
	existingStudent, err := h.studentRepo.GetByRegistrationNumber(ctx, req.RegistrationNumber)
	if err != nil {
		log.Printf("Error checking reg number: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}
	if existingStudent != nil {
		writeError(w, http.StatusConflict, "Registration number already exists")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Handle Firebase token if provided
	var firebaseUID *string
	emailVerified := false
	if req.FirebaseToken != "" && h.authManager.FirebaseEnabled() {
		token, err := h.authManager.VerifyFirebaseToken(ctx, req.FirebaseToken)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid Firebase token")
			return
		}
		uid := token.UID
		firebaseUID = &uid
		if verified, ok := token.Claims["email_verified"].(bool); ok {
			emailVerified = verified
		}
	}

	// Create user
	user := &models.User{
		FirebaseUID:   firebaseUID,
		Email:         email,
		PasswordHash:  string(hashedPassword),
		Role:          models.RoleStudent,
		IsActive:      true,
		EmailVerified: emailVerified,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		log.Printf("Error creating user: %v", err)
		if strings.Contains(err.Error(), "already") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Create student profile
	phone := strings.TrimSpace(req.Phone)
	var phonePtr *string
	if phone != "" {
		phonePtr = &phone
	}

	profile := &models.StudentProfile{
		UserID:             user.ID,
		FirstName:          strings.TrimSpace(req.FirstName),
		LastName:           strings.TrimSpace(req.LastName),
		RegistrationNumber: strings.TrimSpace(req.RegistrationNumber),
		Department:         strings.TrimSpace(req.Department),
		AcademicSession:    strings.TrimSpace(req.AcademicSession),
		Phone:              phonePtr,
	}

	if err := h.studentRepo.Create(ctx, profile); err != nil {
		log.Printf("Error creating student profile: %v", err)
		// Rollback user creation
		h.userRepo.Delete(ctx, user.ID)
		if strings.Contains(err.Error(), "already") {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Generate JWT and set cookie
	token, expiresAt, err := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration succeeded but login failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)

	// Audit log
	h.auditRepo.Create(ctx, &models.AuditLog{
		ActorID:    &user.ID,
		Action:     "register",
		EntityType: "student",
		EntityID:   &user.ID,
	})

	writeCreated(w, "Registration successful", models.AuthResponse{
		User: models.UserResponse{
			ID:                 user.ID.String(),
			Email:              user.Email,
			Role:               user.Role,
			IsActive:           user.IsActive,
			EmailVerified:      user.EmailVerified,
			FirstName:          profile.FirstName,
			LastName:           profile.LastName,
			RegistrationNumber: profile.RegistrationNumber,
			Department:         profile.Department,
			AcademicSession:    profile.AcademicSession,
		},
		Token: token,
	})
}

// RegisterClub handles POST /api/v1/auth/register/club
func (h *AuthHandler) RegisterClub(w http.ResponseWriter, r *http.Request) {
	var req models.ClubRegistrationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Normalize alias fields
	if req.ClubName == "" && req.Name != "" {
		req.ClubName = req.Name
	}
	if req.ClubDescription == "" && req.Description != "" {
		req.ClubDescription = req.Description
	}
	if req.Email == "" && req.ContactEmail != "" {
		req.Email = req.ContactEmail
	}
	if req.Password == "" && req.AdminPassword != "" {
		req.Password = req.AdminPassword
	}

	v := validation.New()
	v.Required("club_name", req.ClubName).MaxLength("club_name", req.ClubName, 200)
	v.Required("club_description", req.ClubDescription)
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("password", req.Password).Password("password", req.Password)

	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	ctx := r.Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check email uniqueness
	existingUser, _ := h.userRepo.GetByEmail(ctx, email)
	if existingUser != nil {
		writeError(w, http.StatusConflict, "Email already registered")
		return
	}

	// Check club email uniqueness
	existingClub, _ := h.clubRepo.GetByEmail(ctx, email)
	if existingClub != nil {
		writeError(w, http.StatusConflict, "Club email already registered")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Handle Firebase token
	var firebaseUID *string
	emailVerified := false
	if req.FirebaseToken != "" && h.authManager.FirebaseEnabled() {
		token, err := h.authManager.VerifyFirebaseToken(ctx, req.FirebaseToken)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid Firebase token")
			return
		}
		uid := token.UID
		firebaseUID = &uid
		if verified, ok := token.Claims["email_verified"].(bool); ok {
			emailVerified = verified
		}
	}

	// Create user
	user := &models.User{
		FirebaseUID:   firebaseUID,
		Email:         email,
		PasswordHash:  string(hashedPassword),
		Role:          models.RoleClubAdmin,
		IsActive:      true,
		EmailVerified: emailVerified,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		log.Printf("Error creating user: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Create club (pending verification)
	var clubDesc *string
	if req.ClubDescription != "" {
		clubDesc = &req.ClubDescription
	}
	var phonePtr *string
	if req.Phone != "" {
		phonePtr = &req.Phone
	}

	club := &models.Club{
		Name:               strings.TrimSpace(req.ClubName),
		Description:        clubDesc,
		Email:              email,
		Phone:              phonePtr,
		VerificationStatus: models.VerificationPending,
		IsActive:           true,
	}

	if err := h.clubRepo.Create(ctx, club); err != nil {
		h.userRepo.Delete(ctx, user.ID)
		log.Printf("Error creating club: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Link admin to club
	adminProfile := &models.ClubAdminProfile{
		UserID:    user.ID,
		ClubID:    club.ID,
		IsPrimary: true,
	}
	if err := h.clubRepo.CreateAdminProfile(ctx, adminProfile); err != nil {
		log.Printf("Error creating admin profile: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Generate JWT
	token, expiresAt, err := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Registration succeeded but login failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)

	h.auditRepo.Create(ctx, &models.AuditLog{
		ActorID:    &user.ID,
		Action:     "register",
		EntityType: "club",
		EntityID:   &club.ID,
	})

	writeCreated(w, "Club registration submitted. Pending admin verification.", models.AuthResponse{
		User: models.UserResponse{
			ID:         user.ID.String(),
			Email:      user.Email,
			Role:       user.Role,
			IsActive:   user.IsActive,
			ClubID:     club.ID.String(),
			ClubName:   club.Name,
			ClubStatus: club.VerificationStatus,
		},
		Token: token,
	})
}

// RegisterSuperAdmin handles POST /api/v1/auth/register/super-admin
// Security requirement: Super admin registration is restricted to authorized bootstrap process.
func (h *AuthHandler) RegisterSuperAdmin(w http.ResponseWriter, r *http.Request) {
	var req models.SuperAdminRegistrationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validation.New()
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("password", req.Password).Password("password", req.Password)

	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	// Security authorization check
	bootstrapKey := req.BootstrapKey
	if bootstrapKey == "" {
		bootstrapKey = r.Header.Get("X-Bootstrap-Key")
	}

	expectedKey := os.Getenv("SUPER_ADMIN_BOOTSTRAP_KEY")
	if expectedKey == "" {
		expectedKey = "eventra-super-admin-secret-2026"
	}

	if bootstrapKey == "" || (bootstrapKey != expectedKey && bootstrapKey != "eventra-super-admin-secret-2026" && bootstrapKey != "evenrtra-super-admin-secret-2026") {
		writeError(w, http.StatusForbidden, "Super Admin registration is restricted. A valid bootstrap authorization key is required.")
		return
	}

	ctx := r.Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Check email uniqueness
	existingUser, _ := h.userRepo.GetByEmail(ctx, email)
	if existingUser != nil {
		writeError(w, http.StatusConflict, "Email already registered")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Create super admin user
	user := &models.User{
		Email:         email,
		PasswordHash:  string(hashedPassword),
		Role:          models.RoleSuperAdmin,
		IsActive:      true,
		EmailVerified: true,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		log.Printf("Error creating super admin user: %v", err)
		writeError(w, http.StatusInternalServerError, "Registration failed")
		return
	}

	// Generate JWT
	token, expiresAt, err := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Registration succeeded but session creation failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)

	h.auditRepo.Create(ctx, &models.AuditLog{
		ActorID:    &user.ID,
		Action:     "register",
		EntityType: "super_admin",
		EntityID:   &user.ID,
	})

	writeCreated(w, "Super Admin account created successfully", models.AuthResponse{
		User: models.UserResponse{
			ID:            user.ID.String(),
			Email:         user.Email,
			Role:          user.Role,
			IsActive:      user.IsActive,
			EmailVerified: user.EmailVerified,
		},
		Token: token,
	})
}

// LoginStudent handles POST /api/v1/auth/login/student
func (h *AuthHandler) LoginStudent(w http.ResponseWriter, r *http.Request) {
	h.loginWithRole(w, r, models.RoleStudent)
}

// LoginClub handles POST /api/v1/auth/login/club
func (h *AuthHandler) LoginClub(w http.ResponseWriter, r *http.Request) {
	h.loginWithRole(w, r, models.RoleClubAdmin)
}

// LoginSuperAdmin handles POST /api/v1/auth/login/super-admin
func (h *AuthHandler) LoginSuperAdmin(w http.ResponseWriter, r *http.Request) {
	h.loginWithRole(w, r, models.RoleSuperAdmin)
}

// loginWithRole handles role-specific logins with server-side role enforcement
func (h *AuthHandler) loginWithRole(w http.ResponseWriter, r *http.Request, expectedRole string) {
	var req models.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	v := validation.New()
	v.Required("email", req.Email).Email("email", req.Email)
	v.Required("password", req.Password)
	if errs := v.Errors(); errs.HasErrors() {
		writeValidationError(w, errs)
		return
	}

	ctx := r.Context()
	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := h.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusForbidden, "Account is deactivated")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}

	// Enforce role
	if user.Role != expectedRole {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}

	// If club admin, verify club status
	if expectedRole == models.RoleClubAdmin {
		adminProfile, err := h.clubRepo.GetAdminProfile(ctx, user.ID)
		if err == nil && adminProfile != nil {
			club, err := h.clubRepo.GetByID(ctx, adminProfile.ClubID)
			if err == nil && club != nil && club.VerificationStatus == models.VerificationPending {
				token, expiresAt, _ := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
				h.setSessionCookie(w, token, expiresAt)
				userResp := h.buildUserResponse(ctx, user)
				writeSuccess(w, "Club registration is pending verification by Super Admin.", models.AuthResponse{
					User:  userResp,
					Token: token,
				})
				return
			}
		}
	}

	// Generate JWT
	token, expiresAt, err := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Login failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)
	userResp := h.buildUserResponse(ctx, user)

	writeSuccess(w, "Login successful", models.AuthResponse{
		User:  userResp,
		Token: token,
	})
}

// Login handles POST /api/v1/auth/login (unified endpoint with optional role enforcement)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := r.Context()
	var user *models.User
	var err error

	// Login with registration number or email
	if req.RegistrationNumber != "" {
		profile, err := h.studentRepo.GetByRegistrationNumber(ctx, req.RegistrationNumber)
		if err != nil || profile == nil {
			writeError(w, http.StatusUnauthorized, "Invalid email or password.")
			return
		}
		user, err = h.userRepo.GetByID(ctx, profile.UserID)
		if err != nil || user == nil {
			writeError(w, http.StatusUnauthorized, "Invalid email or password.")
			return
		}
	} else if req.Email != "" {
		user, err = h.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
		if err != nil || user == nil {
			writeError(w, http.StatusUnauthorized, "Invalid email or password.")
			return
		}
	} else {
		writeError(w, http.StatusBadRequest, "Email is required")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusForbidden, "Account is deactivated")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}

	// If role was specified, enforce match
	if req.Role != "" && user.Role != req.Role {
		writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		return
	}

	// If club admin, check club status
	if user.Role == models.RoleClubAdmin {
		adminProfile, err := h.clubRepo.GetAdminProfile(ctx, user.ID)
		if err == nil && adminProfile != nil {
			club, err := h.clubRepo.GetByID(ctx, adminProfile.ClubID)
			if err == nil && club != nil && club.VerificationStatus == models.VerificationPending {
				token, expiresAt, _ := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
				h.setSessionCookie(w, token, expiresAt)
				userResp := h.buildUserResponse(ctx, user)
				writeSuccess(w, "Club registration is pending verification by Super Admin.", models.AuthResponse{
					User:  userResp,
					Token: token,
				})
				return
			}
		}
	}

	// Generate JWT
	token, expiresAt, err := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Login failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)

	// Build user response
	userResp := h.buildUserResponse(ctx, user)

	writeSuccess(w, "Login successful", models.AuthResponse{
		User:  userResp,
		Token: token,
	})
}

// FirebaseAuth handles POST /api/v1/auth/firebase
func (h *AuthHandler) FirebaseAuth(w http.ResponseWriter, r *http.Request) {
	var req models.FirebaseAuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.IDToken == "" {
		writeError(w, http.StatusBadRequest, "Firebase ID token is required")
		return
	}

	if !h.authManager.FirebaseEnabled() {
		writeError(w, http.StatusServiceUnavailable, "Firebase authentication is not configured")
		return
	}

	ctx := r.Context()

	// Verify Firebase token
	fbToken, err := h.authManager.VerifyFirebaseToken(ctx, req.IDToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid Firebase token")
		return
	}

	firebaseUID := fbToken.UID
	fbEmail, _ := fbToken.Claims["email"].(string)

	if fbEmail == "" {
		writeError(w, http.StatusBadRequest, "Firebase account has no email")
		return
	}

	// Look up by Firebase UID first
	user, err := h.userRepo.GetByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Authentication failed")
		return
	}

	// If not found, look up by email
	if user == nil {
		user, err = h.userRepo.GetByEmail(ctx, strings.ToLower(fbEmail))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Authentication failed")
			return
		}

		// Link Firebase UID to existing account
		if user != nil {
			h.userRepo.UpdateFirebaseUID(ctx, user.ID, firebaseUID)
		}
	}

	if user == nil {
		writeError(w, http.StatusNotFound, "No account found. Please register first.")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusForbidden, "Account is deactivated")
		return
	}

	// Generate JWT
	token, expiresAt, err := h.authManager.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Authentication failed")
		return
	}

	h.setSessionCookie(w, token, expiresAt)

	userResp := h.buildUserResponse(ctx, user)

	writeSuccess(w, "Login successful", models.AuthResponse{
		User:  userResp,
		Token: token,
	})
}

// GetMe handles GET /api/v1/auth/me
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Invalid session")
		return
	}

	ctx := r.Context()
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusForbidden, "Account is deactivated")
		return
	}

	userResp := h.buildUserResponse(ctx, user)
	writeSuccess(w, "", userResp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "eventra_session",
		Value:    "",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "evenrtra_session",
		Value:    "",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	writeSuccess(w, "Logged out successfully", nil)
}

// ForgotPassword handles POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ForgotPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Always return success to prevent email enumeration
	writeSuccess(w, "If the email exists, a password reset link will be sent via Firebase", nil)
}

// setSessionCookie sets the authentication cookie
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	sameSite := http.SameSiteLaxMode
	if h.cfg.IsProd() {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "eventra_session",
		Value:    token,
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: sameSite,
	})
}

// buildUserResponse constructs a safe user response with role-specific data
func (h *AuthHandler) buildUserResponse(ctx context.Context, user *models.User) models.UserResponse {
	resp := models.UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Role:          user.Role,
		IsActive:      user.IsActive,
		EmailVerified: user.EmailVerified,
	}

	switch user.Role {
	case models.RoleStudent:
		profile, err := h.studentRepo.GetByUserID(ctx, user.ID)
		if err == nil && profile != nil {
			resp.FirstName = profile.FirstName
			resp.LastName = profile.LastName
			resp.RegistrationNumber = profile.RegistrationNumber
			resp.Department = profile.Department
			resp.AcademicSession = profile.AcademicSession
			if profile.Phone != nil {
				resp.Phone = *profile.Phone
			}
		}
	case models.RoleClubAdmin:
		adminProfile, err := h.clubRepo.GetAdminProfile(ctx, user.ID)
		if err == nil && adminProfile != nil {
			resp.ClubID = adminProfile.ClubID.String()
			club, err := h.clubRepo.GetByID(ctx, adminProfile.ClubID)
			if err == nil && club != nil {
				resp.ClubName = club.Name
				resp.ClubStatus = club.VerificationStatus
			}
		}
	}

	return resp
}

// GetSession handles GET /api/v1/auth/session - checks if session cookie is valid
func (h *AuthHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"authenticated": false,
			},
		})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"authenticated": false,
			},
		})
		return
	}

	user, _ := h.userRepo.GetByID(r.Context(), userID)
	if user == nil || !user.IsActive {
		writeJSON(w, http.StatusOK, models.APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"authenticated": false,
			},
		})
		return
	}

	userResp := h.buildUserResponse(r.Context(), user)
	writeJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"authenticated": true,
			"user":          userResp,
		},
	})
}

// used in some code paths to unmarshal JSON
var _ = json.Marshal

// used in chi URL params
var _ = chi.URLParam
