package handlers

import (
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
		s.users["admin@evenrtra.edu"] = &models.User{
			ID:           adminID,
			Email:        "admin@evenrtra.edu",
			PasswordHash: string(adminHash),
			Role:         models.RoleSuperAdmin,
			IsActive:     true,
		}

		// Seed Student
		studentID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
		studentHash, _ := bcrypt.GenerateFromPassword([]byte("Student@123456"), bcrypt.DefaultCost)
		s.users["student@evenrtra.edu"] = &models.User{
			ID:           studentID,
			Email:        "student@evenrtra.edu",
			PasswordHash: string(studentHash),
			Role:         models.RoleStudent,
			IsActive:     true,
		}
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
				writeError(w, http.StatusUnauthorized, "Invalid email or password")
				return
			}

			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				writeError(w, http.StatusUnauthorized, "Invalid email or password")
				return
			}

			token, expiresAt, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)

			// Cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "evenrtra_session",
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

		// Google / Gmail Sign-In & Instant Registration
		authRouter.Post("/google", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			}
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			email := strings.ToLower(strings.TrimSpace(req.Email))
			if email == "" {
				writeError(w, http.StatusBadRequest, "Gmail address is required")
				return
			}

			store.mu.Lock()
			user, exists := store.users[email]
			if !exists {
				// Auto-create student account with Google
				newID := uuid.New()
				user = &models.User{
					ID:           newID,
					Email:        email,
					PasswordHash: "google_authenticated",
					Role:         models.RoleStudent,
					IsActive:     true,
				}
				store.users[email] = user

				firstName := "Student"
				lastName := ""
				if req.Name != "" {
					parts := strings.SplitN(strings.TrimSpace(req.Name), " ", 2)
					firstName = parts[0]
					if len(parts) > 1 {
						lastName = parts[1]
					}
				} else {
					firstName = strings.Split(email, "@")[0]
				}

				store.students[newID] = &models.StudentProfile{
					UserID:             newID,
					FirstName:          firstName,
					LastName:           lastName,
					RegistrationNumber: "GML-" + strings.ToUpper(newID.String()[:8]),
					Department:         "General Studies",
					AcademicSession:    "2024-2028",
				}
			}
			store.mu.Unlock()

			token, expiresAt, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)

			http.SetCookie(w, &http.Cookie{
				Name:     "evenrtra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			displayName := user.Email
			var studentID, dept string
			store.mu.RLock()
			if st, ok := store.students[user.ID]; ok {
				displayName = st.FirstName
				if st.LastName != "" {
					displayName += " " + st.LastName
				}
				studentID = st.RegistrationNumber
				dept = st.Department
			}
			store.mu.RUnlock()

			writeSuccess(w, "Google sign-in successful", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":         user.ID,
					"email":      user.Email,
					"role":       user.Role,
					"name":       displayName,
					"student_id": studentID,
					"department": dept,
				},
			})
		})

		// Firebase endpoint alias
		authRouter.Post("/firebase", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			}
			decodeJSON(r, &req)
			if req.Email == "" {
				req.Email = "student@gmail.com"
			}
			// Forward to Google auth handler logic
			email := strings.ToLower(strings.TrimSpace(req.Email))
			store.mu.Lock()
			user, exists := store.users[email]
			if !exists {
				newID := uuid.New()
				user = &models.User{
					ID:           newID,
					Email:        email,
					PasswordHash: "google_authenticated",
					Role:         models.RoleStudent,
					IsActive:     true,
				}
				store.users[email] = user
				store.students[newID] = &models.StudentProfile{
					UserID:             newID,
					FirstName:          "Google",
					LastName:           "User",
					RegistrationNumber: "GML-" + strings.ToUpper(newID.String()[:8]),
					Department:         "General Studies",
					AcademicSession:    "2024-2028",
				}
			}
			store.mu.Unlock()

			token, _, _ := authManager.GenerateJWT(user.ID, user.Email, user.Role)
			writeSuccess(w, "Authenticated via Google", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":    user.ID,
					"email": user.Email,
					"role":  user.Role,
					"name":  user.Email,
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
			if email == "" || req.Password == "" {
				writeError(w, http.StatusBadRequest, "Email and password are required")
				return
			}

			store.mu.Lock()
			if _, exists := store.users[email]; exists {
				store.mu.Unlock()
				writeError(w, http.StatusConflict, "Email already registered")
				return
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

			firstName := req.FirstName
			lastName := req.LastName
			if firstName == "" && req.Name != "" {
				parts := strings.SplitN(req.Name, " ", 2)
				firstName = parts[0]
				if len(parts) > 1 {
					lastName = parts[1]
				} else {
					lastName = parts[0]
				}
			}
			regNo := req.RegistrationNumber
			if regNo == "" && req.StudentID != "" {
				regNo = req.StudentID
			}

			store.students[newID] = &models.StudentProfile{
				UserID:             newID,
				FirstName:          firstName,
				LastName:           lastName,
				RegistrationNumber: regNo,
				Department:         req.Department,
				AcademicSession:    "2024-2028",
			}
			store.mu.Unlock()

			token, expiresAt, _ := authManager.GenerateJWT(newID, email, models.RoleStudent)
			http.SetCookie(w, &http.Cookie{
				Name:     "evenrtra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
			})

			writeCreated(w, "Account created", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":         newID,
					"email":      email,
					"role":       models.RoleStudent,
					"name":       firstName + " " + lastName,
					"student_id": regNo,
					"department": req.Department,
				},
			})
		})

		authRouter.Post("/register/club", func(w http.ResponseWriter, r *http.Request) {
			var req models.ClubRegistrationRequest
			if err := decodeJSON(r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Invalid request body")
				return
			}

			email := req.Email
			if email == "" && req.ContactEmail != "" {
				email = req.ContactEmail
			}
			pass := req.Password
			if pass == "" && req.AdminPassword != "" {
				pass = req.AdminPassword
			}
			cName := req.ClubName
			if cName == "" && req.Name != "" {
				cName = req.Name
			}

			store.mu.Lock()
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
			var descPtr *string
			if req.Description != "" {
				descPtr = &req.Description
			}
			store.clubs[newCID] = &models.Club{
				ID:                 newCID,
				Name:               cName,
				Category:           req.Category,
				Description:        descPtr,
				Email:              email,
				VerificationStatus: "approved",
				IsActive:           true,
			}
			store.mu.Unlock()

			token, expiresAt, _ := authManager.GenerateJWT(newUID, email, models.RoleClubAdmin)
			http.SetCookie(w, &http.Cookie{
				Name:     "evenrtra_session",
				Value:    token,
				Path:     "/",
				Expires:  expiresAt,
				HttpOnly: true,
			})

			writeCreated(w, "Club application registered", map[string]interface{}{
				"token": token,
				"user": map[string]interface{}{
					"id":        newUID,
					"email":     email,
					"role":      models.RoleClubAdmin,
					"name":      cName,
					"club_name": cName,
				},
			})
		})

		authRouter.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{
				Name:     "evenrtra_session",
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
	r.With(middleware.AuthMiddleware(authManager)).Post("/events/{id}/register", func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.GetClaims(r.Context())
		uID, _ := uuid.Parse(claims.UserID)
		eID, _ := uuid.Parse(chi.URLParam(r, "id"))

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

		store.registrations[newRID] = &models.Registration{
			ID:               newRID,
			EventID:          eID,
			UserID:           uID,
			EventName:        ev.Name,
			ClubName:         ev.ClubName,
			Venue:            venue,
			RegistrationType: "individual",
			Status:           "confirmed",
			CreatedAt:        time.Now(),
		}
		ev.RegistrationCount++

		writeSuccess(w, "Registration confirmed", map[string]interface{}{
			"registration_id": newRID,
			"status":          "confirmed",
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
			var list []models.Club
			for _, c := range store.clubs {
				list = append(list, *c)
			}
			writeSuccess(w, "", list)
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
		adminRouter.Get("/feedback", func(w http.ResponseWriter, r *http.Request) {
			store.mu.RLock()
			defer store.mu.RUnlock()
			writeSuccess(w, "", store.feedback)
		})
	})
}
