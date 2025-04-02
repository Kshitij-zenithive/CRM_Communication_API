// internal/service/email_service.go
package service

import (
	"context"
	"crm-communication-api/internal/auth"          // Correctly import auth package
	dbmodel "crm-communication-api/internal/model" // Use alias for clarity
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/util"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	// "net/http" // Import needed for http.Client type from authService.GetGmailService
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq" // For StringArray
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// emailService implements the EmailService interface.
type emailService struct {
	emailRepo       repository.EmailRepository
	clientRepo      repository.ClientRepository // Needed to find clients by email
	authService     auth.Service
	userRepo        repository.AuthRepository // Needed for user lookups
	templateService TemplateService
	timelineService TimelineService
	logger          *slog.Logger
}

// Assert that emailService implements EmailService interface
var _ EmailService = (*emailService)(nil)

// NewEmailService creates a new instance of emailService.
func NewEmailService(
	emailRepo repository.EmailRepository,
	clientRepo repository.ClientRepository,
	authService auth.Service,
	userRepo repository.AuthRepository,
	templateService TemplateService,
	timelineService TimelineService,
	logger *slog.Logger,
) EmailService { // Return the interface type
	return &emailService{
		emailRepo:       emailRepo,
		clientRepo:      clientRepo,
		authService:     authService, // Assign the concrete pointer
		userRepo:        userRepo,
		templateService: templateService,
		timelineService: timelineService,
		logger:          logger.With(slog.String("service", "EmailService")),
	}
}

// --- Email Sending ---

// SendEmail constructs, sends via Gmail, saves, and logs a timeline event.
func (s *emailService) SendEmail(ctx context.Context, email *dbmodel.Email) (*dbmodel.Email, error) {
	l := s.logger.With(slog.String("method", "SendEmail"), slog.String("userID", email.UserID.String()))

	// 1. Validate Input
	if email.UserID == uuid.Nil {
		return nil, errors.New("sender user ID is required")
	}
	// Ensure To field is not nil/empty before joining
	if len(email.To) == 0 {
		return nil, errors.New("recipient(s) are required")
	}
	if email.Subject == "" || (email.BodyHTML == "" && email.BodyText == "") {
		return nil, errors.New("subject and body are required")
	}

	// 2. Ensure 'From' is set (use authenticated user's email if not provided)
	if email.From == "" {
		// Call GetCurrentUser on the concrete AuthService
		// Assuming GetCurrentUser is correctly implemented on AuthService
		currentUser, err := s.authService.GetCurrentUser(ctx)
		if err != nil {
			l.ErrorContext(ctx, "Could not determine sender 'From' address", slog.String("error", err.Error()))
			return nil, fmt.Errorf("could not determine sender email: %w", err)
		}
		email.From = currentUser.Email
	}

	// 3. Get Authenticated Gmail Client
	// Call GetGmailService on the concrete AuthService
	gmailHTTPClient, err := s.authService.GetGmailService(ctx, email.UserID.String())
	if err != nil {
		l.ErrorContext(ctx, "Failed to get Gmail service client", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get Gmail client: %w", err)
	}
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(gmailHTTPClient))
	if err != nil {
		l.ErrorContext(ctx, "Failed to create Gmail service object", slog.String("error", err.Error()))
		return nil, fmt.Errorf("unable to retrieve Gmail client: %w", err)
	}

	// 4. Construct MIME Message
	var message gmail.Message
	rawMessage := s.buildMIMEMessage(email) // Use helper
	message.Raw = base64.URLEncoding.EncodeToString([]byte(rawMessage))

	// 5. Send via Gmail API
	sentMsg, err := gmailService.Users.Messages.Send("me", &message).Do()
	if err != nil {
		l.ErrorContext(ctx, "Gmail API failed to send email", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to send email via Gmail: %w", err)
	}
	l.InfoContext(ctx, "Email sent successfully via Gmail", slog.String("gmailMessageID", sentMsg.Id))

	// 6. Prepare DB Model (Fill in details)
	now := time.Now()
	email.Provider = "gmail"
	email.ProviderID = sentMsg.Id
	email.ThreadID = sentMsg.ThreadId
	email.Direction = dbmodel.EmailDirectionOutbound
	email.SentAt = &now // Set sent time
	email.IsRead = true // Sender has obviously "read" it

	// 7. Attempt to associate with a Client (if ClientID is not already set)
	if email.ClientID == nil || *email.ClientID == uuid.Nil {
		// Use the first recipient to find the client for now
		if len(email.To) > 0 {
			client, err := s.findClientByEmailAddressList(ctx, email.To) // Pass the slice directly
			if err == nil && client != nil {
				email.ClientID = &client.ID
				l.DebugContext(ctx, "Associated outbound email with client", slog.String("clientID", client.ID.String()))
			} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
				l.WarnContext(ctx, "Error trying to find client for outbound email", slog.String("recipients", strings.Join(email.To, ",")), slog.String("error", err.Error()))
				// Continue without client association
			}
		}
	}

	// 8. Save to Repository
	if err := s.emailRepo.CreateEmail(ctx, email); err != nil {
		l.ErrorContext(ctx, "Failed to save sent email to database", slog.String("gmailMessageID", sentMsg.Id), slog.String("error", err.Error()))
		// Return the email object anyway, as it was sent, but log the DB error critically
		return email, fmt.Errorf("email sent via Gmail but failed to save to DB: %w", err)
	}
	l.InfoContext(ctx, "Sent email saved to database", slog.String("emailID", email.ID.String()))

	// 9. Create Timeline Event
	timelineEvent := &dbmodel.TimelineEvent{
		UserID:      email.UserID,
		ClientID:    email.ClientID,
		EventType:   string(dbmodel.InteractionTypeEmailSent), // Use constant
		Description: fmt.Sprintf("Email sent to %s: %s", strings.Join(email.To, ", "), email.Subject),
		RelatedID:   &email.ID,
		RelatedType: util.StringPtr("email"), // Use util helper
		EventTime:   *email.SentAt,
	}
	if _, err := s.timelineService.CreateTimelineEvent(ctx, timelineEvent); err != nil {
		l.WarnContext(ctx, "Failed to create timeline event for sent email", slog.String("emailID", email.ID.String()), slog.String("error", err.Error()))
		// Non-fatal
	}

	return email, nil
}

// SendEmailWithTemplate renders a template and sends the email.
func (s *emailService) SendEmailWithTemplate(ctx context.Context, senderID, clientID, templateID uuid.UUID, recipientEmails []string, variables map[string]interface{}) (*dbmodel.Email, error) {
	l := s.logger.With(slog.String("method", "SendEmailWithTemplate"), slog.String("senderID", senderID.String()), slog.String("templateID", templateID.String()))

	// 1. Get Template
	template, err := s.templateService.GetEmailTemplate(ctx, templateID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get email template", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to retrieve email template: %w", err)
	}

	// 2. Add default/global variables (enhanced)
	if variables == nil {
		variables = make(map[string]interface{})
	}
	sender, err := s.userRepo.GetUserByID(ctx, senderID)
	if err == nil {
		variables["SenderName"] = sender.Name
		variables["SenderEmail"] = sender.Email
	} else {
		l.WarnContext(ctx, "Could not fetch sender for template variables", slog.String("error", err.Error()))
	}
	client, err := s.clientRepo.GetClientByID(ctx, clientID) // Use clientRepo
	if err == nil {
		variables["ClientName"] = client.Name
		variables["ClientEmail"] = client.Email
		if client.Company != nil {
			variables["ClientCompany"] = *client.Company
		}
	} else {
		l.WarnContext(ctx, "Could not fetch client for template variables", slog.String("error", err.Error()))
	}

	// 3. Render Template
	renderedSubject, renderedBody, err := s.templateService.RenderTemplate(template, variables)
	if err != nil {
		l.ErrorContext(ctx, "Failed to render email template", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// 4. Prepare Email Model
	fromEmail := ""
	if sender != nil {
		fromEmail = sender.Email
	}

	email := &dbmodel.Email{
		ClientID:  &clientID,
		UserID:    senderID,
		Subject:   renderedSubject,
		From:      fromEmail, // Let SendEmail fill if empty and user exists
		To:        pq.StringArray(recipientEmails),
		Cc:        nil,
		Bcc:       nil,
		BodyHTML:  renderedBody, // Assuming template produces HTML
		Direction: dbmodel.EmailDirectionOutbound,
	}

	// 5. Call SendEmail
	return s.SendEmail(ctx, email)
}

// --- Email Syncing ---

// SyncGmail fetches emails for a user and stores them.
func (s *emailService) SyncGmail(ctx context.Context, userID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID for sync: %w", err)
	}
	l := s.logger.With(slog.String("method", "SyncGmail"), slog.String("userID", userID))

	// 1. Get Gmail Client
	gmailHTTPClient, err := s.authService.GetGmailService(ctx, userID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get Gmail service client for sync", slog.String("error", err.Error()))
		return fmt.Errorf("failed to get Gmail client for sync: %w", err)
	}
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(gmailHTTPClient))
	if err != nil {
		l.ErrorContext(ctx, "Failed to create Gmail service object for sync", slog.String("error", err.Error()))
		return fmt.Errorf("unable to retrieve Gmail client for sync: %w", err)
	}

	// 2. List Messages - Use pagination!
	l.InfoContext(ctx, "Listing messages from Gmail...")
	var syncedCount int
	// Example: Get recent unread messages in inbox, limit to 50 per page initially
	listCall := gmailService.Users.Messages.List("me").Q("in:inbox is:unread").MaxResults(50)
	err = listCall.Pages(ctx, func(listResp *gmail.ListMessagesResponse) error {
		l.DebugContext(ctx, "Processing page of messages", slog.Int("count", len(listResp.Messages)))
		for _, m := range listResp.Messages {
			select {
			case <-ctx.Done():
				return ctx.Err() // Respect context cancellation
			default:
			}

			// 3. Check if already synced
			_, errDb := s.emailRepo.GetEmailByProviderID(ctx, "gmail", m.Id)
			if errDb == nil {
				l.DebugContext(ctx, "Skipping already synced message", slog.String("gmailMessageID", m.Id))
				continue
			}
			if !errors.Is(errDb, repository.ErrNotFound) {
				l.ErrorContext(ctx, "Error checking if message exists in DB", slog.String("gmailMessageID", m.Id), slog.String("error", errDb.Error()))
				continue
			}

			// 4. Get full message details
			msg, errGet := gmailService.Users.Messages.Get("me", m.Id).Format("full").Do()
			if errGet != nil {
				l.WarnContext(ctx, "Failed to get full message details from Gmail", slog.String("gmailMessageID", m.Id), slog.String("error", errGet.Error()))
				continue
			}

			// 5. Parse and save
			if errParse := s.parseAndSaveGmailMessage(ctx, userUUID, msg); errParse != nil {
				l.ErrorContext(ctx, "Failed to parse and save message", slog.String("gmailMessageID", m.Id), slog.String("error", errParse.Error()))
				// Continue to next message
			} else {
				syncedCount++
				// Optional: Mark as read in Gmail after successful sync
				// modifyReq := &gmail.ModifyMessageRequest{ RemoveLabelIds: []string{"UNREAD"} }
				// _, errMod := gmailService.Users.Messages.Modify("me", msg.Id, modifyReq).Do()
				// if errMod != nil { l.WarnContext(ctx, "Failed to mark message as read in Gmail", ...) }
			}
		}
		return nil // Continue pagination
	})

	if err != nil {
		// Log error during pagination itself (e.g., token expired mid-sync)
		l.ErrorContext(ctx, "Failed during Gmail message listing pagination", slog.String("error", err.Error()))
		return fmt.Errorf("error listing gmail messages: %w", err)
	}

	l.InfoContext(ctx, "Gmail sync completed", slog.Int("syncedCount", syncedCount))
	return nil
}

