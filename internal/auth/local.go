// package auth

// import (
// 	"context"
// 	"crm-communication-api/config"
// 	graphmodel "crm-communication-api/graph/model" // Use alias for graph model
// 	dbmodel "crm-communication-api/internal/model"  // Use alias for DB model
// 	"crm-communication-api/internal/repository"
// 	"errors" // CORRECTED: Added import
// 	"fmt"

// 	"github.com/google/uuid"
// )

// type LocalAuthService struct {
// 	Config     *config.Config
// 	Repository repository.AuthRepository // Use interface
// }

// // Assert that LocalAuthService implements the Service interface
// var _ Service = (*LocalAuthService)(nil)

// // NewLocalAuthService constructor using the AuthRepository interface
// func NewLocalAuthService(cfg *config.Config, repo repository.AuthRepository) *LocalAuthService {
// 	return &LocalAuthService{
// 		Config:     cfg,
// 		Repository: repo,
// 	}
// }

// // Login handles email/password authentication
// func (s *LocalAuthService) Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error) {
// 	// Fetch dbmodel.User
// 	user, err := s.Repository.GetUserByEmail(ctx, email)
// 	if err != nil {
// 		if errors.Is(err, repository.ErrNotFound) { // CORRECTED: Use errors.Is
// 			return nil, fmt.Errorf("invalid email or password") // Generic error
// 		}
// 		return nil, fmt.Errorf("error retrieving user: %w", err)
// 	}

// 	// Compare password using method on dbmodel.User
// 	if err := user.ComparePassword(password); err != nil {
// 		return nil, fmt.Errorf("invalid email or password") // Generic error
// 	}

// 	// Generate tokens using dbmodel.User
// 	accessToken, err := GenerateJWT(user, "local", s.Config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate access token: %w", err)
// 	}
// 	// Pass repo interface to GenerateRefreshToken
// 	refreshToken, err := GenerateRefreshToken(ctx, user.ID, s.Repository, s.Config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
// 	}

// 	// Convert dbmodel.User to graphmodel.User
// 	graphUser := &graphmodel.User{ // CORRECTED Type
// 		ID:    user.ID.String(),
// 		Name:  user.Name,
// 		Email: user.Email,
//         Avatar: &user.Avatar,
// 		Role:  user.Role,
// 	}

// 	// Return graphmodel.AuthPayload
// 	return &graphmodel.AuthPayload{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		User:         graphUser,
// 	}, nil
// }

// // RefreshToken uses shared logic and accepts AuthRepository interface
// func (s *LocalAuthService) RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error) {
// 	// CORRECTED: Pass s.Repository interface directly
// 	accessToken, newRefreshToken, err := RefreshAccessToken(ctx, refreshToken, s.Repository, s.Config)
// 	if err != nil {
// 		return nil, err
// 	}

// 	claims, err := VerifyJWT(accessToken, s.Config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to verify newly created access token: %w", err)
// 	}
// 	userID, _ := uuid.Parse(claims.UserID)

// 	// Fetch dbmodel.User
// 	user, err := s.Repository.GetUserByID(ctx, userID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve user during refresh: %w", err)
// 	}

// 	// Convert to graphmodel.User
// 	graphUser := &graphmodel.User{ // CORRECTED Type
// 		ID:    user.ID.String(),
// 		Name:  user.Name,
// 		Email: user.Email,
//         Avatar: &user.Avatar,
// 		Role:  user.Role,
// 	}

// 	return &graphmodel.AuthPayload{
// 		AccessToken:  accessToken,
// 		RefreshToken: newRefreshToken,
// 		User:         graphUser,
// 	}, nil
// }

// // GetCurrentUser uses context helper and repository (returns dbmodel.User)
// func (s *LocalAuthService) GetCurrentUser(ctx context.Context) (*dbmodel.User, error) {
// 	userID, err := GetUserIDFromContext(ctx) // CORRECTED: No 'auth.' prefix needed
// 	if err != nil {
// 		return nil, err
// 	}

// 	user, err := s.Repository.GetUserByID(ctx, userID)
// 	if err != nil {
// 		if errors.Is(err, repository.ErrNotFound) { // CORRECTED: Use errors.Is
// 			return nil, fmt.Errorf("authenticated user not found in database")
// 		}
// 		return nil, fmt.Errorf("failed to retrieve user from repository: %w", err)
// 	}
// 	return user, nil
// }

// // --- Interface Methods Not Implemented ---

// func (s *LocalAuthService) AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error) {
// 	return nil, fmt.Errorf("local auth service does not handle Google OAuth")
// }

// func (s *LocalAuthService) GetAuthCodeURL() string {
// 	return "" // No URL for local auth
// }

// internal/auth/local.go
package auth

import (
	"context"
	"crm-communication-api/config"
	graphmodel "crm-communication-api/graph/model"
	dbmodel "crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"errors"
	"fmt"
	"log/slog"      // Added import
	"net/http"    // Added import for GetGmailService signature

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt" // Added import for error check
)

// LocalAuthService handles email/password authentication.
type LocalAuthService struct {
	repo          repository.AuthRepository // Use the specific interface for DB access
	config        *config.Config
	logger        *slog.Logger              // Added logger
	jwtSigningKey []byte                  // Added JWT key
}

// Assert that LocalAuthService implements the Service interface
var _ Service = (*LocalAuthService)(nil)

