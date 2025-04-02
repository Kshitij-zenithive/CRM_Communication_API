// package auth

// import (
// 	"context"
// 	"crm-communication-api/config"
// 	graphmodel "crm-communication-api/graph/model" // Use alias for graph model
// 	dbmodel "crm-communication-api/internal/model" // Use alias for DB model
// 	"crm-communication-api/internal/repository"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"net/http"

// 	"github.com/google/uuid"
// 	"golang.org/x/oauth2"
// 	"golang.org/x/oauth2/google"
// )

// type GoogleAuthService struct {
// 	Config     *config.Config
// 	Repository repository.AuthRepository // Use interface for flexibility
// 	oauthConf  *oauth2.Config
// }

// // Assert that GoogleAuthService implements the Service interface
// var _ Service = (*GoogleAuthService)(nil)

// // NewGoogleAuthService constructor using the AuthRepository interface
// func NewGoogleAuthService(cfg *config.Config, repo repository.AuthRepository) *GoogleAuthService {
// 	oauthConf := &oauth2.Config{
// 		ClientID:     cfg.GoogleClientID,
// 		ClientSecret: cfg.GoogleClientSecret,
// 		RedirectURL:  cfg.GoogleRedirectURL,
// 		Scopes: []string{
// 			"https://www.googleapis.com/auth/userinfo.email",
// 			"https://www.googleapis.com/auth/userinfo.profile",
// 			"https://mail.google.com/", // Make sure this scope is needed/approved
// 		},
// 		Endpoint: google.Endpoint,
// 	}
// 	return &GoogleAuthService{
// 		Config:     cfg,
// 		Repository: repo,
// 		oauthConf:  oauthConf,
// 	}
// }

// // GetAuthCodeURL returns the URL for Google OAuth authentication.
// func (s *GoogleAuthService) GetAuthCodeURL() string {
// 	// Consider adding state parameter generation and validation for security
// 	return s.oauthConf.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
// }

// // exchangeCode remains the same
// func (s *GoogleAuthService) exchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
// 	token, err := s.oauthConf.Exchange(ctx, code)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to exchange token: %w", err)
// 	}
// 	return token, nil
// }

// // getUserInfoFromGoogle remains the same
// func (s *GoogleAuthService) getUserInfoFromGoogle(ctx context.Context, token *oauth2.Token) (*GoogleUser, error) {
// 	client := s.oauthConf.Client(ctx, token)
// 	userInfoResp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user info: %w", err)
// 	}
// 	defer userInfoResp.Body.Close()

// 	data, err := io.ReadAll(userInfoResp.Body)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read user info body: %w", err)
// 	}
// 	if userInfoResp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("google API error (%d): %s", userInfoResp.StatusCode, string(data))
// 	}
// 	var googleUser GoogleUser
// 	if err := json.Unmarshal(data, &googleUser); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
// 	}
// 	return &googleUser, nil
// }

// // GoogleUser struct remains the same
// type GoogleUser struct {
// 	Sub           string `json:"sub"`
// 	Name          string `json:"name"`
// 	Email         string `json:"email"`
// 	Picture       string `json:"picture"`
// 	VerifiedEmail bool   `json:"verified_email"`
// }

// // AuthenticateGoogleUser handles the callback (uses dbmodel and graphmodel correctly)
// func (s *GoogleAuthService) AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error) {
// 	oauthToken, googleUser, err := s.exchangeAndGetUserInfo(ctx, code)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !googleUser.VerifiedEmail {
// 		return nil, fmt.Errorf("email not verified with Google")
// 	}

// 	// Use dbmodel.User here
// 	user, isNew, err := s.findOrCreateUserFromGoogle(ctx, googleUser)
// 	if err != nil {
// 		return nil, err
// 	}
//     // Update existing user details if missing
//     if !isNew && (user.Avatar == "" || user.Name == "") {
//         user.Avatar = googleUser.Picture
//         user.Name = googleUser.Name // Update name too if needed
//         if err := s.Repository.UpdateUser(ctx, user); err != nil {
//             fmt.Printf("Warning: failed to update avatar/name for user %s: %v\n", user.Email, err)
//         }
//     }

// 	// Use dbmodel.OAuthProvider here
// 	if err := s.saveOrUpdateOAuthProvider(ctx, user.ID, googleUser.Sub, oauthToken); err != nil {
// 		return nil, err
// 	}

