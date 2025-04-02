// internal/middleware/auth.go
package middleware

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/internal/auth" // Import auth package

	// "fmt"
	"net/http"
	"strings"
	// "github.com/google/uuid" // No longer needed
)

// AuthMiddleware uses the new context key from the auth package
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ... (extract tokenString) ...
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}
			tokenString := parts[1]

			claims, err := auth.VerifyJWT(tokenString, cfg)
			if err != nil {
				next.ServeHTTP(w, r) // Proceed without setting claims if token is invalid/expired
				return
			}

			// Use the exported context key from the auth package
			ctx := context.WithValue(r.Context(), auth.ClaimsContextKey, claims) // CORRECTED: Use exported key
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks role using the claims from context.
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ContextGetClaims(r.Context()) // CORRECTED: Use helper from auth package
			if !ok {
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
