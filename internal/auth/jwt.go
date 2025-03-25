// internal/auth/jwt.go (Corrected function signatures and added Email to claims)

package auth

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

// Claims represents the JWT claims structure
type Claims struct {
	UserID       string `json:"user_id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	AuthProvider string `json:"auth_provider"`
	Email        string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a JWT token for a user
func GenerateJWT(user *model.User, authProvider string, cfg *config.Config) (string, error) {
	// Set expiration time
	expirationTime := time.Now().Add(cfg.JWTExpiry)

	// Create JWT claims
	claims := &Claims{
		UserID:       user.ID.String(),
		Name:         user.Name,
		Role:         user.Role,
		AuthProvider: authProvider,
		Email:        user.Email, // Include email
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret key
	tokenString, err := token.SignedString([]byte(cfg.JWTSecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// VerifyJWT validates a JWT token and returns the claims
func VerifyJWT(tokenString string, cfg *config.Config) (*Claims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	// Validate the claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateRefreshToken generates a new refresh token and stores it in the database
func GenerateRefreshToken(ctx context.Context, userID uuid.UUID, repo *repository.Repository, cfg *config.Config) (string, error) {
	// Generate a new UUID for the refresh token
	refreshTokenID := uuid.New()

	// Set expiration time for the refresh token
	expiresAt := time.Now().Add(cfg.RefreshTokenExpiry)

	// Create the refresh token in the database
	refreshToken := model.RefreshToken{
		ID:        refreshTokenID,
		UserID:    userID,
		Token:     refreshTokenID.String(), // Use the UUID as the token string
		ExpiresAt: expiresAt,
	}

	if err := repo.CreateRefreshToken(ctx, &refreshToken); err != nil {
		return "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return refreshToken.Token, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func RefreshAccessToken(ctx context.Context, refreshTokenString string, repo *repository.Repository, cfg *config.Config) (string, string, error) {
	// Get the refresh token from the database
	refreshToken, err := repo.GetRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if the refresh token has expired
	if time.Now().After(refreshToken.ExpiresAt) {
		// Optionally delete the expired refresh token
		_ = repo.DeleteRefreshToken(ctx, refreshTokenString)
		return "", "", errors.New("refresh token has expired")
	}
	// Get the user associated with the refresh token
	user, err := repo.GetUserByID(ctx, refreshToken.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}

	// Generate a new access token
	newAccessToken, err := GenerateJWT(user, "local", cfg) // Or derive the auth provider appropriately
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate a new refresh token
	newRefreshToken, err := GenerateRefreshToken(ctx, user.ID, repo, cfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	// Delete the old refresh token
	if err := repo.DeleteRefreshToken(ctx, refreshTokenString); err != nil {
		// Log the error, but don't block the process
		fmt.Printf("failed to delete old refresh token: %v\n", err)
	}

	return newAccessToken, newRefreshToken, nil
}
