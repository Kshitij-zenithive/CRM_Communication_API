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
	dbmodel "crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	// "github.com/google/uuid"
	"golang.org/x/oauth2"
)

// GoogleUserInfo represents the user information retrieved from Google.
type GoogleUserInfo struct {
	ID            string `json:"sub"` // Use 'sub' as the standard unique identifier
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"` // URL to profile picture
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Locale        string `json:"locale"`
}

// generateOAuthState creates a secure, signed state token (JWT).
func (s *AuthService) generateOAuthState() (string, error) {
	// Generate some random bytes for uniqueness
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		s.logger.Error("Failed to generate random bytes for OAuth state", slog.String("error", err.Error()))
		return "", ErrOAuthStateGeneration
	}
	stateNonce := base64.URLEncoding.EncodeToString(b)

	// Create a short-lived JWT to act as the state parameter
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   stateNonce, // Embed random nonce
		ExpiresAt: jwt.NewNumericDate(now.Add(oauthStateTokenLifespan)),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    "crm-communication-api/oauth-state", // Specific issuer for state tokens
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedState, err := token.SignedString(s.jwtSigningKey)
	if err != nil {
		s.logger.Error("Failed to sign OAuth state token", slog.String("error", err.Error()))
		return "", ErrOAuthStateGeneration
	}
	return signedState, nil
}

// verifyOAuthState validates the signed state token (JWT).
func (s *AuthService) verifyOAuthState(signedState string) error {
	s.logger.Debug("Verifying OAuth state token")
	token, err := jwt.ParseWithClaims(signedState, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method in state token: %v", token.Header["alg"])
		}
		return s.jwtSigningKey, nil
	})

	if err != nil {
		s.logger.Warn("OAuth state token verification failed", slog.String("error", err.Error()))
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrOAuthStateMismatch // Treat expired state as mismatch
		}
		return ErrOAuthStateMismatch // Treat any other error as mismatch
	}

	if _, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		s.logger.Debug("OAuth state token verified successfully")
		return nil // State is valid
	}

	s.logger.Warn("OAuth state token parsed but marked invalid")
	return ErrOAuthStateMismatch
}

// GenerateGoogleLoginURL generates the URL for the Google OAuth consent screen.
// It includes a signed state parameter to prevent CSRF.
func (s *AuthService) GenerateGoogleLoginURL() (string, error) {
	if s.googleOAuthConfig == nil {
		s.logger.Error("Attempted to generate Google login URL but OAuth is not configured")
		return "", ErrMissingGoogleConfig
	}

	signedState, err := s.generateOAuthState()
	if err != nil {
		return "", err // Error already logged
	}

	// Generate the URL with PKCE options for enhanced security if desired/supported
	// opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline, oauth2.ApprovalForce} // Example options
	// url := s.googleOAuthConfig.AuthCodeURL(signedState, opts...)
	url := s.googleOAuthConfig.AuthCodeURL(signedState, oauth2.AccessTypeOffline) // Request offline access for refresh token

	s.logger.Debug("Generated Google login URL")
	return url, nil
}