// parseAndSaveGmailMessage processes a single Gmail message.
func (s *emailService) parseAndSaveGmailMessage(ctx context.Context, internalUserID uuid.UUID, msg *gmail.Message) error {
	l := s.logger.With(slog.String("method", "parseAndSaveGmailMessage"), slog.String("gmailMessageID", msg.Id))

	var subject, from, bodyHTML, bodyText, snippet string
	var to, cc, bcc []string
	var receivedAt, sentAt time.Time
	var isUnread bool

	// Parse Headers
	for _, h := range msg.Payload.Headers {
		switch strings.ToLower(h.Name) {
		case "subject":
			subject = h.Value
		case "from":
			addr, err := mail.ParseAddress(h.Value)
			if err == nil {
				from = addr.Address
			} else {
				from = h.Value
			}
		case "to":
			to = parseAddressList(h.Value)
		case "cc":
			cc = parseAddressList(h.Value)
		case "bcc":
			bcc = parseAddressList(h.Value)
		case "date":
			t, err := util.ParseDate(h.Value)
			if err == nil {
				receivedAt = t
				sentAt = t
			}
		}
	}
	if msg.InternalDate > 0 { // Use internal date if more precise
		internalTime := time.UnixMilli(msg.InternalDate)
		if !internalTime.IsZero() {
			receivedAt = internalTime
			sentAt = internalTime
		}
	}

	// Check Labels for Read Status
	for _, labelId := range msg.LabelIds {
		if labelId == "UNREAD" {
			isUnread = true
			break
		}
	}

	// Extract Body and Snippet
	bodyHTML, bodyText = extractBodyParts(msg.Payload)
	if msg.Snippet != "" {
		snippet = msg.Snippet
	}

	// Find associated client (check sender first, then recipients)
	var clientID *uuid.UUID
	client, err := s.findClientByEmailAddressList(ctx, append([]string{from}, append(to, cc...)...))
	if err == nil && client != nil {
		clientID = &client.ID
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		l.WarnContext(ctx, "Error finding client for incoming email", slog.String("error", err.Error()))
	}

	// Prepare DB model
	email := &dbmodel.Email{
		// ID:         uuid.MustParse(msg.Id), // Use Gmail ID ONLY if you are SURE it's a valid UUID V4 and want it as PK. Safer to let DB generate internal ID.
		ClientID:   clientID,
		UserID:     internalUserID,
		Provider:   "gmail",
		ProviderID: msg.Id, // Store Gmail's ID
		ThreadID:   msg.ThreadId,
		Subject:    subject,
		From:       from,
		To:         pq.StringArray(to),
		Cc:         pq.StringArray(cc),
		Bcc:        pq.StringArray(bcc),
		BodyHTML:   bodyHTML,
		BodyText:   bodyText,
		Snippet:    snippet,
		SentAt:     &sentAt,
		ReceivedAt: &receivedAt,
		IsRead:     !isUnread,
		Direction:  dbmodel.EmailDirectionInbound,
	}

	// Save to DB (CreateEmail handles CreatedAt/UpdatedAt)
	if err := s.emailRepo.CreateEmail(ctx, email); err != nil {
		return fmt.Errorf("failed to save synced email to database: %w", err)
	}
	l.InfoContext(ctx, "Successfully synced and saved email", slog.String("internalEmailID", email.ID.String()))

	// Trigger Timeline Event
	timelineEvent := &dbmodel.TimelineEvent{
		UserID:      internalUserID,
		ClientID:    clientID,
		EventType:   string(dbmodel.InteractionTypeEmailReceived),
		Description: fmt.Sprintf("Email received from %s: %s", from, subject),
		RelatedID:   &email.ID, // Link to our internal email ID
		RelatedType: util.StringPtr("email"),
		EventTime:   receivedAt,
	}
	if _, err := s.timelineService.CreateTimelineEvent(ctx, timelineEvent); err != nil {
		l.WarnContext(ctx, "Failed to create timeline event for received email", slog.String("emailID", email.ID.String()), slog.String("error", err.Error()))
	}

	return nil
}

