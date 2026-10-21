package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/evenrtra/backend/internal/auth"
	"github.com/evenrtra/backend/internal/config"
	"github.com/evenrtra/backend/internal/database"
	"github.com/evenrtra/backend/internal/handlers"
	"github.com/evenrtra/backend/internal/middleware"
	"github.com/evenrtra/backend/internal/models"
	"github.com/evenrtra/backend/internal/repositories"
)

func main() {
	log.Println("Starting Eventra Event Management Platform...")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning loading config: %v", err)
		log.Println("Falling back to default development configuration")
		cfg = &config.Config{
			Port:            "8080",
			Environment:     "development",
			DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/eventra?sslmode=disable"),
			JWTSecret:       getEnv("JWT_SECRET", "eventra-super-secret-key-that-is-at-least-32-chars-long!"),
			JWTIssuer:       "eventra",
			JWTAudience:     "eventra-web",
			FirebaseProject: os.Getenv("FIREBASE_PROJECT_ID"),
			FrontendOrigin:  getEnv("FRONTEND_ORIGIN", "http://localhost:5500"),
			CookieDomain:    getEnv("COOKIE_DOMAIN", "localhost"),
			CookieSecure:    false,
			RateLimitRPS:    20,
			RateLimitBurst:  50,
		}
	}

	// 2. Initialize database connection
	log.Printf("Connecting to PostgreSQL at %s ...", maskURL(cfg.DatabaseURL))
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Println("Server will start in restricted mode (database unavailable until reconnected)")
	} else {
		defer db.Close()
		log.Println("Successfully connected to PostgreSQL database")

		// Run database migrations if migrations folder is found
		migrationsDir := findMigrationsDir()
		if migrationsDir != "" {
			log.Printf("Applying database migrations from %s ...", migrationsDir)
			if err := db.RunMigrations(migrationsDir); err != nil {
				log.Printf("Migration notice: %v", err)
			} else {
				log.Println("Database migrations up to date")
			}
		}
	}

	// 3. Initialize Auth Manager (Firebase + JWT)
	authManager, err := auth.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.FirebaseProject)
	if err != nil {
		log.Fatalf("Failed to initialize auth manager: %v", err)
	}
	if authManager.FirebaseEnabled() {
		log.Println("Firebase Authentication enabled and verified")
	} else {
		log.Println("Firebase Authentication not configured (using custom JWT & password credentials)")
	}

	// 4. Setup Router
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.LoggingMiddleware)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORSMiddleware(cfg.FrontendOrigin))

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	r.Use(rateLimiter.Limit)
	r.Use(chimiddleware.Recoverer)

	// Repositories and Handlers (if DB is active)
	if db != nil {
		userRepo := repositories.NewUserRepository(db.Pool)
		studentRepo := repositories.NewStudentRepository(db.Pool)
		clubRepo := repositories.NewClubRepository(db.Pool)
		eventRepo := repositories.NewEventRepository(db.Pool)
		regRepo := repositories.NewRegistrationRepository(db.Pool)
		feedbackRepo := repositories.NewFeedbackRepository(db.Pool)
		auditRepo := repositories.NewAuditLogRepository(db.Pool)

		authHandler := handlers.NewAuthHandler(userRepo, studentRepo, clubRepo, auditRepo, authManager, cfg)
		eventHandler := handlers.NewEventHandler(eventRepo, regRepo, clubRepo)
		clubAdminHandler := handlers.NewClubAdminHandler(eventRepo, clubRepo, regRepo, auditRepo)
		adminHandler := handlers.NewAdminHandler(userRepo, studentRepo, clubRepo, eventRepo, regRepo, feedbackRepo, auditRepo, db.Pool)
		feedbackHandler := handlers.NewFeedbackHandler(feedbackRepo)

		// API Routes
		r.Route("/api/v1", func(api chi.Router) {
			// Health check
			api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"ok","app":"Eventra","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
			})

			// Public Platform Stats
			api.Get("/stats", adminHandler.GetDashboardStats)

			// Public Feedback
			api.With(middleware.OptionalAuth(authManager)).Post("/feedback", feedbackHandler.SubmitFeedback)

			// Authentication endpoints
			api.Route("/auth", func(authRouter chi.Router) {
				authRouter.Post("/register/student", authHandler.RegisterStudent)
				authRouter.Post("/register/club", authHandler.RegisterClub)
				authRouter.Post("/register/super-admin", authHandler.RegisterSuperAdmin)
				authRouter.Post("/login", authHandler.Login)
				authRouter.Post("/login/student", authHandler.LoginStudent)
				authRouter.Post("/login/club", authHandler.LoginClub)
				authRouter.Post("/login/super-admin", authHandler.LoginSuperAdmin)
				authRouter.Post("/forgot-password", authHandler.ForgotPassword)

				// Authenticated auth routes
				authRouter.Group(func(priv chi.Router) {
					priv.Use(middleware.AuthMiddleware(authManager))
					priv.Get("/me", authHandler.GetMe)
					priv.Post("/logout", authHandler.Logout)
					priv.Get("/session", authHandler.GetSession)
				})
			})

			// Public Events and Clubs
			api.Route("/events", func(eventsRouter chi.Router) {
				eventsRouter.With(middleware.OptionalAuth(authManager)).Get("/", eventHandler.ListPublicEvents)
				eventsRouter.Get("/timeline", eventHandler.GetTimeline)
				eventsRouter.With(middleware.OptionalAuth(authManager)).Get("/{id}", eventHandler.GetEvent)

				// Student Registration endpoints
				eventsRouter.Group(func(studentOnly chi.Router) {
					studentOnly.Use(middleware.AuthMiddleware(authManager))
					studentOnly.Use(middleware.RequireRole(models.RoleStudent))
					studentOnly.Post("/{id}/register", eventHandler.RegisterForEvent)
					studentOnly.Post("/{id}/register-group", eventHandler.RegisterGroup)
				})
			})

			api.Route("/clubs", func(clubsRouter chi.Router) {
				clubsRouter.Get("/", eventHandler.ListClubs)
				clubsRouter.Get("/{id}", eventHandler.GetClub)
			})

			// Student Portal Routes
			api.Route("/student", func(studentRouter chi.Router) {
				studentRouter.Use(middleware.AuthMiddleware(authManager))
				studentRouter.Use(middleware.RequireRole(models.RoleStudent))
				studentRouter.Get("/registrations", eventHandler.GetMyRegistrations)
				studentRouter.Get("/registrations/{id}", eventHandler.GetRegistration)
			})

			// Club Admin Portal Routes
			api.Route("/club-admin", func(caRouter chi.Router) {
				caRouter.Use(middleware.AuthMiddleware(authManager))
				caRouter.Use(middleware.RequireRole(models.RoleClubAdmin, models.RoleSuperAdmin))

				caRouter.Get("/me", clubAdminHandler.GetMe)
				caRouter.Get("/events", clubAdminHandler.ListEvents)
				caRouter.Post("/events", clubAdminHandler.CreateEvent)
				caRouter.Put("/events/{id}", clubAdminHandler.UpdateEvent)
				caRouter.Delete("/events/{id}", clubAdminHandler.DeleteEvent)
				caRouter.Post("/events/{id}/extend-deadline", clubAdminHandler.ExtendDeadline)
				caRouter.Get("/events/{id}/registrations", clubAdminHandler.GetEventRegistrations)
			})

			// Super Admin Portal Routes
			api.Route("/admin", func(adminRouter chi.Router) {
				adminRouter.Use(middleware.AuthMiddleware(authManager))
				adminRouter.Use(middleware.RequireRole(models.RoleSuperAdmin))

				adminRouter.Get("/stats", adminHandler.GetDashboardStats)

				// Club management
				adminRouter.Get("/clubs", adminHandler.ListClubs)
				adminRouter.Get("/clubs/{id}", adminHandler.GetClub)
				adminRouter.Post("/clubs", adminHandler.CreateClub)
				adminRouter.Put("/clubs/{id}", adminHandler.UpdateClub)
				adminRouter.Delete("/clubs/{id}", adminHandler.DeleteClub)
				adminRouter.Post("/clubs/{id}/approve", adminHandler.ApproveClub)
				adminRouter.Post("/clubs/{id}/reject", adminHandler.RejectClub)

				// Student management
				adminRouter.Get("/students", adminHandler.ListStudents)
				adminRouter.Get("/students/{id}", adminHandler.GetStudent)
				adminRouter.Put("/students/{id}", adminHandler.UpdateStudent)
				adminRouter.Delete("/students/{id}", adminHandler.DeleteStudent)

				// Global event management
				adminRouter.Get("/events", adminHandler.ListAllEvents)
				adminRouter.Put("/events/{id}", adminHandler.UpdateEvent)
				adminRouter.Delete("/events/{id}", adminHandler.DeleteEvent)
				adminRouter.Get("/events/{id}/registrations", adminHandler.GetEventRegistrations)

				// Feedback moderation
				adminRouter.Get("/feedback", adminHandler.ListFeedback)
				adminRouter.Patch("/feedback/{id}", adminHandler.UpdateFeedback)
			})
		})
	} else {
		log.Println("PostgreSQL is not running locally. Initializing in-memory interactive demo mode...")
		r.Route("/api/v1", func(api chi.Router) {
			handlers.RegisterDemoRoutes(api, authManager, cfg)
		})
	}

	// Serve Frontend Static Files
	frontendDir := findFrontendDir()
	if frontendDir != "" {
		log.Printf("Serving frontend assets from %s", frontendDir)
		fileServer := http.FileServer(http.Dir(frontendDir))
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := filepath.Join(frontendDir, r.URL.Path)
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
			// If file doesn't exist, try index.html for SPA routing or 404
			indexPath := filepath.Join(frontendDir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				http.ServeFile(w, r, indexPath)
				return
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	// 5. Start HTTP server with graceful shutdown
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Eventra server running on http://localhost:%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Eventra server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Eventra server stopped successfully")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func maskURL(url string) string {
	if len(url) <= 15 {
		return "***"
	}
	return url[:12] + "..."
}

func findMigrationsDir() string {
	candidates := []string{
		"./migrations",
		"../migrations",
		"./backend/migrations",
		"c:/Users/USER/project250/evenrtra/backend/migrations",
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

func findFrontendDir() string {
	candidates := []string{
		"../../frontend",
		"../frontend",
		"./frontend",
		"c:/Users/USER/project250/evenrtra/frontend",
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}
