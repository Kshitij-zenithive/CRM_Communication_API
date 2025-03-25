// internal/middleware/auth.go (Corrected import, added GetProviderFromContext)

package middleware

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/internal/auth"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	userContextKey contextKey = "user"
)

// AuthMiddleware is a middleware function to authenticate users via JWT
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w,r) //Just skipping if there is no token
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Verify the token
			claims, err := auth.VerifyJWT(tokenString, cfg)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add user information to the context
			ctx := context.WithValue(r.Context(), userContextKey, claims)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves the user claims from the request context.
func GetUserFromContext(ctx context.Context) (*auth.Claims, error) {
	user, ok := ctx.Value(userContextKey).(*auth.Claims)
	if !ok {
		return nil, ErrUserNotInContext
	}
	return user, nil
}

var (
	ErrUserNotInContext = fmt.Errorf("user not found in context")
)

// GetProviderFromContext retrieves the authentication provider from context
func GetProviderFromContext(ctx context.Context) (string, error) {
	claims, ok := ctx.Value(userContextKey).(*auth.Claims)
	if !ok || claims == nil {
		return "", fmt.Errorf("user claims not found in context")
	}
	return claims.AuthProvider, nil
}

// RequireRole checks if the authenticated user has the specified role.
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims, err := GetUserFromContext(r.Context())
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            if claims.Role != requiredRole {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// GetUserIDFromContext retrieves the user ID from the request context.
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
    claims, ok := ctx.Value(userContextKey).(*auth.Claims)
    if !ok || claims == nil {
        return uuid.Nil, fmt.Errorf("user claims not found in context")
    }

    userID, err := uuid.Parse(claims.UserID)
    if err != nil {
        return uuid.Nil, fmt.Errorf("invalid user ID in claims: %w", err)
    }

    return userID, nil
}