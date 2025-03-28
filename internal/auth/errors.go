// internal/auth/errors.go
package auth

import "errors"

var (
	// Authentication / Credential Errors
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrPasswordHashing      = errors.New("failed to hash password")
	ErrPasswordMismatch     = errors.New("password does not match") // Specific bcrypt mismatch
	ErrInvalidAuthHeader    = errors.New("invalid authorization header format")
	ErrMissingAuthHeader    = errors.New("missing authorization header")

	// User Related Errors
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailTaken           = errors.New("email address is already taken")
	ErrGetUserInfo          = errors.New("failed to retrieve user information")

	// Token Errors
	ErrTokenInvalid         = errors.New("token is invalid or malformed")
	ErrTokenExpired         = errors.New("token has expired")
	ErrTokenVerification    = errors.New("failed to verify token")
	ErrTokenGeneration      = errors.New("failed to generate token")
	ErrRefreshTokenNotFound = errors.New("refresh token not found or already used")
	ErrRefreshTokenExpired  = errors.New("refresh token has expired")
	ErrMissingRefreshToken  = errors.New("refresh token is missing")

	// OAuth Errors
	ErrOAuthStateMismatch   = errors.New("oauth state parameter mismatch")
	ErrOAuthCodeExchange    = errors.New("failed to exchange oauth code for token")
	ErrOAuthGetUserInfo     = errors.New("failed to get user info from oauth provider")
	ErrOAuthProviderUpdate  = errors.New("failed to save or update oauth provider details")
	ErrOAuthStateGeneration = errors.New("failed to generate oauth state")
	ErrOAuthStateVerify     = errors.New("failed to verify oauth state token")

	// Configuration Errors
	ErrMissingJWTSecret     = errors.New("JWT secret key is not configured")
	ErrMissingGoogleConfig  = errors.New("Google OAuth client ID or secret is not configured")
)