// 	// Generate tokens using dbmodel.User
// 	accessToken, err := GenerateJWT(user, "google", s.Config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate access token: %w", err)
// 	}
// 	refreshToken, err := GenerateRefreshToken(ctx, user.ID, s.Repository, s.Config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
// 	}

// 	// Convert dbmodel.User to graphmodel.User for payload
// 	graphUser := &graphmodel.User{
// 		ID:    user.ID.String(),
// 		Name:  user.Name,
// 		Email: user.Email,
//         Avatar: &user.Avatar, // Assuming Avatar is string in dbmodel, pointer in graphmodel
// 		Role:  user.Role,
// 		// CreatedAt/UpdatedAt might not be in graphmodel.User, add if needed
// 	}

// 	// Return graphmodel.AuthPayload
// 	return &graphmodel.AuthPayload{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		User:         graphUser,
// 	}, nil
// }

// // exchangeAndGetUserInfo remains the same
// func (s *GoogleAuthService) exchangeAndGetUserInfo(ctx context.Context, code string) (*oauth2.Token, *GoogleUser, error) {
// 	token, err := s.exchangeCode(ctx, code)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	googleUser, err := s.getUserInfoFromGoogle(ctx, token)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	return token, googleUser, nil
// }

// // findOrCreateUserFromGoogle (Uses dbmodel.User)
// func (s *GoogleAuthService) findOrCreateUserFromGoogle(ctx context.Context, googleUser *GoogleUser) (*dbmodel.User, bool, error) {
// 	user, err := s.Repository.GetUserByEmail(ctx, googleUser.Email)
// 	if err == nil {
// 		return user, false, nil // Found
// 	}
// 	if !errors.Is(err, repository.ErrNotFound) { // Use errors.Is for specific error check
// 		return nil, false, fmt.Errorf("error checking for existing user: %w", err)
// 	}

// 	// Create new dbmodel.User
// 	newUser := &dbmodel.User{
// 		// ID will be generated by DB or GORM hook
// 		Name:   googleUser.Name,
// 		Email:  googleUser.Email,
// 		Avatar: googleUser.Picture,
// 		Role:   "user", // Assign default role
// 		// PasswordHash will be empty for OAuth users
// 	}
// 	if err := s.Repository.CreateUser(ctx, newUser); err != nil {
// 		return nil, true, fmt.Errorf("failed to create new user: %w", err)
// 	}
// 	return newUser, true, nil
// }

// // saveOrUpdateOAuthProvider (Uses dbmodel.OAuthProvider)
// func (s *GoogleAuthService) saveOrUpdateOAuthProvider(ctx context.Context, userID uuid.UUID, providerUserID string, token *oauth2.Token) error {
// 	oauthProvider, err := s.Repository.GetOAuthProvider(ctx, userID, "google")

// 	if errors.Is(err, repository.ErrNotFound) {
// 		// Create new dbmodel.OAuthProvider
// 		newProvider := &dbmodel.OAuthProvider{
// 			UserID:       userID,
// 			Provider:     "google",
// 			ProviderID:   providerUserID,
// 			AccessToken:  token.AccessToken,
// 			RefreshToken: token.RefreshToken, // Store the initial refresh token
// 			ExpiresAt:    token.Expiry,
// 		}
// 		if err := s.Repository.CreateOAuthProvider(ctx, newProvider); err != nil {
// 			return fmt.Errorf("failed to create oauth provider record: %w", err)
// 		}
// 	} else if err != nil {
// 		return fmt.Errorf("failed to query oauth provider record: %w", err)
// 	} else {
// 		// Update existing dbmodel.OAuthProvider
// 		oauthProvider.AccessToken = token.AccessToken
// 		oauthProvider.ExpiresAt = token.Expiry
// 		if token.RefreshToken != "" { // Only update if Google provides a new one
// 			oauthProvider.RefreshToken = token.RefreshToken
// 		}
// 		if err := s.Repository.UpdateOAuthProvider(ctx, oauthProvider); err != nil {
// 			return fmt.Errorf("failed to update oauth provider record: %w", err)
// 		}
// 	}
// 	return nil
// }

// // --- Interface Methods Implementation ---

// // Login is not implemented by GoogleAuthService
// func (s *GoogleAuthService) Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error) {
// 	return nil, fmt.Errorf("google auth service does not handle local login")
// }