// NewLocalAuthService creates a new LocalAuthService instance.
func NewLocalAuthService(repo repository.AuthRepository, cfg *config.Config, logger *slog.Logger) (*LocalAuthService, error) {
	// Validate essential configuration
	if cfg.JWTSecretKey == "" { // Check correct field name
		return nil, ErrMissingJWTSecret
	}
	jwtKey := []byte(cfg.JWTSecretKey) // Use correct field name

	return &LocalAuthService{
		repo:          repo,
		config:        cfg,
		logger:        logger.With(slog.String("service", "LocalAuthService")), // Add context
		jwtSigningKey: jwtKey,
	}, nil
}

// Login handles email/password authentication.
func (s *LocalAuthService) Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error) {
	l := s.logger.With(slog.String("method", "Login"), slog.String("email", email))
	l.Debug("Attempting local login")

	// Fetch dbmodel.User
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		l.Warn("Login failed: user lookup error", slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials // Generic error
		}
		return nil, fmt.Errorf("internal error retrieving user: %w", err)
	}

	if user.PasswordHash == "" { // Check correct field name
		l.Warn("Login attempt failed for OAuth user", slog.String("userID", user.ID.String()))
		return nil, ErrInvalidCredentials
	}

	// Compare password using method on dbmodel.User
	if err := user.ComparePassword(password); err != nil {
		l.Warn("Invalid password attempt", slog.String("userID", user.ID.String()))
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}
		l.Error("Password comparison unexpected error", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return nil, ErrInvalidCredentials
	}

	// Generate tokens using package-level helpers from jwt.go
	accessToken, err := GenerateJWT(user, "local", s.config) // Pass necessary args
	if err != nil {
		l.Error("Failed to generate access token", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return nil, fmt.Errorf("login failed during access token generation: %w", err)
	}

	// Pass repo interface to GenerateRefreshToken
	refreshToken, err := GenerateRefreshToken(ctx, user.ID, s.repo, s.config) // Pass repo
	if err != nil {
		l.Error("Failed to generate refresh token", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return nil, fmt.Errorf("login failed during refresh token generation: %w", err)
	}

	l.Info("User logged in successfully via local auth", slog.String("userID", user.ID.String()))
	return createAuthPayload(user, accessToken, refreshToken), nil // Use helper
}

// RefreshToken uses shared logic via package-level helper.
func (s *LocalAuthService) RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error) {
	// Pass repo interface directly to the package-level helper
	newAccessToken, newRefreshToken, err := RefreshAccessToken(ctx, refreshToken, s.repo, s.config)
	if err != nil {
		return nil, err // Error is already context-rich from helper
	}

	// Verify new token to get claims
	claims, err := VerifyJWT(newAccessToken, s.config) // Use package-level helper
	if err != nil {
		// This indicates a potential issue if the token generated by RefreshAccessToken is invalid
		s.logger.Error("Failed to verify newly generated access token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("internal error verifying refreshed token: %w", err)
	}
	userID, err := uuid.Parse(claims.Subject) // Use Subject claim
	if err != nil {
        s.logger.Error("Invalid UserID found in refreshed token claims", slog.String("subject", claims.Subject), slog.String("error", err.Error()))
        return nil, fmt.Errorf("internal error processing refreshed token claims")
    }


	// Fetch dbmodel.User
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user during refresh: %w", err)
	}

	return createAuthPayload(user, newAccessToken, newRefreshToken), nil // Use helper
}

// GetCurrentUser uses context helper and repository.
func (s *LocalAuthService) GetCurrentUser(ctx context.Context) (*dbmodel.User, error) {
	// Use helper function defined in auth/context.go
	userID, err := GetUserIDFromContext(ctx)
	if err != nil {
		s.logger.Warn("Attempted to get current user but no user ID in context", slog.String("error", err.Error()))
		return nil, err // Propagate ErrUserNotInContext or parsing error
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to retrieve authenticated user from repository", slog.String("userID", userID.String()), slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound // User in token claims doesn't exist in DB
		}
		return nil, fmt.Errorf("internal error retrieving user: %w", err)
	}
	return user, nil
}

// VerifyJWT delegates to the package-level helper in jwt.go
func (s *LocalAuthService) VerifyJWT(tokenString string) (*Claims, error) {
    return VerifyJW(tokenString, s.config)
}


// --- Google Specific Methods (Stubs for LocalAuthService) ---

func (s *LocalAuthService) AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error) {
	return nil, fmt.Errorf("local auth service does not handle Google OAuth")
}

func (s *LocalAuthService) GetAuthCodeURL() string {
	return "" // No URL applicable
}

func (s *LocalAuthService) GetGmailService(ctx context.Context, userID string) (*http.Client, error) {
    return nil, fmt.Errorf("local auth service cannot provide Gmail service client")
}

// createAuthPayload is a helper common to different auth methods
func createAuthPayload(user *dbmodel.User, accessToken, refreshToken string) *graphmodel.AuthPayload {
	return &graphmodel.AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &graphmodel.User{
			ID:     user.ID.String(),
			Name:   user.Name,
			Email:  user.Email,
			Avatar: &user.Avatar,
			Role:   user.Role,
			// Add CreatedAt/UpdatedAt if they exist in graphmodel.User
		},
	}
}
