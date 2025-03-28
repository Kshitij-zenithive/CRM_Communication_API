// package auth

// import (
// 	"context"
// 	"crm-communication-api/config"
// 	dbmodel "crm-communication-api/internal/model" // CORRECTED ALIAS
// 	"crm-communication-api/internal/repository"
// 	"errors" // Ensure errors is imported if used elsewhere (like in VerifyJWT)
// 	"fmt"
// 	"time"

// 	"github.com/golang-jwt/jwt/v4"
// 	"github.com/google/uuid"
// )

// // Claims struct remains the same
// type Claims struct {
// 	UserID       string `json:"user_id"`
// 	Name         string `json:"name"`
// 	Role         string `json:"role"`
// 	AuthProvider string `json:"auth_provider"`
// 	Email        string `json:"email"`
// 	jwt.RegisteredClaims
// }

// // GenerateJWT uses dbmodel.User
// func GenerateJWT(user *dbmodel.User, authProvider string, cfg *config.Config) (string, error) {
// 	expirationTime := time.Now().Add(cfg.JWTExpiry)
// 	claims := &Claims{
// 		UserID:       user.ID.String(),
// 		Name:         user.Name,
// 		Role:         user.Role,
// 		AuthProvider: authProvider,
// 		Email:        user.Email,
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			ExpiresAt: jwt.NewNumericDate(expirationTime),
// 			IssuedAt:  jwt.NewNumericDate(time.Now()),
// 			NotBefore: jwt.NewNumericDate(time.Now()),
// 			Issuer:    "crm-communication-api",
// 			Subject:   user.ID.String(),
// 		},
// 	}
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	tokenString, err := token.SignedString([]byte(cfg.JWTSecretKey))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to sign token: %w", err)
// 	}
// 	return tokenString, nil
// }

// // VerifyJWT remains the same
// func VerifyJWT(tokenString string, cfg *config.Config) (*Claims, error) {
// 	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
// 		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
// 			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
// 		}
// 		return []byte(cfg.JWTSecretKey), nil
// 	})

// 	if err != nil {
// 		if ve, ok := err.(*jwt.ValidationError); ok {
// 			if ve.Errors&jwt.ValidationErrorExpired != 0 {
// 				return nil, fmt.Errorf("token has expired: %w", err)
// 			}
// 		}
// 		return nil, fmt.Errorf("invalid token: %w", err)
// 	}

// 	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
// 		return claims, nil
// 	}

// 	return nil, errors.New("invalid token")
// }

// // GenerateRefreshToken uses dbmodel.RefreshToken and accepts AuthRepository interface
// func GenerateRefreshToken(ctx context.Context, userID uuid.UUID, repo repository.AuthRepository, cfg *config.Config) (string, error) { // CORRECTED: repo type
// 	refreshTokenID := uuid.New()
// 	expiresAt := time.Now().Add(cfg.RefreshTokenExpiry)

// 	// Use dbmodel.RefreshToken
// 	refreshToken := dbmodel.RefreshToken{
// 		// ID handled by GORM
// 		UserID:    userID,
// 		Token:     refreshTokenID.String(),
// 		ExpiresAt: expiresAt,
// 	}

// 	if err := repo.CreateRefreshToken(ctx, &refreshToken); err != nil {
// 		return "", fmt.Errorf("failed to create refresh token: %w", err)
// 	}

// 	return refreshToken.Token, nil
// }

// // RefreshAccessToken accepts AuthRepository interface
// func RefreshAccessToken(ctx context.Context, refreshTokenString string, repo repository.AuthRepository, cfg *config.Config) (string, string, error) { // CORRECTED: repo type
// 	refreshToken, err := repo.GetRefreshToken(ctx, refreshTokenString)
// 	if err != nil {
// 		// More specific error check combining not found and expiration
// 		if errors.Is(err, repository.ErrNotFound) {
// 			return "", "", fmt.Errorf("refresh token not found or expired")
// 		}
// 		return "", "", fmt.Errorf("error getting refresh token: %w", err)
// 	}