// // RefreshToken uses shared logic and converts user model
// func (s *GoogleAuthService) RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error) {
// 	accessToken, newRefreshToken, err := RefreshAccessToken(ctx, refreshToken, s.Repository, s.Config)
// 	if err != nil {
// 		return nil, err // Error already formatted
// 	}

// 	// Verify new token to get claims (including UserID)
// 	claims, err := VerifyJWT(accessToken, s.Config)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to verify newly created access token: %w", err)
// 	}
// 	userID, _ := uuid.Parse(claims.UserID) // Error already checked by VerifyJWT basically

// 	// Fetch dbmodel.User
// 	user, err := s.Repository.GetUserByID(ctx, userID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve user during refresh: %w", err)
// 	}

// 	// Convert to graphmodel.User
// 	graphUser := &graphmodel.User{
// 		ID:    user.ID.String(),
// 		Name:  user.Name,
// 		Email: user.Email,
//         Avatar: &user.Avatar,
// 		Role:  user.Role,
// 	}

// 	// Return graphmodel.AuthPayload
// 	return &graphmodel.AuthPayload{
// 		AccessToken:  accessToken,
// 		RefreshToken: newRefreshToken,
// 		User:         graphUser,
// 	}, nil
// }

// // GetCurrentUser uses context helper and repository (returns dbmodel.User)
// func (s *GoogleAuthService) GetCurrentUser(ctx context.Context) (*dbmodel.User, error) {
// 	// Use the helper from auth/context.go
// 	userID, err := GetUserIDFromContext(ctx)
// 	if err != nil {
// 		return nil, err // Pass through ErrUserNotInContext or parse error
// 	}

// 	user, err := s.Repository.GetUserByID(ctx, userID)
// 	if err != nil {
// 		// Handle case where user in token doesn't exist in DB anymore
// 		if errors.Is(err, repository.ErrNotFound) {
// 			return nil, fmt.Errorf("authenticated user not found in database")
// 		}
// 		return nil, fmt.Errorf("failed to retrieve user from repository: %w", err)
// 	}
// 	return user, nil
// }

// // GetGmailService remains specific to Google Auth, implementation unchanged
// func (s *GoogleAuthService) GetGmailService(ctx context.Context, userID string) (*http.Client, error) {
// 	// Note: Repository might need adjustment if GetGmailToken isn't part of AuthRepository interface
// 	// For now, assume it is or handle type assertion if needed.
// 	// tokenJSON, err := s.Repository.GetGmailToken(ctx, userID) // This might need adjustment based on repo interface

//     // Assuming GetOAuthProvider gives enough info
//     providerInfo, err := s.Repository.GetOAuthProvider(ctx, uuid.MustParse(userID), "google")
//     if err != nil {
//          return nil, fmt.Errorf("failed to get google provider token info for user %s: %w", userID, err)
//     }

//     token := &oauth2.Token{
//         AccessToken:  providerInfo.AccessToken,
//         RefreshToken: providerInfo.RefreshToken,
//         Expiry:       providerInfo.ExpiresAt,
//     }

// 	// Create an OAuth2 HTTP client. It automatically handles token refreshes.
// 	return s.oauthConf.Client(ctx, token), nil
// }

// internal/auth/google.go
package auth

