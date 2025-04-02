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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"crm-communication-api/config"
	graphmodel "crm-communication-api/graph/model"
	dbmodel "crm-communication-api/internal/model" // Alias for database models
	"crm-communication-api/internal/repository"

	"github.com/golang-jwt/jwt/v5" // Use v5 consistently
	"github.com/google/uuid"
)

// Claims represents the structure of the JWT claims.
// It's defined here as it's tightly coupled with JWT generation/verification.
type Claims struct {
	jwt.RegisteredClaims        // Standard claims (sub, iss, aud, exp, nbf, iat, jti)
	Email                string `json:"email"`         // Custom claim: User's email
	Role                 string `json:"role"`          // Custom claim: User's role
	AuthProvider         string `json:"auth_provider"` // Custom claim: How the user authenticated (local, google)
	Name                 string `json:"name"`          // Custom claim: User's name
}

// --- JWT Access Token Helper Functions ---

// GenerateJWT creates a new JWT access token for a user.
// Accepts the database user model, provider string, and application config.
func GenerateJWT(user *dbmodel.User, authProvider string, cfg *config.Config) (string, error) {
	if cfg.JWTSecretKey == "" {
		slog.Error("JWT generation failed: JWT_SECRET_KEY is not configured")
		return "", ErrMissingJWTSecret
	}
	jwtKey := []byte(cfg.JWTSecretKey)
	// Use cfg.JWTExpiry (time.Duration) directly from config struct
	expirationTime := time.Now().Add(cfg.JWTExpiry)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),                                     // Standard 'sub' claim is user ID
			IssuedAt:  jwt.NewNumericDate(time.Now()),                       // Standard 'iat' claim
			ExpiresAt: jwt.NewNumericDate(expirationTime),                   // Standard 'exp' claim
			Issuer:    cfg.JWTIssuer,                                        // Standard 'iss' claim from config
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)), // Optional: Allow for slight clock skew
		},
		Email:        user.Email,
		Role:         user.Role,
		AuthProvider: authProvider,
		Name:         user.Name,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtKey)
	if err != nil {
		slog.Error("Failed to sign JWT", slog.String("error", err.Error()), slog.String("userID", user.ID.String()))
		return "", fmt.Errorf("%w: could not sign token: %v", ErrTokenGeneration, err)
	}
	// slog.Debug("JWT generated successfully", slog.String("userID", user.ID.String())) // Keep logs in service layer generally
	return signedToken, nil
}

// VerifyJWT validates a JWT string using the application config and returns the claims if valid.
func VerifyJWT(tokenString string, cfg *config.Config) (*Claims, error) {
	if cfg.JWTSecretKey == "" {
		slog.Error("JWT verification failed: JWT_SECRET_KEY is not configured")
		return nil, ErrMissingJWTSecret
	}
	jwtKey := []byte(cfg.JWTSecretKey)

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// slog.Debug("JWT verification failed: token expired") // Log in calling function if needed
			return nil, ErrTokenExpired
		}
		// slog.Warn("JWT verification failed: invalid token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err) // Wrap specific parse error
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// slog.Debug("JWT verified successfully", slog.String("userID", claims.Subject))
		return claims, nil
	}

	// slog.Warn("JWT token was parsed but marked invalid")
	return nil, ErrTokenInvalid
}

// --- Refresh Token Helper Functions ---

// GenerateRefreshToken creates, stores via repository, and returns a new secure refresh token string.
func GenerateRefreshToken(ctx context.Context, userID uuid.UUID, repo repository.AuthRepository, cfg *config.Config) (string, error) {
	tokenString := uuid.NewString() // Using UUID string as the refresh token identifier

	// Use cfg.RefreshTokenExpiry (time.Duration) directly
	expiresAt := time.Now().Add(cfg.RefreshTokenExpiry)

	refreshToken := &dbmodel.RefreshToken{
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
	}

	if err := repo.CreateRefreshToken(ctx, refreshToken); err != nil {
		slog.ErrorContext(ctx, "Failed to store refresh token in repository", slog.String("userID", userID.String()), slog.String("error", err.Error()))
		return "", fmt.Errorf("%w: could not save refresh token: %v", ErrTokenGeneration, err)
	}

	slog.InfoContext(ctx, "Refresh token generated and stored", slog.String("userID", userID.String()))
	return tokenString, nil
}