// 	// Explicitly check expiry again just in case GetRefreshToken logic changes
// 	if time.Now().After(refreshToken.ExpiresAt) {
// 		_ = repo.DeleteRefreshToken(ctx, refreshTokenString)
// 		return "", "", errors.New("refresh token has expired")
// 	}

// 	user, err := repo.GetUserByID(ctx, refreshToken.UserID)
// 	if err != nil {
// 		return "", "", fmt.Errorf("failed to get user associated with refresh token: %w", err)
// 	}

// 	// Determine AuthProvider
// 	authProvider := "local" // Default
// 	_, err = repo.GetOAuthProvider(ctx, user.ID, "google")
// 	if err == nil {
// 		authProvider = "google"
// 	} else if !errors.Is(err, repository.ErrNotFound) {
//         // Log unexpected error fetching OAuth provider, but proceed with 'local'
//         fmt.Printf("Warning: error checking OAuth provider for user %s: %v\n", user.ID, err)
//     }

// 	newAccessToken, err := GenerateJWT(user, authProvider, cfg)
// 	if err != nil {
// 		return "", "", fmt.Errorf("failed to generate new access token: %w", err)
// 	}
// 	newRefreshToken, err := GenerateRefreshToken(ctx, user.ID, repo, cfg) // Pass repo interface
// 	if err != nil {
// 		return "", "", fmt.Errorf("failed to generate new refresh token: %w", err)
// 	}

// 	if err := repo.DeleteRefreshToken(ctx, refreshTokenString); err != nil {
// 		fmt.Printf("Warning: failed to delete old refresh token %s: %v\n", refreshTokenString, err)
// 	}

//		return newAccessToken, newRefreshToken, nil
//	}
//

// internal/auth/jwt.go
package auth

import (
	"context"
	dbmodel "crm-communication-api/internal/model" // Alias to avoid clash if needed
	"crm-communication-api/internal/repository"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims containing user information.
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"` // e.g., "user", "admin"
}

// GenerateJWT creates a new JWT access token for a user.
func (s *AuthService) GenerateJWT(userID uuid.UUID, email string, role string) (string, error) {
	s.logger.Debug("Generating JWT", slog.String("userID", userID.String()), slog.String("email", email))

	duration := time.Duration(s.config.JWTExpirationMinutes) * time.Minute
	claims := s.generateStandardClaims(userID, email, role, duration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSigningKey)
	if err != nil {
		s.logger.Error("Failed to sign JWT", slog.String("error", err.Error()))
		// Don't wrap internal signing error directly for security, return generic error
		return "", ErrTokenGeneration
	}

	s.logger.Debug("JWT generated successfully", slog.String("userID", userID.String()))
	return signedToken, nil
}

// VerifyJWT validates a JWT string and returns the claims if valid.
func (s *AuthService) VerifyJWT(tokenString string) (*Claims, error) {
	s.logger.Debug("Verifying JWT")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSigningKey, nil
	})

	if err != nil {
		s.logger.Warn("JWT verification failed", slog.String("error", err.Error()))
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		// Includes malformed token, signature invalid, etc.
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		s.logger.Debug("JWT verified successfully", slog.String("userID", claims.Subject))
		return claims, nil
	}

	s.logger.Warn("JWT token was parsed but marked invalid")
	return nil, ErrTokenInvalid
}