import (
	"context"
	"crypto/rand"     // Needed for state generation helper
	"encoding/base64" // Needed for state generation helper
	"encoding/json"
	"errors" // CORRECTED: Added import
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"crm-communication-api/config"
	graphmodel "crm-communication-api/graph/model" // CORRECTED: Added alias
	dbmodel "crm-communication-api/internal/model" // CORRECTED: Added alias
	"crm-communication-api/internal/repository"

	"github.com/golang-jwt/jwt/v5" // CORRECTED: Added for state signing helpers
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// CORRECTED: Define constant at package level
const oauthStateTokenLifespan = 10 * time.Minute

// GoogleAuthService implements the auth.Service interface for Google OAuth.
type GoogleAuthService struct {
	repo              repository.AuthRepository // Use interface
	config            *config.Config
	logger            *slog.Logger
	googleOAuthConfig *oauth2.Config
	jwtSigningKey     []byte
}

// Assert that GoogleAuthService implements the Service interface
var _ Service = (*GoogleAuthService)(nil)

// NewGoogleAuthService constructor using the AuthRepository interface
func (s *GoogleAuthService) IsConfigured() bool {
	return s.googleOAuthConfig != nil && s.config.GoogleClientID != "" && s.config.GoogleClientSecret != ""
}
func NewGoogleAuthService(repo repository.AuthRepository, cfg *config.Config, logger *slog.Logger) (*GoogleAuthService, error) {
	if cfg.JWTSecretKey == "" { // Correct field name
		return nil, ErrMissingJWTSecret
	}
	jwtKey := []byte(cfg.JWTSecretKey) // Correct field name

	var googleOAuthConfig *oauth2.Config
	// ... (Google OAuth config setup remains the same) ...
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		googleOAuthConfig = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"openid", "profile", "email", "https://mail.google.com/"},
			Endpoint:     google.Endpoint,
		}
	} else {
		logger.Warn("Google OAuth not configured for GoogleAuthService")
	}

	return &GoogleAuthService{
		repo:              repo,
		config:            cfg,
		logger:            logger.With(slog.String("service", "GoogleAuthService")),
		googleOAuthConfig: googleOAuthConfig,
		jwtSigningKey:     jwtKey, // Store key for state signing
	}, nil
}

// --- Service Interface Methods ---

// Login - Not Implemented for GoogleAuthService
func (s *GoogleAuthService) Login(ctx context.Context, email, password string) (*graphmodel.AuthPayload, error) {
	return nil, fmt.Errorf("google auth service does not handle local login")
}

// AuthenticateGoogleUser handles the callback from Google OAuth.
func (s *GoogleAuthService) AuthenticateGoogleUser(ctx context.Context, code string) (*graphmodel.AuthPayload, error) {
	l := s.logger.With(slog.String("method", "AuthenticateGoogleUser"))
	l.Info("Handling Google OAuth callback")

	if s.googleOAuthConfig == nil {
		l.Error("Google OAuth is not configured")
		return nil, ErrMissingGoogleConfig
	}

	// TODO: Add state verification Step
	// state := r.URL.Query().Get("state") // Get state from actual HTTP request context if possible
	// if err := s.verifyOAuthState(state); err != nil { return nil, err }
	l.Debug("OAuth state verified (placeholder)")

	// Exchange code
	googleToken, err := s.googleOAuthConfig.Exchange(ctx, code, oauth2.AccessTypeOffline) // Use Exchange directly
	if err != nil {
		l.Error("Failed to exchange Google auth code", slog.String("error", err.Error()))
		return nil, ErrOAuthCodeExchange
	}
	if !googleToken.Valid() {
		l.Error("Received invalid token from Google exchange")
		return nil, ErrOAuthCodeExchange
	}
	l.Debug("Google code exchanged successfully")

	// Get user info
	userInfo, err := s.getUserInfoFromGoogle(ctx, googleToken) // Use helper method
	if err != nil {
		return nil, err
	}
	l.Debug("Retrieved Google user info", slog.String("googleID", userInfo.ID), slog.String("email", userInfo.Email))
	if !userInfo.EmailVerified {
		return nil, fmt.Errorf("google email (%s) is not verified", userInfo.Email)
	}

	// Find or create user (Uses dbmodel.User)
	user, err := s.findOrCreateUserFromGoogleInfo(ctx, userInfo) // Use helper method
	if err != nil {
		return nil, err
	}

	// Save/Update OAuth Provider info (Uses dbmodel.OAuthProvider)
	providerData := &dbmodel.OAuthProvider{
		UserID:       user.ID,
		Provider:     "google",
		ProviderID:   userInfo.ID,
		AccessToken:  googleToken.AccessToken,
		RefreshToken: googleToken.RefreshToken,
		ExpiresAt:    googleToken.Expiry,
	}
	if err := s.repo.CreateOrUpdateOAuthProvider(ctx, providerData); err != nil {
		l.Error("Failed to save Google OAuth provider data", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		// Log error but proceed with login for now
	} else {
		l.Debug("Saved Google OAuth provider data", slog.String("userID", user.ID.String()))
	}

	// Generate App Tokens using package-level helpers
	accessToken, err := GenerateJWT(user, "google", s.config) // Pass dbmodel.User
	if err != nil {
		return nil, fmt.Errorf("google login failed during access token generation: %w", err)
	}

	// Pass repo interface to GenerateRefreshToken
	refreshToken, err := GenerateRefreshToken(ctx, user.ID, s.repo, s.config)
	if err != nil {
		return nil, fmt.Errorf("google login failed during refresh token generation: %w", err)
	}

	l.Info("User logged in successfully via Google OAuth", slog.String("userID", user.ID.String()))
	return createAuthPayload(user, accessToken, refreshToken), nil // Use helper
}

// RefreshToken uses shared package-level logic.
func (s *GoogleAuthService) RefreshToken(ctx context.Context, refreshToken string) (*graphmodel.AuthPayload, error) {
	// Pass repo interface directly to the package-level helper
	newAccessToken, newRefreshToken, err := RefreshAccessToken(ctx, refreshToken, s.repo, s.config)
	if err != nil {
		return nil, err
	}

	// Verify new token to get claims using package-level helper
	claims, err := VerifyJWT(newAccessToken, s.config)
	if err != nil {
		s.logger.Error("Failed to verify newly generated access token", slog.String("error", err.Error()))
		return nil, fmt.Errorf("internal error verifying refreshed token: %w", err)
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		s.logger.Error("Invalid UserID in refreshed token claims", slog.String("subject", claims.Subject), slog.String("error", err.Error()))
		return nil, fmt.Errorf("internal error processing refreshed token claims")
	}

	// Fetch dbmodel.User
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user during refresh: %w", err)
	}

	return createAuthPayload(user, newAccessToken, newRefreshToken), nil // Use helper
}