// --- Email Retrieval ---

func (s *emailService) GetEmails(ctx context.Context, clientID uuid.UUID) ([]dbmodel.Email, error) {
	l := s.logger.With(slog.String("method", "GetEmails"), slog.String("clientID", clientID.String()))
	emails, err := s.emailRepo.GetEmailsByClientID(ctx, clientID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to get emails by client ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error getting emails: %w", err)
	}
	return emails, nil
}

func (s *emailService) GetEmail(ctx context.Context, emailID uuid.UUID) (*dbmodel.Email, error) {
	l := s.logger.With(slog.String("method", "GetEmail"), slog.String("emailID", emailID.String()))
	email, err := s.emailRepo.GetEmailByID(ctx, emailID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.WarnContext(ctx, "Email not found")
			return nil, repository.ErrNotFound
		}
		l.ErrorContext(ctx, "Failed to get email by ID", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error getting email: %w", err)
	}
	return email, nil
}

// --- Helper Functions ---

// buildMIMEMessage constructs the raw email string for sending via API.
func (s *emailService) buildMIMEMessage(email *dbmodel.Email) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ",")))
	if len(email.Cc) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.Cc, ",")))
	}
	// Note: BCC is handled by the API's 'To', 'Cc', 'Bcc' parameters, not raw headers.
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	builder.WriteString("MIME-Version: 1.0\r\n")

	body := ""
	contentType := "text/plain"
	if email.BodyHTML != "" {
		contentType = "text/html"
		body = email.BodyHTML // Body will be base64 encoded later
	} else {
		body = email.BodyText // Fallback to plain text
	}

	builder.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n", contentType))
	builder.WriteString("Content-Transfer-Encoding: base64\r\n")
	builder.WriteString("\r\n")                                          // Header/Body separator
	builder.WriteString(base64.URLEncoding.EncodeToString([]byte(body))) // Encode the chosen body

	return builder.String()
}

