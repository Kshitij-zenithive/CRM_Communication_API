// // package auth

// // import (
// // 	"context"
// // 	graphmodel "crm-communication-api/graph/model" // Use graph model for payload
// // 	dbmodel "crm-communication-api/internal/model"
// // 	//"github.com/google/uuid"
// // )

// // // Service defines the interface for authentication operations.
// // // Service defines the interface for authentication operations.
// // type Service interface {
// // 	// Login handles standard email/password authentication.
// // 	Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error)

// // 	// AuthenticateGoogleUser handles the callback from Google OAuth, finds/creates user, and returns tokens.
// // 	AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error)

// // 	// RefreshToken generates new access and refresh tokens based on a valid refresh token.
// // 	RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error)

// // 	// GetCurrentUser retrieves the currently authenticated user based on the context.
// // 	// This typically uses GetUserIDFromContext internally.
// // 	GetCurrentUser(ctx context.Context) (*dbmodel.User, error) // Returns internal db model

// // 	// GetAuthCodeURL returns the URL to initiate the Google OAuth flow.
// // 	// Specific to Google implementation but useful at the service layer boundary.
// // 	GetAuthCodeURL() string
// // }

// // internal/auth/service.go
// package auth

// import (
// 	"crm-communication-api/config"
// 	"crm-communication-api/internal/repository"
// 	"fmt"
// 	"log/slog"
// 	"time"

// 	"context"
// 	"net/http"

// 	"github.com/golang-jwt/jwt/v5"
// 	"golang.org/x/oauth2"
// 	"golang.org/x/oauth2/google" // Use alias if preferred: jwtv5 "github.com/golang-jwt/jwt/v5"

// 	// Needed for GetGmailService return type

// 	graphmodel "crm-communication-api/graph/model"

// 	// Alias for graph model
// 	dbmodel "crm-communication-api/internal/model"
// )

// // Alias for DB model

// // Service defines the interface for all authentication operations.
// // Concrete implementations (like LocalAuthService, GoogleAuthService) will provide the logic.
// type Service interface {
// 	// Login handles standard email/password authentication.
// 	Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error)

// 	// AuthenticateGoogleUser handles the callback from Google OAuth.
// 	AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error)

// 	// RefreshToken generates new access and refresh tokens.
// 	RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error)

// 	// GetCurrentUser retrieves the currently authenticated user from context claims.
// 	GetCurrentUser(ctx context.Context) (*dbmodel.User, error) // Returns internal db model

// 	// GetAuthCodeURL returns the URL to initiate the Google OAuth flow.
// 	GetAuthCodeURL() string

// 	// GetGmailService returns an authenticated http client for Gmail API calls using user's token.
// 	GetGmailService(ctx context.Context, userID string) (*http.Client, error)

// 	// VerifyJWT validates a token string. Useful for middleware or other services.
// 	VerifyJWT(tokenString string) (*Claims, error)
// }

// // Note: The concrete AuthService struct and NewAuthService constructor are removed from this file.
// // Implementations will be in local.go and google.go.

// const (
// 	// Lifespan for the signed OAuth state token
// 	oauthStateTokenLifespan = 10 * time.Minute
// )

// // AuthService provides authentication related functionalities.
// // It encapsulates dependencies like repository, config, logger, etc.
// type AuthService struct {
// 	repo              repository.AuthRepository // Use the specific interface for DB access
// 	config            *config.Config
// 	logger            *slog.Logger
// 	googleOAuthConfig *oauth2.Config // Google OAuth config, ready to use
// 	jwtSigningKey     []byte         // Parsed JWT key for signing/verifying tokens
// }

// // NewAuthService creates a new AuthService instance.
// // It initializes dependencies required by the authentication logic.
// func NewAuthService(repo repository.AuthRepository, cfg *config.Config, logger *slog.Logger) (*AuthService, error) {
// 	// Validate essential configuration
// 	if cfg.JWTSecret == "" {
// 		return nil, ErrMissingJWTSecret
// 	}
// 	jwtKey := []byte(cfg.JWTSecret)

// 	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
// 		logger.Warn("Google OAuth Client ID or Secret is missing. Google login will be disabled.")
// 		// Return service without Google config, methods using it will fail or should check
// 		// Or return an error if Google login is mandatory: return nil, ErrMissingGoogleConfig
// 	}

// 	var googleOAuthConfig *oauth2.Config
// 	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
// 		googleOAuthConfig = &oauth2.Config{
// 			ClientID:     cfg.GoogleClientID,
// 			ClientSecret: cfg.GoogleClientSecret,
// 			RedirectURL:  cfg.GoogleRedirectURL, // Ensure this is configured
// 			Scopes:       []string{"openid", "profile", "email"},
// 			Endpoint:     google.Endpoint,
// 		}
// 		logger.Info("Google OAuth configured")
// 	} else {
// 		logger.Warn("Google OAuth not configured")
// 	}

// 	service := &AuthService{
// 		repo:              repo,
// 		config:            cfg,
// 		logger:            logger.With(slog.String("service", "AuthService")), // Add service context to logger
// 		jwtSigningKey:     jwtKey,
// 		googleOAuthConfig: googleOAuthConfig,
// 	}

//		logger.Info("AuthService initialized successfully")
//		return service, nil
//	}
//
// internal/auth/service.go
package auth

import (
	"context"
	"net/http" // Needed for GetGmailService return type

	graphmodel "crm-communication-api/graph/model" // Alias for graph model
	dbmodel "crm-communication-api/internal/model" // Alias for DB model
)

// Service defines the interface for all authentication operations.
// Concrete implementations (like LocalAuthService, GoogleAuthService) will provide the logic.
type Service interface {
	// Login handles standard email/password authentication.
	Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error)

	// AuthenticateGoogleUser handles the callback from Google OAuth.
	AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error)

	// RefreshToken generates new access and refresh tokens based on a valid refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error)

	// GetCurrentUser retrieves the currently authenticated user from context claims.
	GetCurrentUser(ctx context.Context) (*dbmodel.User, error) // Returns internal db model

	// GetAuthCodeURL returns the URL to initiate the Google OAuth flow.
	GetAuthCodeURL() string

	// GetGmailService returns an authenticated http client for Gmail API calls using user's token.
	GetGmailService(ctx context.Context, userID string) (*http.Client, error)

	// VerifyJWT validates a token string. Useful for middleware or other services.
	VerifyJWT(tokenString string) (*Claims, error)
}

// NO AuthService struct or NewAuthService here. Implementations are in local.go and google.go.