// GetCurrentUser uses context helper and repository (returns dbmodel.User)
func (s *GoogleAuthService) GetCurrentUser(ctx context.Context) (*dbmodel.User, error) {
	// Use the helper from auth/context.go
	userID, ok := ContextGetUserID(ctx) // CORRECTED function name call
	if !ok {
		// Error retrieving or parsing user ID from context claims
		s.logger.WarnContext(ctx, "GetCurrentUser: could not get valid user ID from context")
		// Check claims directly for better error message
		_, claimsOk := ContextGetClaims(ctx)
		if !claimsOk {
			return nil, ErrUserNotInContext
		}
		return nil, fmt.Errorf("invalid user ID format in context claims")
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to retrieve authenticated user from repository", slog.String("userID", userID.String()), slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound // User in token claims doesn't exist in DB
		}
		return nil, fmt.Errorf("internal error retrieving user: %w", err)
	}
	return user, nil
}

// GetAuthCodeURL generates the Google OAuth login URL.
func (s *GoogleAuthService) GetAuthCodeURL() string {
	if !s.IsConfigured() { // Use the helper check
		s.logger.Error("Attempted to get Google Auth URL, but OAuth is not configured")
		return "" // Return empty string if not configured
	}
	// For production, USE generateOAuthState and verifyOAuthState with signed JWTs
	state := "pseudo-random-state-for-dev" // Placeholder - IMPLEMENT SECURE STATE LATER
	return s.googleOAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	// Generate and sign state for CSRF protection
	signedState, err := s.generateOAuthState()
	if err != nil {
		// Log error and return URL without state (less secure) or return empty string/error
		return s.googleOAuthConfig.AuthCodeURL("fallback-state", oauth2.AccessTypeOffline) // Less secure fallback
	}
	return s.googleOAuthConfig.AuthCodeURL(signedState, oauth2.AccessTypeOffline)
}

// GetGmailService retrieves authenticated client for Gmail API.
func (s *GoogleAuthService) GetGmailService(ctx context.Context, userID string) (*http.Client, error) {
	l := s.logger.With(slog.String("method", "GetGmailService"), slog.String("userID", userID))
	if s.googleOAuthConfig == nil {
		return nil, ErrMissingGoogleConfig
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}

	providerInfo, err := s.repo.GetOAuthProvider(ctx, userUUID, "google")
	if err != nil {
		l.Error("Failed to get Google provider info", slog.String("error", err.Error()))
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("user has not linked their Google account")
		}
		return nil, fmt.Errorf("failed to retrieve google provider info: %w", err)
	}
	if providerInfo.AccessToken == "" {
		l.Error("Stored Google provider info missing access token")
		return nil, fmt.Errorf("google account linked, but access token missing")
	}

	token := &oauth2.Token{
		AccessToken:  providerInfo.AccessToken,
		RefreshToken: providerInfo.RefreshToken,
		Expiry:       providerInfo.ExpiresAt,
		TokenType:    "Bearer",
	}

	// Create client that automatically refreshes token
	tokenSource := s.googleOAuthConfig.TokenSource(ctx, token)
	httpClient := oauth2.NewClient(ctx, tokenSource)
	l.Debug("Created authenticated Gmail HTTP client")
	return httpClient, nil
}