// HandleGoogleCallback processes the callback from Google after user consent.
// It verifies state, exchanges code for tokens, gets user info, finds/creates user,
// saves OAuth provider details, and issues application tokens (access + refresh).
func (s *AuthService) HandleGoogleCallback(ctx context.Context, state, code string) (accessToken string, refreshToken string, err error) {
	s.logger.Info("Handling Google OAuth callback")

	if s.googleOAuthConfig == nil {
		s.logger.Error("Attempted Google callback handling but OAuth is not configured")
		return "", "", ErrMissingGoogleConfig
	}

	// 1. Verify OAuth State
	if err := s.verifyOAuthState(state); err != nil {
		// Error already logged by verifyOAuthState
		return "", "", err // Return specific state mismatch error
	}
	s.logger.Debug("OAuth state verified")


	// 2. Exchange authorization code for tokens
	// Use context for potential cancellation during HTTP request
	googleToken, err := s.googleOAuthConfig.Exchange(ctx, code)
	if err != nil {
		s.logger.Error("Failed to exchange Google auth code for token", slog.String("error", err.Error()))
		return "", "", ErrOAuthCodeExchange
	}
	if !googleToken.Valid() {
		s.logger.Error("Received invalid token from Google OAuth exchange")
		return "", "", ErrOAuthCodeExchange
	}
	s.logger.Debug("Google code exchanged for token successfully")

	// 3. Get user information from Google using the access token
	userInfo, err := s.getUserInfoFromGoogle(ctx, googleToken)
	if err != nil {
		// Error already logged by helper
		return "", "", err
	}
	s.logger.Debug("Retrieved user info from Google", slog.String("googleID", userInfo.ID), slog.String("email", userInfo.Email))

	// 4. Find or Create User in local database
	user, err := s.findOrCreateUserFromGoogleInfo(ctx, userInfo)
	if err != nil {
		// Error logged by helper
		return "", "", err
	}

	// 5. Create or Update OAuthProvider details
	providerData := &dbmodel.OAuthProvider{
		UserID:       user.ID,
		Provider:     "google",
		ProviderID:   userInfo.ID,
		AccessToken:  googleToken.AccessToken, // Consider encrypting these tokens at rest
		RefreshToken: googleToken.RefreshToken, // May be empty if not requested or previously granted
		ExpiresAt:    googleToken.Expiry,
	}
	if err := s.repo.CreateOrUpdateOAuthProvider(ctx, providerData); err != nil {
		s.logger.Error("Failed to save Google OAuth provider data", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		// This is problematic but maybe not fatal for login? Log critically.
		// Allow login but future API calls requiring Google token refresh might fail.
		// return "", "", ErrOAuthProviderUpdate
	} else {
		s.logger.Debug("Saved Google OAuth provider data", slog.String("userID", user.ID.String()))
	}

	// --- Authentication successful (User linked/created) ---

	// 6. Generate application's access and refresh tokens
	accessToken, err = s.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate access token after Google login", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		return "", "", fmt.Errorf("google login succeeded but failed to issue access token: %w", err)
	}

	refreshToken, err = s.GenerateRefreshToken(ctx, user.ID)
	if err != nil {
		s.logger.Error("Failed to generate refresh token after Google login", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
		// Allow login with access token, but log critical failure for refresh token
		return accessToken, "", fmt.Errorf("google login succeeded but failed to issue refresh token: %w", err)
	}

	s.logger.Info("User logged in successfully via Google OAuth", slog.String("userID", user.ID.String()))
	return accessToken, refreshToken, nil
}

// getUserInfoFromGoogle retrieves user details from Google's userinfo endpoint.
func (s *AuthService) getUserInfoFromGoogle(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := s.googleOAuthConfig.Client(ctx, token)
	userInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo" // Standard OIDC endpoint

	resp, err := client.Get(userInfoURL)
	if err != nil {
		s.logger.Error("Failed to request user info from Google", slog.String("error", err.Error()))
		return nil, ErrOAuthGetUserInfo
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for debugging
		s.logger.Error("Received non-OK status from Google userinfo endpoint",
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(bodyBytes)))
		return nil, ErrOAuthGetUserInfo
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read Google user info response body", slog.String("error", err.Error()))
		return nil, ErrOAuthGetUserInfo
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		s.logger.Error("Failed to unmarshal Google user info JSON", slog.String("error", err.Error()))
		return nil, ErrOAuthGetUserInfo
	}

	// Optional: Check if email is verified by Google
	if !userInfo.EmailVerified {
		s.logger.Warn("Google account email is not verified", slog.String("email", userInfo.Email))
		// Decide policy: Allow login or return an error?
		// return nil, fmt.Errorf("google email not verified")
	}
	if userInfo.Email == "" || userInfo.ID == "" {
		s.logger.Error("Missing essential user info (email or sub) from Google")
		return nil, ErrOAuthGetUserInfo
	}


	return &userInfo, nil
}

// findOrCreateUserFromGoogleInfo finds an existing user by email or creates a new one.
func (s *AuthService) findOrCreateUserFromGoogleInfo(ctx context.Context, userInfo *GoogleUserInfo) (*dbmodel.User, error) {
	// Try finding user by email first
	user, err := s.repo.GetUserByEmail(ctx, userInfo.Email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		s.logger.Error("Failed checking for existing user during Google auth", slog.String("email", userInfo.Email), slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error while checking user: %w", err)
	}

	// User found by email - return existing user
	if user != nil {
		s.logger.Debug("Found existing user by email during Google auth", slog.String("userID", user.ID.String()), slog.String("email", user.Email))
		// Optional: Update user's avatar or name from Google info if desired?
		// user.Avatar = userInfo.Picture
		// user.Name = userInfo.Name
		// if errUpdate := s.repo.UpdateUser(ctx, user); errUpdate != nil { ... log error ... }
		return user, nil
	}

	// User not found - create a new user
	s.logger.Info("User not found by email, creating new user from Google info", slog.String("email", userInfo.Email))
	newUser := &dbmodel.User{
		// ID will be generated by DB
		Name:     userInfo.Name,
		Email:    strings.ToLower(userInfo.Email),
		Avatar:   userInfo.Picture,
		Role:     "user", // Default role
		Password: "",   // No password for OAuth-only user initially
	}

	if err := s.repo.CreateUser(ctx, newUser); err != nil {
		s.logger.Error("Failed to create new user from Google info", slog.String("email", newUser.Email), slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to save new user from google auth: %w", err)
	}

	s.logger.Info("Created new user from Google info successfully", slog.String("userID", newUser.ID.String()), slog.String("email", newUser.Email))
	return newUser, nil
}