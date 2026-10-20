package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

// Manager handles Firebase verification and JWT operations
type Manager struct {
	firebaseAuth *fbauth.Client
	jwtSecret    []byte
	jwtIssuer    string
	jwtAudience  string
}

// Claims represents JWT token claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// NewManager initializes the auth manager
func NewManager(jwtSecret, jwtIssuer, jwtAudience, firebaseProjectID string) (*Manager, error) {
	m := &Manager{
		jwtSecret:   []byte(jwtSecret),
		jwtIssuer:   jwtIssuer,
		jwtAudience: jwtAudience,
	}

	// Initialize Firebase if project ID is configured
	if firebaseProjectID != "" {
		ctx := context.Background()
		var app *firebase.App
		var err error

		credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		if credFile != "" {
			app, err = firebase.NewApp(ctx, &firebase.Config{
				ProjectID: firebaseProjectID,
			}, option.WithCredentialsFile(credFile))
		} else {
			app, err = firebase.NewApp(ctx, &firebase.Config{
				ProjectID: firebaseProjectID,
			})
		}

		if err != nil {
			return nil, fmt.Errorf("error initializing firebase app: %w", err)
		}

		authClient, err := app.Auth(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting firebase auth client: %w", err)
		}
		m.firebaseAuth = authClient
	}

	return m, nil
}

// FirebaseEnabled returns whether Firebase is configured
func (m *Manager) FirebaseEnabled() bool {
	return m.firebaseAuth != nil
}

// VerifyFirebaseToken verifies a Firebase ID token and returns the decoded token
func (m *Manager) VerifyFirebaseToken(ctx context.Context, idToken string) (*fbauth.Token, error) {
	if m.firebaseAuth == nil {
		return nil, fmt.Errorf("firebase auth not initialized - set FIREBASE_PROJECT_ID and GOOGLE_APPLICATION_CREDENTIALS")
	}
	return m.firebaseAuth.VerifyIDToken(ctx, idToken)
}

// GenerateJWT creates a new application JWT
func (m *Manager) GenerateJWT(userID uuid.UUID, email, role string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	claims := Claims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    m.jwtIssuer,
			Audience:  jwt.ClaimStrings{m.jwtAudience},
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.jwtSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("error signing JWT: %w", err)
	}

	return signedToken, expiresAt, nil
}

// ValidateJWT parses and validates an application JWT
func (m *Manager) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.jwtSecret, nil
	},
		jwt.WithIssuer(m.jwtIssuer),
		jwt.WithAudience(m.jwtAudience),
		jwt.WithValidMethods([]string{"HS256"}),
	)

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