// VerifyJWT delegates to the package-level helper in jwt.go
func (s *GoogleAuthService) VerifyJWT(tokenString string) (*Claims, error) {
	return VerifyJWT(tokenString, s.config)
}

// --- Internal Helper Methods ---

// GoogleUserInfo struct definition
type GoogleUserInfo struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string `json:"locale"`
}

// generateOAuthState creates a secure, signed state token (JWT).
// CORRECTED: Receiver is *GoogleAuthService
func (s *GoogleAuthService) generateOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil { /* ... logging ... */
		return "", ErrOAuthStateGeneration
	}
	stateNonce := base64.URLEncoding.EncodeToString(b)
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   stateNonce,
		ExpiresAt: jwt.NewNumericDate(now.Add(oauthStateTokenLifespan)), // Use defined const
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    "crm-communication-api/oauth-state",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedState, err := token.SignedString(s.jwtSigningKey) // Use service key
	if err != nil {                                         /* ... logging ... */
		return "", ErrOAuthStateGeneration
	}
	return signedState, nil
}

// verifyOAuthState validates the signed state token (JWT).
// CORRECTED: Receiver is *GoogleAuthService
func (s *GoogleAuthService) verifyOAuthState(signedState string) error {
	// ... (implementation remains the same, uses s.jwtSigningKey) ...
	token, err := jwt.ParseWithClaims(signedState, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method in state token: %v", token.Header["alg"])
		}
		return s.jwtSigningKey, nil
	})
	if err != nil { /* ... logging ... */
		return ErrOAuthStateMismatch
	}
	if _, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return nil
	}
	return ErrOAuthStateMismatch
}

// getUserInfoFromGoogle retrieves user details from Google's userinfo endpoint.
// CORRECTED: Receiver is *GoogleAuthService
func (s *GoogleAuthService) getUserInfoFromGoogle(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	if s.googleOAuthConfig == nil {
		return nil, ErrMissingGoogleConfig
	} // Check needed here too
	client := s.googleOAuthConfig.Client(ctx, token)
	userInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo"
	resp, err := client.Get(userInfoURL)
	if err != nil { /* ... logging ... */
		return nil, ErrOAuthGetUserInfo
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { /* ... logging ... */
		return nil, ErrOAuthGetUserInfo
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil { /* ... logging ... */
		return nil, ErrOAuthGetUserInfo
	}
	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil { /* ... logging ... */
		return nil, ErrOAuthGetUserInfo
	}
	if userInfo.Email == "" || userInfo.ID == "" { /* ... logging ... */
		return nil, ErrOAuthGetUserInfo
	}
	return &userInfo, nil
}

// findOrCreateUserFromGoogleInfo finds an existing user by email or creates a new one.
// CORRECTED: Receiver is *GoogleAuthService
func (s *GoogleAuthService) findOrCreateUserFromGoogleInfo(ctx context.Context, userInfo *GoogleUserInfo) (*dbmodel.User, error) {
	user, err := s.repo.GetUserByEmail(ctx, userInfo.Email)
	if err == nil {
		return user, nil
	} // Found
	if !errors.Is(err, repository.ErrNotFound) { /* ... logging ... */
		return nil, fmt.Errorf("db error checking user: %w", err)
	}

	// Create
	newUser := &dbmodel.User{ // Use dbmodel
		Name:   userInfo.Name,
		Email:  strings.ToLower(userInfo.Email),
		Avatar: userInfo.Picture,
		Role:   "user",
		// Password field remains empty/null for OAuth user
	}
	if err := s.repo.CreateUser(ctx, newUser); err != nil { /* ... logging ... */
		return nil, fmt.Errorf("failed to save new user from google auth: %w", err)
	}
	s.logger.InfoContext(ctx, "Created new user from Google info successfully", slog.String("userID", newUser.ID.String()))
	return newUser, nil
}
