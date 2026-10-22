package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/evenrtra/backend/internal/auth"
	"github.com/evenrtra/backend/internal/config"
	"github.com/evenrtra/backend/internal/middleware"
	"github.com/evenrtra/backend/internal/models"
)

// DemoStore provides an in-memory fallback when PostgreSQL is not running locally
type DemoStore struct {
	mu            sync.RWMutex
	users         map[string]*models.User
	students      map[uuid.UUID]*models.StudentProfile
	clubs         map[uuid.UUID]*models.Club
	events        map[uuid.UUID]*models.Event
	registrations map[uuid.UUID]*models.Registration
	feedback      []models.Feedback
}

var globalDemoStore *DemoStore
var onceDemo sync.Once

func GetDemoStore() *DemoStore {
	onceDemo.Do(func() {
		s := &DemoStore{
			users:         make(map[string]*models.User),
			students:      make(map[uuid.UUID]*models.StudentProfile),
			clubs:         make(map[uuid.UUID]*models.Club),
			events:        make(map[uuid.UUID]*models.Event),
			registrations: make(map[uuid.UUID]*models.Registration),
			feedback:      make([]models.Feedback, 0),
		}

		// Seed Super Admin
		adminID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		adminHash, _ := bcrypt.GenerateFromPassword([]byte("Admin@123456"), bcrypt.DefaultCost)
		adminUser := &models.User{
			ID:           adminID,
			Email:        "admin@eventra.edu",
			PasswordHash: string(adminHash),
			Role:         models.RoleSuperAdmin,
			IsActive:     true,
		}
		s.users["admin@eventra.edu"] = adminUser
		s.users["admin@evenrtra.edu"] = adminUser

		// Seed Student
		studentID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
		studentHash, _ := bcrypt.GenerateFromPassword([]byte("Student@123456"), bcrypt.DefaultCost)
		studentUser := &models.User{
			ID:           studentID,
			Email:        "student@eventra.edu",
			PasswordHash: string(studentHash),
			Role:         models.RoleStudent,
			IsActive:     true,
		}
		s.users["student@eventra.edu"] = studentUser
		s.users["student@evenrtra.edu"] = studentUser
		s.students[studentID] = &models.StudentProfile{
			UserID:             studentID,
			FirstName:          "Alex",
			LastName:           "Chen",
			RegistrationNumber: "STU-2026-001",
			Department:         "Computer Science & Engineering",
			AcademicSession:    "2024-2028",
		}

		// Initial clean state: 0 clubs, 0 events, 0 registrations until registered by users
		globalDemoStore = s
	})
	return globalDemoStore
}

