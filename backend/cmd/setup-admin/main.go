package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/evenrtra/backend/internal/config"
	"github.com/evenrtra/backend/internal/database"
	"github.com/evenrtra/backend/internal/models"
	"github.com/evenrtra/backend/internal/repositories"
)

func main() {
	emailFlag := flag.String("email", "", "Super Admin email address")
	passwordFlag := flag.String("password", "", "Super Admin password (min 8 chars)")
	flag.Parse()

	if *emailFlag == "" || *passwordFlag == "" {
		fmt.Println("Usage: go run ./cmd/setup-admin -email admin@eventra.edu -password Admin@123456")
		os.Exit(1)
	}

	if len(*passwordFlag) < 8 {
		log.Fatal("Error: Password must contain at least 8 characters")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}
	ctx := context.Background()

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	userRepo := repositories.NewUserRepository(db.Pool)
	email := strings.ToLower(strings.TrimSpace(*emailFlag))

	existing, _ := userRepo.GetByEmail(ctx, email)
	if existing != nil {
		log.Fatalf("Error: User with email %s already exists", email)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*passwordFlag), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Password hashing failed: %v", err)
	}

	user := &models.User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  string(hash),
		Role:          models.RoleSuperAdmin,
		IsActive:      true,
		EmailVerified: true,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		log.Fatalf("Failed to create Super Admin: %v", err)
	}

	fmt.Printf("Successfully provisioned Super Admin: %s (ID: %s)\n", user.Email, user.ID)
}
