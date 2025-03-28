// internal/auth/context.go
package auth

import (
	"context"
	dbmodel "crm-communication-api/internal/model" // Use alias if needed
	"fmt"

	"github.com/google/uuid"
)

// contextKey defines a type for context keys to avoid collisions.
type contextKey string

const (
	// userContextKey is the key for storing/retrieving the User object in context.
	userContextKey contextKey = "user"
	// claimsContextKey is the key for storing/retrieving JWT Claims in context.
	claimsContextKey contextKey = "claims"
)

// ContextSetUser adds the User object to the context.
func ContextSetUser(ctx context.Context, user *dbmodel.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// ContextGetUser retrieves the User object from the context.
// Returns the User and true if found, otherwise nil and false.
func ContextGetUser(ctx context.Context) (*dbmodel.User, bool) {
	user, ok := ctx.Value(userContextKey).(*dbmodel.User)
	return user, ok
}

// ContextRequireUser retrieves the User object from the context.
// Panics if the user is not found - use only when user presence is guaranteed (e.g., after auth middleware).
func ContextRequireUser(ctx context.Context) *dbmodel.User {
	user, ok := ContextGetUser(ctx)
	if !ok {
		// This indicates a programming error - auth middleware should have added the user.
		panic("auth: user not found in context where required")
	}
	return user
}


// ContextSetClaims adds the JWT Claims object to the context.
func ContextSetClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ContextGetClaims retrieves the JWT Claims object from the context.
// Returns the Claims and true if found, otherwise nil and false.
func ContextGetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

// ContextRequireClaims retrieves the Claims object from the context.
// Panics if claims are not found - use only when claims presence is guaranteed.
func ContextRequireClaims(ctx context.Context) *Claims {
    claims, ok := ContextGetClaims(ctx)
    if !ok {
        panic("auth: claims not found in context where required")
    }
    return claims
}

// ContextGetUserID retrieves the user ID (as UUID) from the context's claims.
// Returns the UUID and true if found and valid, otherwise zero UUID and false.
func ContextGetUserID(ctx context.Context) (uuid.UUID, bool) {
	claims, ok := ContextGetClaims(ctx)
	if !ok || claims.Subject == "" {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		// Log this? It implies Subject in a valid token wasn't a UUID.
		fmt.Printf("Error parsing userID from context claims: %v\n", err) // Replace with logger
		return uuid.Nil, false
	}
	return userID, true
}