// RefreshAccessToken validates a refresh token, rotates it, and issues a new pair of access/refresh tokens.
func RefreshAccessToken(ctx context.Context, refreshTokenString string, repo repository.AuthRepository, cfg *config.Config) (newAccessToken string, newRefreshToken string, err error) {
	l := slog.Default().With(slog.String("function", "RefreshAccessToken")) // Use default logger for helpers
	l.DebugContext(ctx, "Attempting to refresh access token")

	if refreshTokenString == "" {
		return "", "", ErrMissingRefreshToken
	}

	// 1. Find the refresh token
	refreshToken, err := repo.FindRefreshToken(ctx, refreshTokenString)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.WarnContext(ctx, "Refresh token not found in repository", slog.String("error", err.Error()))
			return "", "", ErrRefreshTokenNotFound
		}
		l.ErrorContext(ctx, "Failed to query refresh token from repository", slog.String("error", err.Error()))
		return "", "", fmt.Errorf("failed to retrieve refresh token data: %w", err)
	}

	// 2. Delete the used token *immediately* (Rotation Step 1)
	if delErr := repo.DeleteRefreshToken(ctx, refreshToken.ID); delErr != nil {
		l.ErrorContext(ctx, "CRITICAL: Failed to delete used refresh token", slog.String("tokenID", refreshToken.ID.String()), slog.String("error", delErr.Error()))
		return "", "", fmt.Errorf("failed to invalidate used refresh token: %w", delErr) // Fail hard if delete fails
	}
	l.DebugContext(ctx, "Used refresh token deleted", slog.String("tokenID", refreshToken.ID.String()))

	// 3. Check if the retrieved token *was* expired
	if time.Now().After(refreshToken.ExpiresAt) {
		l.WarnContext(ctx, "Attempted to use expired refresh token", slog.String("userID", refreshToken.UserID.String()))
		return "", "", ErrRefreshTokenExpired
	}

	// 4. Get user information
	user, err := repo.GetUserByID(ctx, refreshToken.UserID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get user associated with refresh token", slog.String("userID", refreshToken.UserID.String()), slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", ErrUserNotFound
		}
		return "", "", ErrGetUserInfo
	}

	// 5. Determine original AuthProvider for the new JWT
	authProvider := "local" // Default
	_, err = repo.GetOAuthProvider(ctx, user.ID, "google")
	if err == nil {
		authProvider = "google"
	} else if !errors.Is(err, repository.ErrNotFound) {
		l.WarnContext(ctx, "Error checking OAuth provider during refresh", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
	}

	// 6. Generate new access token
	newAccessToken, err = GenerateJWT(user, authProvider, cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	// 7. Generate a *new* refresh token (Rotation Step 2)
	newRefreshToken, err = GenerateRefreshToken(ctx, user.ID, repo, cfg)
	if err != nil {
		l.ErrorContext(ctx, "CRITICAL: Failed to generate new refresh token after deleting old one", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return "", "", fmt.Errorf("failed to complete token refresh cycle: %w", err) // Force re-login
	}

	l.InfoContext(ctx, "Access token refreshed successfully", slog.String("userID", user.ID.String()))
	return newAccessToken, newRefreshToken, nil
}

// generateSecureRandomString - kept if needed for other purposes, but UUID is fine for refresh token
func generateSecureRandomString(length int) (string, error) {
	byteLength := (length * 6) / 8
	if (length*6)%8 != 0 {
		byteLength++
	}
	randomBytes := make([]byte, byteLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(randomBytes)[:length], nil
}

// createAuthPayload is a helper to create the GraphQL response payload
func createAuthPayload(user *dbmodel.User, accessToken, refreshToken string) *graphmodel.AuthPayload {
	return &graphmodel.AuthPayload{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user, // CORRECTED: Assign the *dbmodel.User directly, as it's bound to the GraphQL User type
	}
}