// findClientByEmailAddress searches for a client matching a single email address.
func (s *emailService) findClientByEmailAddress(ctx context.Context, emailStr string) (*dbmodel.Client, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(emailStr))
	if err != nil {
		return nil, fmt.Errorf("invalid email format '%s': %w", emailStr, err)
	}
	client, err := s.clientRepo.GetClientByEmail(ctx, addr.Address) // Use normalized address
	if err != nil {
		// Don't wrap ErrNotFound here, let caller check
		return nil, err
	}
	return client, nil
}

// findClientByEmailAddressList searches for a client matching any email in the list.
func (s *emailService) findClientByEmailAddressList(ctx context.Context, emails []string) (*dbmodel.Client, error) {
	if len(emails) == 0 {
		return nil, repository.ErrNotFound // Or nil, nil depending on desired behavior
	}
	checkedEmails := make(map[string]bool) // Avoid redundant checks for same address

	for _, emailStr := range emails {
		addr, err := mail.ParseAddress(strings.TrimSpace(emailStr))
		if err != nil || addr.Address == "" || checkedEmails[addr.Address] {
			continue // Skip invalid or already checked addresses
		}
		checkedEmails[addr.Address] = true

		client, err := s.clientRepo.GetClientByEmail(ctx, addr.Address)
		if err == nil && client != nil {
			return client, nil // Found one
		}
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			// Log unexpected errors but continue searching other emails
			s.logger.WarnContext(ctx, "Error during findClientByEmailAddressList lookup", slog.String("email", addr.Address), slog.String("error", err.Error()))
		}
	}
	return nil, repository.ErrNotFound // Not found after checking all unique, valid emails
}

