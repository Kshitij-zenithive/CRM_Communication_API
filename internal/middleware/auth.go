package middleware

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/internal/auth" // Import auth package
	// "fmt"
	"net/http"
	"strings"

	// "github.com/google/uuid" // Keep uuid import
)


// AuthMiddleware uses the new context key from the auth package
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				// Proceed without setting claims if header format is wrong
				next.ServeHTTP(w, r)
				return
			}
			tokenString := parts[1]

			claims, err := auth.VerifyJWT(tokenString, cfg) // Use auth.VerifyJWT
			if err != nil {
				// Token is invalid or expired, proceed without setting claims
				next.ServeHTTP(w, r)
				return
			}

			// Use the exported context key from the auth package
			ctx := context.WithValue(r.Context(), auth.UserClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}


// RequireRole checks role using the claims from context.
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := auth.GetUserClaimsFromContext(r.Context()) // Use helper from auth package
			if err != nil {
				http.Error(w, "Unauthorized: Authentication required", http.StatusUnauthorized)
				return
			}
			if claims.Role != requiredRole {
				http.Error(w, "Forbidden: Insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}