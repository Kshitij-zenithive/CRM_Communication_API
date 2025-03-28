// package auth

// import (
// 	"context"
// 	graphmodel "crm-communication-api/graph/model" // Use graph model for payload
// 	dbmodel "crm-communication-api/internal/model"
// 	//"github.com/google/uuid"
// )

// // Service defines the interface for authentication operations.
// // Service defines the interface for authentication operations.
// type Service interface {
// 	// Login handles standard email/password authentication.
// 	Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error)

// 	// AuthenticateGoogleUser handles the callback from Google OAuth, finds/creates user, and returns tokens.
// 	AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error)

// 	// RefreshToken generates new access and refresh tokens based on a valid refresh token.
// 	RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error)

// 	// GetCurrentUser retrieves the currently authenticated user based on the context.
// 	// This typically uses GetUserIDFromContext internally.
// 	GetCurrentUser(ctx context.Context) (*dbmodel.User, error) // Returns internal db model

// 	// GetAuthCodeURL returns the URL to initiate the Google OAuth flow.
// 	// Specific to Google implementation but useful at the service layer boundary.
// 	GetAuthCodeURL() string
// }

// internal/auth/service.go
package auth

import (
	"crm-communication-api/config"
	"crm-communication-api/internal/repository"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5" // Use alias if preferred: jwtv5 "github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// Lifespan for the signed OAuth state token
	oauthStateTokenLifespan = 10 * time.Minute
)

// AuthService provides authentication related functionalities.
// It encapsulates dependencies like repository, config, logger, etc.
type AuthService struct {
	repo              repository.AuthRepository // Use the specific interface for DB access
	config            *config.Config
	logger            *slog.Logger
	googleOAuthConfig *oauth2.Config // Google OAuth config, ready to use
	jwtSigningKey     []byte         // Parsed JWT key for signing/verifying tokens
}

// NewAuthService creates a new AuthService instance.
// It initializes dependencies required by the authentication logic.
func NewAuthService(repo repository.AuthRepository, cfg *config.Config, logger *slog.Logger) (*AuthService, error) {
	// Validate essential configuration
	if cfg.JWTSecret == "" {
		return nil, ErrMissingJWTSecret
	}
	jwtKey := []byte(cfg.JWTSecret)

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		logger.Warn("Google OAuth Client ID or Secret is missing. Google login will be disabled.")
		// Return service without Google config, methods using it will fail or should check
		// Or return an error if Google login is mandatory: return nil, ErrMissingGoogleConfig
	}

	var googleOAuthConfig *oauth2.Config
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		googleOAuthConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL, // Ensure this is configured
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     google.Endpoint,
		}
		logger.Info("Google OAuth configured")
	} else {
		logger.Warn("Google OAuth not configured")
	}


	service := &AuthService{
		repo:              repo,
		config:            cfg,
		logger:            logger.With(slog.String("service", "AuthService")), // Add service context to logger
		jwtSigningKey:     jwtKey,
		googleOAuthConfig: googleOAuthConfig,
	}

	logger.Info("AuthService initialized successfully")
	return service, nil
}


// --- Helper Methods ---

// generateStandardClaims generates standard JWT claims with expiry.
func (s *AuthService) generateStandardClaims(userID fmt.Stringer, email string, role string, duration time.Duration) *Claims {
	now := time.Now()
	expiresAt := now.Add(duration)
	return &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "crm-communication-api", // Consider making this configurable
			// Audience: []string{"your-client"}, // Optional: If needed
		},
		Email: email,
		Role:  role,
	}
}