// parseAddressList parses a comma-separated list of email addresses.
func parseAddressList(list string) []string {
	if list == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(list)
	if err != nil {
		// Basic fallback: split by comma and trim whitespace
		parts := strings.Split(list, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			// Very basic email format check (presence of '@')
			if trimmed != "" && strings.Contains(trimmed, "@") {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
		return nil // Return nil if parsing fails and fallback yields nothing
	}
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = addr.Address // Extract the actual address part
	}
	return result
}

// extractBodyParts iterates through message parts to find HTML and Text bodies.
// Prioritizes HTML if available.
func extractBodyParts(part *gmail.MessagePart) (bodyHTML, bodyText string) {
	if part == nil {
		return "", ""
	}

	// Check current part's body
	currentHTML, currentText := "", ""
	if part.Body != nil && part.Body.Size > 0 {
		decodedBody, err := util.DecodeBase64String(part.Body.Data)
		if err == nil {
			if strings.HasPrefix(part.MimeType, "text/html") {
				currentHTML = decodedBody
			} else if strings.HasPrefix(part.MimeType, "text/plain") {
				currentText = decodedBody
			}
		}
	}

	// If this part *is* multipart/alternative, find the best content among children
	if strings.Contains(part.MimeType, "multipart/alternative") {
		for _, subPart := range part.Parts {
			html, text := extractBodyParts(subPart)
			if html != "" {
				bodyHTML = html
			} // Prefer HTML from subparts
			if text != "" {
				bodyText = text
			}
		}
		// If HTML found within alternative, prioritize it
		if bodyHTML != "" {
			return bodyHTML, bodyText
		}
		// If only text found within alternative, use it
		if bodyText != "" {
			return "", bodyText
		} // Return empty HTML, non-empty Text
	}

	// If this part is multipart/related or multipart/mixed, check children
	if strings.HasPrefix(part.MimeType, "multipart/") && !strings.Contains(part.MimeType, "multipart/alternative") {
		for _, subPart := range part.Parts {
			html, text := extractBodyParts(subPart)
			// Accumulate or prioritize? Let's prioritize the first non-empty found.
			if bodyHTML == "" && html != "" {
				bodyHTML = html
			}
			if bodyText == "" && text != "" {
				bodyText = text
			}
		}
	}

	// If the current part had content, use it if we didn't find better in children
	if bodyHTML == "" {
		bodyHTML = currentHTML
	}
	if bodyText == "" {
		bodyText = currentText
	}

	// Final check: Prefer HTML if both are somehow populated at this level
	if bodyHTML != "" {
		return bodyHTML, bodyText
	}
	return "", bodyText // Default to text if HTML is empty
}
