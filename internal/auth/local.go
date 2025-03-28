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
	dbmodel "crm-communication-api/internal/model"
	"crm-communication-api/internal/repository" // Import repository specifically for ErrNotFound check
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/bcrypt" // Used for error comparison
)

// Register creates a new user account with email and password.
func (s *AuthService) Register(ctx context.Context, name, email, password string) (*dbmodel.User, error) {
	s.logger.Debug("Attempting user registration", slog.String("email", email))

	// Validate input (basic example)
	if name == "" || email == "" || password == "" {
		return nil, fmt.Errorf("name, email, and password are required")
	}
	// Consider adding more robust email validation

	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		s.logger.Error("Failed to check for existing user during registration", slog.String("email", email), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to verify user existence: %w", err) // Internal error
	}
	if existingUser != nil {
		s.logger.Warn("Registration attempt with existing email", slog.String("email", email))
		return nil, ErrEmailTaken
	}

	// Create user model
	user := &dbmodel.User{
		Name:  strings.TrimSpace(name),
		Email: strings.ToLower(strings.TrimSpace(email)),
		Role:  "user", // Default role, consider making this configurable or dynamic
	}

	// Hash password using the method on the User model
	if err := user.SetPassword(password); err != nil {
		s.logger.Error("Failed to hash password during registration", slog.String("email", email), slog.String("error", err.Error()))
		return nil, ErrPasswordHashing
	}

	// Save user to database
	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user in repository", slog.String("email", email), slog.String("error", err.Error()))
		// Check for potential duplicate email race condition errors if the DB enforces uniqueness
		// if errors.Is(err, <specific_db_duplicate_error>) { return nil, ErrEmailTaken }
		return nil, fmt.Errorf("failed to save new user: %w", err) // Internal error
	}

	s.logger.Info("User registered successfully", slog.String("userID", user.ID.String()), slog.String("email", email))
	// Avoid returning password hash in the response object
	user.Password = "" // Clear password hash before returning
	return user, nil
}

// LocalLogin authenticates a user with email and password, returning new tokens.
func (s *AuthService) LocalLogin(ctx context.Context, email, password string) (accessToken string, refreshToken string, err error) {
	s.logger.Debug("Attempting local login", slog.String("email", email))

	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Warn("Login attempt for non-existent email", slog.String("email", email), slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", ErrInvalidCredentials // Use generic error for non-existent user
		}
		return "", "", fmt.Errorf("failed to retrieve user: %w", err) // Internal error
	}

	// Compare password using the method on the User model
	if err := user.ComparePassword(password); err != nil {
		s.logger.Warn("Invalid password attempt", slog.String("userID", user.ID.String()), slog.String("email", email))
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", "", ErrInvalidCredentials // Generic error for password mismatch
		}
		// Log other potential comparison errors (e.g., invalid hash format)
		s.logger.Error("Password comparison failed", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return "", "", ErrInvalidCredentials // Still return generic credential error
	}

	// --- Authentication successful ---

	// Generate new access token
	accessToken, err = s.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate access token after login", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return "", "", fmt.Errorf("login succeeded but failed to issue access token: %w", err)
	}

	// Generate new refresh token
	refreshToken, err = s.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		s.logger.Error("Failed to generate refresh token after login", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		// Allow login to succeed with access token, but log critical failure for refresh token
		return accessToken, "", fmt.Errorf("login succeeded but failed to issue refresh token: %w", err)
	}

	s.logger.Info("User logged in successfully via local auth", slog.String("userID", user.ID.String()))
	return accessToken, refreshToken, nil
}