// RegisterDemoRoutes registers in-memory endpoints so user can test immediately without PostgreSQL
func RegisterDemoRoutes(r chi.Router, authManager *auth.Manager, cfg *config.Config) {
	store := GetDemoStore()

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
			"app":    "Eventra",
			"mode":   "local_in_memory_ready",
			"note":   "PostgreSQL not connected locally; running in interactive demo mode",
		})
	})

	// Public Platform Real-Time Stats
	r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		defer store.mu.RUnlock()
		writeSuccess(w, "", map[string]interface{}{
			"total_clubs":         len(store.clubs),
			"total_students":      len(store.students),
			"total_events":        len(store.events),
			"total_registrations": len(store.registrations),
		})
	})

	// Public Events
	r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		defer store.mu.RUnlock()

		search := strings.ToLower(r.URL.Query().Get("search"))
		var list []models.Event
		for _, ev := range store.events {
			if search != "" && !strings.Contains(strings.ToLower(ev.Name), search) {
				continue
			}
			list = append(list, *ev)
		}
		writeSuccess(w, "", models.PaginatedResponse{
			Items:    list,
			Total:    len(list),
			Page:     1,
			PageSize: 20,
		})
	})

	r.Get("/events/timeline", func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		defer store.mu.RUnlock()

		var timeline []map[string]interface{}
		for _, ev := range store.events {
			venue := "Main Campus"
			if ev.Venue != nil {
				venue = *ev.Venue
			}
			timeline = append(timeline, map[string]interface{}{
				"id":        ev.ID,
				"title":     ev.Name,
				"club_name": ev.ClubName,
				"date":      ev.EventDate.Format("2006-01-02"),
				"time":      ev.EventDate.Format("03:04 PM"),
				"venue":     venue,
			})
		}
		writeSuccess(w, "", timeline)
	})

	r.Get("/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid event ID")
			return
		}

		store.mu.RLock()
		ev, exists := store.events[id]
		store.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, "Event not found")
			return
		}

		claims := middleware.GetClaims(r.Context())
		if claims != nil {
			uID, _ := uuid.Parse(claims.UserID)
			store.mu.RLock()
			for _, reg := range store.registrations {
				if reg.EventID == id && reg.UserID == uID {
					ev.IsRegistered = true
					break
				}
			}
			store.mu.RUnlock()
		}

		writeSuccess(w, "", ev)
	})

	// Public Clubs
	r.Get("/clubs", func(w http.ResponseWriter, r *http.Request) {
		store.mu.RLock()
		defer store.mu.RUnlock()

		var list []models.Club
		for _, c := range store.clubs {
			list = append(list, *c)
		}
		writeSuccess(w, "", list)
	})

	// Public Feedback
	r.Post("/feedback", func(w http.ResponseWriter, r *http.Request) {
		var req models.FeedbackRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		store.mu.Lock()
		var emailPtr *string
		if req.Email != "" {
			emailPtr = &req.Email
		}
		store.feedback = append(store.feedback, models.Feedback{
			ID:        uuid.New(),
			Category:  req.Category,
			Message:   req.Message,
			Email:     emailPtr,
			Status:    "new",
			CreatedAt: time.Now(),
		})
		store.mu.Unlock()
		writeSuccess(w, "Feedback received", nil)
	})

	// Auth Endpoints
	r.Route("/auth", func(authRouter chi.Router) {
		// Unified login
		authRouter.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			var req models.LoginRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			store.mu.RLock()
			email := strings.ToLower(strings.TrimSpace(req.Email))
			user, exists := store.users[email]
			store.mu.RUnlock()

			if !exists {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			if req.Role != "" && user.Role != req.Role {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			token, expiresAt, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)

			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			name := user.Email
			var studentID, dept, clubName, clubStatus string
			store.mu.RLock()
			if st, ok := store.students[user.ID]; ok {
				name = st.FirstName + " " + st.LastName
				studentID = st.RegistrationNumber
				dept = st.Department
			}
			for _, cl := range store.clubs {
				if strings.EqualFold(cl.Email, user.Email) {
					clubName = cl.Name
					clubStatus = cl.VerificationStatus
					break
				}
			}
			store.mu.RUnlock()

			msg := "Login successful"
			if user.Role == models.RoleClubAdmin && clubStatus == "pending" {
				msg = "Club registration is pending verification by Super Admin."
			}

			writeSuccess(w, msg, map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":          user.ID,
					"email":       user.Email,
					"role":        user.Role,
					"name":        name,
					"student_id":  studentID,
					"department":  dept,
					"club_name":   clubName,
					"club_status": clubStatus,
				},
			})
		})

		// Role-specific login endpoints
		authRouter.Post("/login/student", func(w http.ResponseWriter, r *http.Request) {
			var req models.LoginRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			store.mu.RLock()
			email := strings.ToLower(strings.TrimSpace(req.Email))
			user, exists := store.users[email]
			store.mu.RUnlock()

			if !exists || user.Role != models.RoleStudent {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			token, expiresAt, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			name := user.Email
			var studentID, dept string
			store.mu.RLock()
			if st, ok := store.students[user.ID]; ok {
				name = st.FirstName + " " + st.LastName
				studentID = st.RegistrationNumber
				dept = st.Department
			}
			store.mu.RUnlock()

			writeSuccess(w, "Login successful", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":         user.ID,
					"email":      user.Email,
					"role":       user.Role,
					"name":       name,
					"student_id": studentID,
					"department": dept,
				},
			})
		})

		authRouter.Post("/login/club", func(w http.ResponseWriter, r *http.Request) {
			var req models.LoginRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			store.mu.RLock()
			email := strings.ToLower(strings.TrimSpace(req.Email))
			user, exists := store.users[email]
			store.mu.RUnlock()

			if !exists || user.Role != models.RoleClubAdmin {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			token, expiresAt, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			var clubName, clubStatus string
			store.mu.RLock()
			for _, cl := range store.clubs {
				if strings.EqualFold(cl.Email, user.Email) {
					clubName = cl.Name
					clubStatus = cl.VerificationStatus
					break
				}
			}
			store.mu.RUnlock()

			msg := "Login successful"
			if clubStatus == "pending" {
				msg = "Club registration is pending verification by Super Admin."
			}

			writeSuccess(w, msg, map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":          user.ID,
					"email":       user.Email,
					"role":        user.Role,
					"name":        clubName,
					"club_name":   clubName,
					"club_status": clubStatus,
				},
			})
		})

		authRouter.Post("/login/super-admin", func(w http.ResponseWriter, r *http.Request) {
			var req models.LoginRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			store.mu.RLock()
			email := strings.ToLower(strings.TrimSpace(req.Email))
			user, exists := store.users[email]
			store.mu.RUnlock()

			if !exists || user.Role != models.RoleSuperAdmin {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid email or password.")
				return
			}

			token, expiresAt, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			writeSuccess(w, "Login successful", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":    user.ID,
					"email": user.Email,
					"role":  user.Role,
					"name":  "Super Admin",
				},
			})
		})

		authRouter.Post("/register/student", func(w http.ResponseWriter, r *http.Request) {
			var req models.StudentRegistrationRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			email := strings.ToLower(strings.TrimSpace(req.Email))
			if req.FirstName == "" || req.LastName == "" || email == "" || req.RegistrationNumber == "" || req.AcademicSession == "" || req.Department == "" || req.Password == "" {
				writeError(w, http.StatusBadRequest, "All fields are required")
				return
			}
			if len(req.Password) < 8 {
				writeError(w, http.StatusBadRequest, "Password must be at least 8 characters")
				return
			}

			store.mu.Lock()
			if _, exists := store.users[email]; exists {
				store.mu.Unlock()
				writeError(w, http.StatusConflict, "Email already registered")
				return
			}

			for _, st := range store.students {
				if strings.EqualFold(st.RegistrationNumber, req.RegistrationNumber) {
					store.mu.Unlock()
					writeError(w, http.StatusConflict, "Registration number already exists")
					return
				}
			}

			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			newID := uuid.New()
			user := &models.User{
				ID:           newID,
				Email:        email,
				PasswordHash: string(hash),
				Role:         models.RoleStudent,
				IsActive:     true,
			}
			store.users[email] = user

			store.students[newID] = &models.StudentProfile{
				UserID:             newID,
				FirstName:          req.FirstName,
				LastName:           req.LastName,
				RegistrationNumber: req.RegistrationNumber,
				Department:         req.Department,
				AcademicSession:    req.AcademicSession,
			}
			store.mu.Unlock()

			token, expiresAt, _ := authManager.GenerateJWT(newID, email, models.RoleStudent)
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
			})

			writeCreated(w, "Student account created successfully", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":                  newID,
					"email":               email,
					"role":                models.RoleStudent,
					"name":                req.FirstName + " " + req.LastName,
					"first_name":          req.FirstName,
					"last_name":           req.LastName,
					"registration_number": req.RegistrationNumber,
					"department":          req.Department,
					"academic_session":    req.AcademicSession,
				},
			})
		})

		authRouter.Post("/register/club", func(w http.ResponseWriter, r *http.Request) {
			var req models.ClubRegistrationRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			cName := strings.TrimSpace(req.ClubName)
			if cName == "" && req.Name != "" {
				cName = req.Name
			}
			cDesc := strings.TrimSpace(req.ClubDescription)
			if cDesc == "" && req.Description != "" {
				cDesc = req.Description
			}
			email := strings.ToLower(strings.TrimSpace(req.Email))
			if email == "" && req.ContactEmail != "" {
				email = req.ContactEmail
			}
			pass := req.Password
			if pass == "" && req.AdminPassword != "" {
				pass = req.AdminPassword
			}

			if cName == "" || cDesc == "" || email == "" || pass == "" {
				writeError(w, http.StatusBadRequest, "All fields are required")
				return
			}
			if len(pass) < 8 {
				writeError(w, http.StatusBadRequest, "Password must be at least 8 characters")
				return
			}

			store.mu.Lock()
			if _, exists := store.users[email]; exists {
				store.mu.Unlock()
				writeError(w, http.StatusConflict, "Email already registered")
				return
			}

			newUID := uuid.New()
			newCID := uuid.New()
			hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
			user := &models.User{
				ID:           newUID,
				Email:        email,
				PasswordHash: string(hash),
				Role:         models.RoleClubAdmin,
				IsActive:     true,
			}
			store.users[email] = user
			store.clubs[newCID] = &models.Club{
				ID:                 newCID,
				Name:               cName,
				Description:        &cDesc,
				Email:              email,
				VerificationStatus: "pending",
				IsActive:           true,
			}
			store.mu.Unlock()

			token, expiresAt, _ := authManager.GenerateJWT(newUID, email, models.RoleClubAdmin)
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
			})

			writeCreated(w, "Club registration submitted. Pending admin verification.", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":          newUID,
					"email":       email,
					"role":        models.RoleClubAdmin,
					"name":        cName,
					"club_name":   cName,
					"club_status": "pending",
				},
			})
		})

		authRouter.Post("/register/super-admin", func(w http.ResponseWriter, r *http.Request) {
			var req models.SuperAdminRegistrationRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			bootstrapKey := req.BootstrapKey
			if bootstrapKey == "" {
				bootstrapKey = r.Header.Get("X-Bootstrap-Key")
			}

			expectedKey := "eventra-super-admin-secret-2026"
			if bootstrapKey == "" || (bootstrapKey != expectedKey && bootstrapKey != "evenrtra-super-admin-secret-2026") {
				writeError(w, http.StatusForbidden, "Super Admin registration is restricted. A valid bootstrap authorization key is required.")
				return
			}

			email := strings.ToLower(strings.TrimSpace(req.Email))
			if email == "" || req.Password == "" {
				writeError(w, http.StatusBadRequest, "Email and password are required")
				return
			}
			if len(req.Password) < 8 {
				writeError(w, http.StatusBadRequest, "Password must be at least 8 characters")
				return
			}

			store.mu.Lock()
			if _, exists := store.users[email]; exists {
				store.mu.Unlock()
				writeError(w, http.StatusConflict, "Email already registered")
				return
			}

			newUID := uuid.New()
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			user := &models.User{
				ID:           newUID,
				Email:        email,
				PasswordHash: string(hash),
				Role:         models.RoleSuperAdmin,
				IsActive:     true,
			}
			store.users[email] = user
			store.mu.Unlock()

			token, expiresAt, _ := authManager.GenerateJWT(newUID, email, models.RoleSuperAdmin)
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
			})

			writeCreated(w, "Super Admin account created successfully", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":    newUID,
					"email": email,
					"role":  models.RoleSuperAdmin,
					"name":  "Super Admin",
				},
			})
		})

		authRouter.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{
				Name:     "eventra_session",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			writeSuccess(w, "Logged out", nil)
		})
	})

	// Student Registrations Protected
	r.Route("/student", func(studentRouter chi.Router) {
		studentRouter.Use(middleware.AuthMiddleware(authManager))
		studentRouter.Get("/registrations", func(w http.ResponseWriter, r *http.Request) {
			claims := middleware.GetClaims(r.Context())
			uID, _ := uuid.Parse(claims.UserID)

			store.mu.RLock()
			var list []models.Registration
			for _, reg := range store.registrations {
				if reg.UserID == uID {
					list = append(list, *reg)
				}
			}
			store.mu.RUnlock()

			writeSuccess(w, "", list)
		})
	})

	// Event Register Protected
	r.Route("/events/{id}", func(eventRouter chi.Router) {
		eventRouter.Use(middleware.AuthMiddleware(authManager))

		eventRouter.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			claims := middleware.GetClaims(r.Context())
			uID, _ := uuid.Parse(claims.UserID)
			eID, _ := uuid.Parse(chi.URLParam(r, "id"))

			var req models.CreateRegistrationRequest
			decodeJSON(r, &req)

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

			store.mu.Lock()
			defer store.mu.Unlock()

			ev, ok := store.events[eID]
			if !ok {
				writeError(w, http.StatusNotFound, "Event not found")
				return
			}

			newRID := uuid.New()
			venue := "Campus Auditorium"
			if ev.Venue != nil {
				venue = *ev.Venue
			}

			studentName := claims.Email
			studentRegNo := "STU-" + newRID.String()[:8]
			department := "Campus Department"
			phone := ""
			if st, exists := store.students[uID]; exists {
				studentName = strings.TrimSpace(st.FirstName + " " + st.LastName)
				if studentName == "" {
					studentName = claims.Email
				}
				if st.RegistrationNumber != "" {
					studentRegNo = st.RegistrationNumber
				}
				department = st.Department
				if st.Phone != nil {
					phone = *st.Phone
				}
			}

			reg := &models.Registration{
				ID:               newRID,
				EventID:          eID,
				UserID:           uID,
				EventName:        ev.Name,
				ClubName:         ev.ClubName,
				Venue:            venue,
				RegistrationType: "individual",
				Status:           "confirmed",
				CustomFieldsData: customData,
				StudentName:      studentName,
				StudentRegNo:     studentRegNo,
				StudentID:        studentRegNo,
				StudentEmail:     claims.Email,
				Email:            claims.Email,
				Department:       department,
				Phone:            phone,
				CreatedAt:        time.Now(),
			}

			store.registrations[newRID] = reg
			ev.RegistrationCount++

			writeSuccess(w, "Registration confirmed", map[string]interface{}{
				"registration_id": newRID,
				"status":          "confirmed",
			})
		})

		eventRouter.Post("/register-group", func(w http.ResponseWriter, r *http.Request) {
			claims := middleware.GetClaims(r.Context())
			uID, _ := uuid.Parse(claims.UserID)
			eID, _ := uuid.Parse(chi.URLParam(r, "id"))

			var req models.CreateGroupRegistrationRequest
			decodeJSON(r, &req)

			teamName := req.GroupName
			if teamName == "" {
				teamName = req.TeamName
			}
			if teamName == "" {
				teamName = "Team " + uuid.New().String()[:6]
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

			store.mu.Lock()
			defer store.mu.Unlock()

			ev, ok := store.events[eID]
			if !ok {
				writeError(w, http.StatusNotFound, "Event not found")
				return
			}

			newRID := uuid.New()
			venue := "Campus Auditorium"
			if ev.Venue != nil {
				venue = *ev.Venue
			}

			studentName := claims.Email
			studentRegNo := "STU-" + newRID.String()[:8]
			department := "Campus Department"
			phone := ""
			if st, exists := store.students[uID]; exists {
				studentName = strings.TrimSpace(st.FirstName + " " + st.LastName)
				if studentName == "" {
					studentName = claims.Email
				}
				if st.RegistrationNumber != "" {
					studentRegNo = st.RegistrationNumber
				}
				department = st.Department
				if st.Phone != nil {
					phone = *st.Phone
				}
			}

			reg := &models.Registration{
				ID:               newRID,
				EventID:          eID,
				UserID:           uID,
				EventName:        ev.Name,
				ClubName:         ev.ClubName,
				Venue:            venue,
				RegistrationType: "group",
				TeamName:         teamName,
				Status:           "confirmed",
				CustomFieldsData: customData,
				StudentName:      studentName,
				StudentRegNo:     studentRegNo,
				StudentID:        studentRegNo,
				StudentEmail:     claims.Email,
				Email:            claims.Email,
				Department:       department,
				Phone:            phone,
				CreatedAt:        time.Now(),
			}

			store.registrations[newRID] = reg
			ev.RegistrationCount++

			writeSuccess(w, "Group registration confirmed", map[string]interface{}{
				"registration_id": newRID,
				"team_name":       teamName,
				"status":          "confirmed",
			})
		})
	})

	// Club Admin Endpoints
	r.Route("/club-admin", func(ca chi.Router) {
		ca.Use(middleware.AuthMiddleware(authManager))
		ca.Get("/events", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			var list []models.Event
			for _, ev := range store.events {
				list = append(list, *ev)
			}
			store.mu.RUnlock()
			writeSuccess(w, "", list)
		})
		ca.Post("/events", func(w http.ResponseWriter, r *http.Request) {
			var ev models.Event
			decodeJSON(r, &ev)
			ev.ID = uuid.New()
			ev.RegistrationCount = 0
			ev.Status = "active"

			store.mu.Lock()
			store.events[ev.ID] = &ev
			store.mu.Unlock()

			writeCreated(w, "Event published", ev)
		})
		ca.Get("/events/{id}/registrations", func(w http.ResponseWriter, r *http.Request) {
			eID, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid event ID")
				return
			}
			store.mu.RLock()
			ev := store.events[eID]
			var list []models.Registration
			for _, reg := range store.registrations {
				if reg.EventID == eID {
					list = append(list, *reg)
				}
			}
			store.mu.RUnlock()

			var cFields []models.EventCustomField
			if ev != nil && ev.CustomFields != nil {
				cFields = ev.CustomFields
			} else {
				cFields = []models.EventCustomField{}
			}

			writeSuccess(w, "", map[string]interface{}{
				"registrations": models.PaginatedResponse{
					Items:      list,
					Total:      len(list),
					Page:       1,
					PageSize:   50,
					TotalPages: 1,
				},
				"items":         list,
				"groups":        []interface{}{},
				"custom_fields": cFields,
				"event":         ev,
			})
		})
		ca.Post("/events/{id}/extend-deadline", func(w http.ResponseWriter, r *http.Request) {
			eID, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid event ID")
				return
			}
			var req models.ExtendDeadlineRequest
			decodeJSON(r, &req)

			store.mu.Lock()
			ev, ok := store.events[eID]
			if ok && req.NewDeadline != "" {
				t, parseErr := time.Parse(time.RFC3339, req.NewDeadline)
				if parseErr == nil {
					ev.RegistrationDeadline = t
				}
			}
			store.mu.Unlock()

			writeSuccess(w, "Deadline extended successfully", nil)
		})
		ca.Delete("/events/{id}", func(w http.ResponseWriter, r *http.Request) {
			eID, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid event ID")
				return
			}
			store.mu.Lock()
			delete(store.events, eID)
			store.mu.Unlock()

			writeSuccess(w, "Event deleted", nil)
		})
	})

	// Super Admin Stats
	r.Route("/admin", func(adminRouter chi.Router) {
		adminRouter.Use(middleware.AuthMiddleware(authManager))
		adminRouter.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			defer store.mu.RUnlock()
			writeSuccess(w, "", map[string]interface{}{
				"total_clubs":         len(store.clubs),
				"total_students":      len(store.students),
				"total_events":        len(store.events),
				"total_registrations": len(store.registrations),
			})
		})
		adminRouter.Get("/clubs", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			defer store.mu.RUnlock()
			var list []map[string]interface{}
			for _, c := range store.clubs {
				list = append(list, map[string]interface{}{
					"id":                  c.ID,
					"name":                c.Name,
					"category":            c.Category,
					"contact_email":       c.Email,
					"email":               c.Email,
					"verification_status": c.VerificationStatus,
					"is_approved":         c.VerificationStatus == "approved",
					"created_at":          c.CreatedAt,
				})
			}
			writeSuccess(w, "", list)
		})

		adminRouter.Post("/clubs/{id}/approve", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid club ID")
				return
			}

			store.mu.Lock()
			defer store.mu.Unlock()
			c, exists := store.clubs[id]
			if !exists {
				writeError(w, http.StatusNotFound, "Club not found")
				return
			}

			c.VerificationStatus = "approved"
			writeSuccess(w, "Club application approved successfully", nil)
		})

		adminRouter.Post("/clubs/{id}/reject", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid club ID")
				return
			}

			store.mu.Lock()
			defer store.mu.Unlock()
			c, exists := store.clubs[id]
			if !exists {
				writeError(w, http.StatusNotFound, "Club not found")
				return
			}

			c.VerificationStatus = "rejected"
			writeSuccess(w, "Club application rejected", nil)
		})

		adminRouter.Delete("/clubs/{id}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid club ID")
				return
			}

			store.mu.Lock()
			defer store.mu.Unlock()
			delete(store.clubs, id)
			writeSuccess(w, "Club removed successfully", nil)
		})

		adminRouter.Get("/students", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			defer store.mu.RUnlock()
			var list []map[string]interface{}
			for _, s := range store.students {
				email := ""
				for em, u := range store.users {
					if u.ID == s.UserID {
						email = em
						break
					}
				}
				list = append(list, map[string]interface{}{
					"id":         s.UserID,
					"name":       s.FirstName + " " + s.LastName,
					"user_name":  s.FirstName + " " + s.LastName,
					"student_id": s.RegistrationNumber,
					"department": s.Department,
					"email":      email,
					"created_at": time.Now(),
				})
			}
			writeSuccess(w, "", list)
		})

		adminRouter.Delete("/students/{id}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid student ID")
				return
			}

			store.mu.Lock()
			defer store.mu.Unlock()
			delete(store.students, id)
			for em, u := range store.users {
				if u.ID == id {
					delete(store.users, em)
					break
				}
			}
			writeSuccess(w, "Student removed successfully", nil)
		})

		adminRouter.Get("/events", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			defer store.mu.RUnlock()
			var list []models.Event
			for _, e := range store.events {
				list = append(list, *e)
			}
			writeSuccess(w, "", list)
		})

		adminRouter.Delete("/events/{id}", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := uuid.Parse(idStr)
			if err != nil {
				writeError(w, http.StatusBadRequest, "Invalid event ID")
				return
			}

			store.mu.Lock()
			defer store.mu.Unlock()
			delete(store.events, id)
			writeSuccess(w, "Event removed successfully", nil)
		})

		adminRouter.Get("/feedback", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			defer store.mu.RUnlock()
			writeSuccess(w, "", store.feedback)
		})
	})
}
