// internal/auth/google.go (Corrected import and AuthenticateUser method)

package auth

import (
	"context"
	"crm-communication-api/config"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleAuthService struct {
	Config     *config.Config
	Repository *repository.Repository
	oauthConf  *oauth2.Config // OAuth configuration
}

func NewGoogleAuthService(cfg *config.Config, repo *repository.Repository) *GoogleAuthService {
	// Initialize the OAuth2 config
	oauthConf := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://mail.google.com/", // Gmail scope
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleAuthService{
		Config:     cfg,
		Repository: repo,
		oauthConf:  oauthConf,
	}
}

// GetAuthCodeURL returns the URL to redirect the user to for Google OAuth2 authentication.
func (s *GoogleAuthService) GetAuthCodeURL() string {
	// Generate the URL for the Google OAuth2 consent page
	return s.oauthConf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// Exchange exchanges the authorization code for an access token and user information.
func (s *GoogleAuthService) Exchange(ctx context.Context, code string) (*oauth2.Token, *GoogleUser, error) {
	// Exchange the authorization code for an access token
	token, err := s.oauthConf.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Fetch user information from Google
	client := s.oauthConf.Client(ctx, token)
	userInfo, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer userInfo.Body.Close()

	data, err := io.ReadAll(userInfo.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read user info: %w", err)
	}

	var googleUser GoogleUser
	if err := json.Unmarshal(data, &googleUser); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return token, &googleUser, nil
}

// GoogleUser represents the user information returned by Google's OAuth2 API.
type GoogleUser struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// AuthenticateUser handles the Google OAuth flow, user creation/update, and JWT generation
func (s *GoogleAuthService) AuthenticateUser(ctx context.Context, code string) (string, string, error) {
    token, googleUser, err := s.Exchange(ctx, code)
    if err != nil {
        return "", "", fmt.Errorf("failed to exchange code: %w", err)
    }

    // Check if the user's email is verified
    if !googleUser.VerifiedEmail {
        return "", "", fmt.Errorf("email not verified with Google")
    }

    // Find or create the user in the database
    user, err := s.Repository.GetUserByEmail(ctx, googleUser.Email)
	if err != nil && err != repository.ErrNotFound {
		return "", "", fmt.Errorf("error getting user: %w", err)
	}
    if err == repository.ErrNotFound {
        // Create a new user
        user = &model.User{
            Name:     googleUser.Name,
            Email:    googleUser.Email,
            Avatar:   googleUser.Picture,
            Role:     "user", // Set a default role
        }
        if err := s.Repository.CreateUser(ctx, user); err != nil {
            return "", "", fmt.Errorf("failed to create user: %w", err)
        }
    } else {
        // Update existing user
        user.Name = googleUser.Name
        user.Avatar = googleUser.Picture
        if err := s.Repository.UpdateUser(ctx, user); err != nil {
            return "", "", fmt.Errorf("failed to update user: %w", err)
        }
    }


    // Find or create an OAuthProvider record
    oauthProvider, err := s.Repository.GetOAuthProvider(ctx, user.ID, "google")
    if err != nil && err != repository.ErrNotFound {
        return "", "", fmt.Errorf("failed to get OAuth provider: %w", err)
    }
    if err == repository.ErrNotFound {
		// Create OAuth Provider
        oauthProvider = &model.OAuthProvider{
            UserID:       user.ID,
            Provider:     "google",
            ProviderID:   googleUser.Sub,
            AccessToken:  token.AccessToken,
            RefreshToken: token.RefreshToken,
            ExpiresAt:    token.Expiry,

        }
        if err := s.Repository.CreateOAuthProvider(ctx, oauthProvider); err != nil {
            return "", "", fmt.Errorf("failed to create OAuth provider: %w", err)
        }
	}else{
		// Update OAuth Provider
		oauthProvider.AccessToken = token.AccessToken
        oauthProvider.RefreshToken = token.RefreshToken
        oauthProvider.ExpiresAt = token.Expiry

        if err := s.Repository.UpdateOAuthProvider(ctx, oauthProvider); err != nil {
            return "", "", fmt.Errorf("failed to update OAuth provider: %w", err)
        }
	}

    // Generate JWT token
    jwtToken, err := GenerateJWT(user, "google", s.Config) // Pass "google" as authProvider
    if err != nil {
        return "", "", fmt.Errorf("failed to generate JWT: %w", err)
    }

    // Generate refresh token
    refreshToken, err := GenerateRefreshToken(ctx, user.ID, s.Repository, s.Config)
    if err != nil {
        return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
    }

    return jwtToken, refreshToken, nil
}

// GetGmailService creates a new Gmail service using the provided OAuth token.
func (s *GoogleAuthService) GetGmailService(ctx context.Context, userID string) (*http.Client, error) {
	tokenJSON, err := s.Repository.GetGmailToken(ctx, userID)
	if err != nil {
		return nil, err
	}

    var token oauth2.Token
    if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
        return nil, fmt.Errorf("failed to unmarshal token JSON: %w", err)
    }

	return s.oauthConf.Client(ctx, &token), nil // Create an *http.Client
}