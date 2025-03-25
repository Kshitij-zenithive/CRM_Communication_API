// internal/service/email.go (Corrected, added missing functions and dependencies)
package service

import (
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/util" // Corrected import
	"encoding/base64"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type EmailService struct {
	repo         *repository.Repository
	GmailService func(ctx context.Context, userID string) (*http.Client, error) // Inject the Gmail service

}

func NewEmailService(repo *repository.Repository, gmailServiceFunc func(ctx context.Context, userID string) (*http.Client, error)) *EmailService {
	return &EmailService{
		repo:         repo,
		GmailService: gmailServiceFunc,
	}
}

// SendEmail sends an email using the provided input.
func (s *EmailService) SendEmail(ctx context.Context, email *model.Email) (*model.Email, error) {

	// Retrieve the user's Gmail service
	gmailClient, err := s.GmailService(ctx, email.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(gmailClient))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	// Construct the email message
	var message gmail.Message

	// Build the MIME message
	rawMessage := fmt.Sprintf("From: %s\r\n", email.From)
	rawMessage += fmt.Sprintf("To: %s\r\n", email.To)
	rawMessage += fmt.Sprintf("Subject: %s\r\n", email.Subject)
	rawMessage += "MIME-Version: 1.0\r\n"
	rawMessage += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	rawMessage += "Content-Transfer-Encoding: base64\r\n\r\n"
	rawMessage += email.Body

	// Encode the message
	message.Raw = base64.URLEncoding.EncodeToString([]byte(rawMessage))

	// Send the email
	_, err = srv.Users.Messages.Send("me", &message).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}
	// Create the email in the database first
	email.CreatedAt = time.Now()
	if err := s.repo.CreateEmail(ctx, email); err != nil {
		return nil, fmt.Errorf("email created on gmail, but failed to persist to database: %w", err)
	}
	return email, nil
}

// GetEmails retrieves emails for a client.
func (s *EmailService) GetEmails(ctx context.Context, clientID uuid.UUID) ([]model.Email, error) {
	return s.repo.GetEmailsByClientID(ctx, clientID)
}

// GetEmail retrieves a specific email by ID.
func (s *EmailService) GetEmail(ctx context.Context, emailID uuid.UUID) (*model.Email, error) {
	return s.repo.GetEmailByID(ctx, emailID)
}

// Utility function to find client by email, used both in sending and syncing.
func (s *EmailService) findClientByEmail(ctx context.Context, emails []string) (*model.Client, error) {
	for _, email := range emails {
		// Trim whitespace and parse the email address
		parsedEmail, err := mail.ParseAddress(strings.TrimSpace(email))
		if err != nil {
			continue // Skip invalid email addresses
		}

		// Attempt to find a client with this email
		client, err := s.repo.GetClientByEmail(ctx, parsedEmail.Address)
		if err != nil && err != repository.ErrNotFound {
			return nil, fmt.Errorf("error finding client by email: %w", err)
		}
		if client != nil {
			return client, nil // Found a client, return immediately
		}
	}

	return nil, repository.ErrNotFound // No client found for any of the emails
}

// SyncGmail synchronizes emails from Gmail.
func (s *EmailService) SyncGmail(ctx context.Context, userID string) error {

	// Retrieve the Gmail OAuth token from the database
	gmailClient, err := s.GmailService(ctx, userID)
	if err != nil {
		return fmt.Errorf("unable to create Gmail service: %w", err)
	}

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(gmailClient))
	if err != nil {
		return fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	// List messages from the Gmail API.  You could add query parameters for filtering.
	listCall := srv.Users.Messages.List("me").Q("in:inbox") // Example query
	listResp, err := listCall.Do()
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("failed to parse UserID: %w", err)
	}
	// Process the messages
	for _, m := range listResp.Messages {
		msg, err := srv.Users.Messages.Get("me", m.Id).Format("full").Do()
		if err != nil {
			//log.Printf("Unable to retrieve message %s: %v", m.Id, err)
			continue // Skip to the next message on error
		}

		// Check if the email already exists in our database
		existingEmail, _ := s.repo.GetEmailByID(ctx, uuid.MustParse(msg.Id)) // Assuming email ID maps to message ID
		if existingEmail != nil {
			continue // Skip if the email already exists
		}

		// Extract relevant data from the Gmail message
		email := model.Email{
			ID:       uuid.MustParse(msg.Id),
			GoogleID: msg.Id,
			ThreadID: msg.ThreadId,
			UserID:   userUUID, // Set the user ID to the logged-in user
			// ClientID: ... // You'll need to figure out the client ID, possibly by matching the sender/recipient email
		}

		// Parse headers
		for _, h := range msg.Payload.Headers {
			switch strings.ToLower(h.Name) {
			case "subject":
				email.Subject = h.Value
			case "from":
				parsedAddress, err := mail.ParseAddress(h.Value)
				if err != nil {
					//log.Printf("Failed to parse 'From' address: %v", err)
					email.From = h.Value // Use raw
				} else {
					email.From = parsedAddress.Address
				}

			case "to":
				parsedAddress, err := mail.ParseAddress(h.Value)
				if err != nil {
					//log.Printf("Failed to parse 'To' address: %v", err)
					email.To = h.Value
				} else {
					email.To = parsedAddress.Address
				}

			case "date":
				parsedTime, err := util.ParseDate(h.Value)
				if err == nil {
					email.Received = parsedTime
				}
			}
		}

		// Extract the body (handle both plain text and HTML)
		if msg.Payload.Body.Size > 0 {
			email.Body, err = util.DecodeBase64String(msg.Payload.Body.Data)
		} else {
			for _, part := range msg.Payload.Parts {
				switch part.MimeType {
				case "text/plain":
					email.Snippet, err = util.DecodeBase64String(part.Body.Data) // Snippet use plain text
				case "text/html":
					email.Body, err = util.DecodeBase64String(part.Body.Data)
				}
			}
		}
		// Set the snippet (use the Go snippet if the Gmail snippet is empty)
		if msg.Snippet != "" {
			email.Snippet = msg.Snippet
		}

		//Try to parse the email and assign to client
		client, err := s.findClientByEmail(ctx, []string{email.From, email.To})
		if err != nil {
			//Decide to proceed or return
		}
		if client != nil {
			email.ClientID = client.ID
		}

		// Save the email to the database
		if err := s.repo.CreateEmail(ctx, &email); err != nil {
			//log.Printf("Failed to save email: %v", err)
			continue // Skip to the next message
		}
	}

	return nil
}
