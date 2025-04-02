// internal/auth/context.go
package auth

import (
	"context"
	// dbmodel "crm-communication-api/internal/model"
	"fmt"

	"github.com/google/uuid"
)

type contextKey string

const (
	// UserContextKey is for storing/retrieving the full User dbmodel object (less common).
	// UserContextKey contextKey = "user"
	// ClaimsContextKey is the key for storing/retrieving JWT Claims in context. Exported for middleware use.
	ClaimsContextKey contextKey = "claims" // CORRECTED: Exported (Capitalized)
)

var (
	ErrUserNotInContext = fmt.Errorf("user claims not found in context")
)

// ContextSetClaims adds the JWT Claims object to the context using the exported key.
func ContextSetClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims) // Use exported key
}

// ContextGetClaims retrieves the JWT Claims object from the context using the exported key.
func ContextGetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims) // Use exported key
	return claims, ok
}

// ContextRequireClaims retrieves the Claims object from the context. Panics if not found.
func ContextRequireClaims(ctx context.Context) *Claims {
	claims, ok := ContextGetClaims(ctx)
	if !ok {
		panic("auth: claims not found in context where required")
	}
	return claims
}

// ContextGetUserID retrieves the user ID (as UUID) from the context's claims.
func ContextGetUserID(ctx context.Context) (uuid.UUID, bool) {
	claims, ok := ContextGetClaims(ctx) // This function now correctly uses the exported key internally
	if !ok || claims.Subject == "" {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		fmt.Printf("Error parsing userID from context claims: %v\n", err) // Replace with logger
		return uuid.Nil, false
	}
	return userID, true
}

// --- User object context functions (Keep if needed, but claims are usually sufficient) ---

/*
// ContextSetUser adds the User object to the context.
func ContextSetUser(ctx context.Context, user *dbmodel.User) context.Context {
	// If using this, define and export UserContextKey = "user"
	return context.WithValue(ctx, UserContextKey, user)
}

// ContextGetUser retrieves the User object from the context.
func ContextGetUser(ctx context.Context) (*dbmodel.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*dbmodel.User)
	return user, ok
}

// ContextRequireUser retrieves the User object from the context. Panics if not found.
func ContextRequireUser(ctx context.Context) *dbmodel.User {
	user, ok := ContextGetUser(ctx)
	if !ok {
		panic("auth: user not found in context where required")
	}
	return user
}
*/