// GenerateRefreshToken creates, stores, and returns a new secure refresh token.
func (s *AuthService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	s.logger.Debug("Generating refresh token", slog.String("userID", userID.String()))

	// Generate a cryptographically secure random token string
	byteLength := 32 // 32 bytes = 256 bits
	randomBytes := make([]byte, byteLength)
	if _, err := rand.Read(randomBytes); err != nil {
		s.logger.Error("Failed to generate random bytes for refresh token", slog.String("error", err.Error()))
		return "", ErrTokenGeneration
	}
	tokenString := base64.URLEncoding.EncodeToString(randomBytes)

	expiresAt := time.Now().AddDate(0, 0, s.config.RefreshTokenExpirationDays)

	refreshToken := &dbmodel.RefreshToken{
		UserID:    userID,
		Token:     tokenString, // Store the raw token string
		ExpiresAt: expiresAt,
	}

	if err := s.repo.CreateRefreshToken(ctx, refreshToken); err != nil {
		s.logger.Error("Failed to store refresh token in repository", slog.String("error", err.Error()))
		// Don't expose DB errors directly
		return "", fmt.Errorf("%w: failed to save refresh token", ErrTokenGeneration)
	}

	s.logger.Info("Refresh token generated and stored", slog.String("userID", userID.String()))
	return tokenString, nil
}

// RefreshAccessToken validates a refresh token and issues a new pair of access and refresh tokens.
// Implements refresh token rotation for enhanced security.
func (s *AuthService) RefreshAccessToken(ctx context.Context, refreshTokenString string) (newAccessToken string, newRefreshToken string, err error) {
	s.logger.Debug("Attempting to refresh access token")

	if refreshTokenString == "" {
		return "", "", ErrMissingRefreshToken
	}

	// 1. Find the refresh token in the database
	// Note: Consider hashing the token in the DB for extra security, requires changing FindRefreshToken logic.
	// For simplicity now, we store the raw token.
	refreshToken, err := s.repo.FindRefreshToken(ctx, refreshTokenString)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("Refresh token not found in repository", slog.String("error", err.Error()))
			return "", "", ErrRefreshTokenNotFound // Or ErrTokenInvalid if we suspect reuse attempt
		}
		s.logger.Error("Failed to query refresh token from repository", slog.String("error", err.Error()))
		return "", "", fmt.Errorf("failed to retrieve refresh token data: %w", err) // Internal error
	}

	// 2. Immediately delete the used refresh token to prevent reuse (Rotation step 1)
	if err := s.repo.DeleteRefreshToken(ctx, refreshToken.ID); err != nil {
		// Log the error but proceed if possible, as the token is now invalid anyway.
		// Critical failure here might indicate a DB issue.
		s.logger.Error("Failed to delete used refresh token", slog.String("refreshTokenID", refreshToken.ID.String()), slog.String("error", err.Error()))
		// Depending on policy, you might want to return an error here to halt the process.
		// return "", "", fmt.Errorf("failed to invalidate used refresh token: %w", err)
	} else {
		s.logger.Debug("Used refresh token deleted", slog.String("refreshTokenID", refreshToken.ID.String()))
	}


	// 3. Check if the token has expired
	if time.Now().After(refreshToken.ExpiresAt) {
		s.logger.Warn("Attempted to use expired refresh token", slog.String("userID", refreshToken.UserID.String()))
		return "", "", ErrRefreshTokenExpired
	}


	// 4. Get user information
	user, err := s.repo.GetUserByID(ctx, refreshToken.UserID)
	if err != nil {
		s.logger.Error("Failed to get user associated with refresh token", slog.String("userID", refreshToken.UserID.String()), slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", ErrUserNotFound // User might have been deleted
		}
		return "", "", ErrGetUserInfo // Internal error
	}

	// 5. Generate a new access token
	newAccessToken, err = s.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate new access token during refresh", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		// If we can't generate a new access token, we can't proceed.
		return "", "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	// 6. Generate a new refresh token (Rotation step 2)
	newRefreshToken, err = s.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		s.logger.Error("Failed to generate new refresh token during refresh", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		// Critical: If we issued an access token but failed to issue a new refresh token,
		// the user can use the API but cannot refresh again. Log critically.
		// Depending on policy, you might try to rollback or just return the access token.
		// Let's return the access token but no refresh token, forcing re-login next time.
		return newAccessToken, "", fmt.Errorf("issued new access token but failed to generate new refresh token: %w", err)
	}

	s.logger.Info("Access token refreshed successfully", slog.String("userID", user.ID.String()))
	return newAccessToken, newRefreshToken, nil
}