// internal/repository/repository.go (Corrected Preloads and error handling)

package repository

import (
	"context"
	"crm-communication-api/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Add ErrNotFound, for not founds resources
var ErrNotFound = errors.New("resource not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// User related repository methods
func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("OAuthProviders").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).Preload("OAuthProviders").First(&user).Error //Preload OAuthProviders
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, err
}
func (r *Repository) UpdateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Client related methods
func (r *Repository) CreateClient(ctx context.Context, client *model.Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

func (r *Repository) GetClientByID(ctx context.Context, id uuid.UUID) (*model.Client, error) {
	var client model.Client
	err := r.db.WithContext(ctx).First(&client, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &client, err
}
func (r *Repository) GetClientByEmail(ctx context.Context, email string) (*model.Client, error) {
	var client model.Client
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &client, err
}

// Message related methods
func (r *Repository) CreateMessage(ctx context.Context, message *model.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetMessageByID retrieves a message by its ID.
func (r *Repository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*model.Message, error) {
	var message model.Message
	err := r.db.WithContext(ctx).
		Preload("Conversation"). // Important for later use in the service
		First(&message, messageID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &message, nil
}

// GetMessagesByConversationID retrieves messages for a specific conversation.
func (r *Repository) GetMessagesByConversationID(ctx context.Context, conversationID uuid.UUID) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Conversation").
		Preload("Mentions").
		Preload("Mentions.User").
		Preload("ReadBy").      // Preload read receipts
		Preload("ReadBy.User"). // Preload user details for read receipts
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}

// CreateMessageMention creates a new message mention.
func (r *Repository) CreateMessageMention(ctx context.Context, mention *model.MessageMention) error {
	return r.db.WithContext(ctx).Create(mention).Error
}

// CreateMessageReadReceipt creates a new message read receipt.
func (r *Repository) CreateMessageReadReceipt(ctx context.Context, receipt *model.MessageReadReceipt) error {
	return r.db.WithContext(ctx).Create(receipt).Error
}

// GetMessageReadReceipt checks if a read receipt exists for a message and user.
func (r *Repository) GetMessageReadReceipt(ctx context.Context, messageID, userID uuid.UUID) (*model.MessageReadReceipt, error) {
	var receipt model.MessageReadReceipt
	err := r.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ?", messageID, userID).
		First(&receipt).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound // Return custom error
		}
		return nil, err // Return other errors
	}
	return &receipt, nil
}

// Email related repository methods
func (r *Repository) CreateEmail(ctx context.Context, email *model.Email) error {
	return r.db.WithContext(ctx).Create(email).Error
}
func (r *Repository) GetEmailByID(ctx context.Context, id uuid.UUID) (*model.Email, error) {
	var email model.Email
	err := r.db.WithContext(ctx).Preload("Client").Preload("User").First(&email, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &email, err
}

func (r *Repository) GetEmailsByClientID(ctx context.Context, clientID uuid.UUID) ([]model.Email, error) {
	var emails []model.Email
	err := r.db.WithContext(ctx).
		Preload("Client").
		Preload("User").
		Where("client_id = ?", clientID).
		Order("created_at DESC").
		Find(&emails).Error
	return emails, err
}

// Timeline related repository methods
func (r *Repository) CreateTimelineEvent(ctx context.Context, event *model.TimelineEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// Get Timeline Events with Polymorphic Association
func (r *Repository) GetTimelineEventsByClientID(ctx context.Context, clientID uuid.UUID) ([]model.TimelineEvent, error) {
	var events []model.TimelineEvent
	err := r.db.WithContext(ctx).
		Preload("Client").
		Preload("User").
		Preload("Eventable"). // Preload the polymorphic association
		Where("client_id = ?", clientID).
		Order("event_time DESC").
		Find(&events).Error
	return events, err
}

// OAuthProvider related methods
func (r *Repository) CreateOAuthProvider(ctx context.Context, provider *model.OAuthProvider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}
func (r *Repository) GetOAuthProvider(ctx context.Context, userID uuid.UUID, providerName string) (*model.OAuthProvider, error) {
	var provider model.OAuthProvider
	err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, providerName).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &provider, err
}

func (r *Repository) UpdateOAuthProvider(ctx context.Context, provider *model.OAuthProvider) error {
	return r.db.WithContext(ctx).Save(provider).Error
}

// RefreshToken related methods
func (r *Repository) CreateRefreshToken(ctx context.Context, refreshToken *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(refreshToken).Error
}

func (r *Repository) GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	var refreshToken model.RefreshToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&refreshToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &refreshToken, err
}

func (r *Repository) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).Where("token = ?", token).Delete(&model.RefreshToken{}).Error
}
func (r *Repository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&model.RefreshToken{}).Error
}

// CreateEmailTemplate creates a new email template.
func (r *Repository) CreateEmailTemplate(ctx context.Context, template *model.EmailTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}

// GetEmailTemplate retrieves an email template by ID.
func (r *Repository) GetEmailTemplate(ctx context.Context, id uuid.UUID) (*model.EmailTemplate, error) {
	var template model.EmailTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &template, err
}

// UpdateEmailTemplate updates an existing email template.
func (r *Repository) UpdateEmailTemplate(ctx context.Context, template *model.EmailTemplate) error {
	return r.db.WithContext(ctx).Save(template).Error
}

// DeleteEmailTemplate deletes an email template by ID.
func (r *Repository) DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.EmailTemplate{}, id).Error
}

// GetAllEmailTemplates retrieves all email templates.
func (r *Repository) GetAllEmailTemplates(ctx context.Context) ([]model.EmailTemplate, error) {
	var templates []model.EmailTemplate
	err := r.db.WithContext(ctx).Find(&templates).Error
	return templates, err
}

// SaveGmailToken saves or updates the Gmail OAuth token for a user.
func (r *Repository) SaveGmailToken(ctx context.Context, userID string, tokenJSON string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	var oauthProvider model.OAuthProvider
	result := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userUUID, "google").First(&oauthProvider)

	var token map[string]interface{}
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return fmt.Errorf("failed to unmarshal token JSON: %w", err)
	}

	// Extract token details
	accessToken, _ := token["access_token"].(string)
	refreshToken, _ := token["refresh_token"].(string) // OK if this is missing
	expiry, _ := token["expiry"].(string)

	// Convert expiry to time.Time
	expiryTime := time.Now().Add(time.Hour) // Default in case of parsing issues
	if expiry != "" {
		if parsedTime, err := time.Parse(time.RFC3339, expiry); err == nil {
			expiryTime = parsedTime
		}
	}

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create a new OAuthProvider record
			oauthProvider = model.OAuthProvider{
				UserID:       userUUID,
				Provider:     "google",
				AccessToken:  accessToken,
				RefreshToken: refreshToken, // Save the refresh token
				ExpiresAt:    expiryTime,
			}
			if err := r.db.WithContext(ctx).Create(&oauthProvider).Error; err != nil {
				return fmt.Errorf("failed to create OAuth provider: %w", err)
			}
		} else {
			return fmt.Errorf("failed to query OAuth provider: %w", result.Error)
		}
	} else {
		// Update existing OAuthProvider record
		oauthProvider.AccessToken = accessToken
		if refreshToken != "" { // Only update if a new refresh token is provided
			oauthProvider.RefreshToken = refreshToken
		}

		oauthProvider.ExpiresAt = expiryTime
		if err := r.db.WithContext(ctx).Save(&oauthProvider).Error; err != nil {
			return fmt.Errorf("failed to update OAuth provider: %w", err)
		}
	}

	return nil
}

// GetGmailToken retrieves the Gmail OAuth token for a user.
func (r *Repository) GetGmailToken(ctx context.Context, userID string) (string, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	var oauthProvider model.OAuthProvider
	result := r.db.WithContext(ctx).Preload(clause.Associations).Where("user_id = ? AND provider = ?", userUUID, "google").First(&oauthProvider)
	if result.Error != nil {
		return "", fmt.Errorf("failed to get OAuth provider: %w", result.Error)
	}

	// Marshal the token details back into JSON
	token := map[string]interface{}{
		"access_token":  oauthProvider.AccessToken,
		"refresh_token": oauthProvider.RefreshToken,
		"expiry":        oauthProvider.ExpiresAt.Format(time.RFC3339),
		"token_type":    "Bearer", // Add token type
	}

	tokenJSON, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	return string(tokenJSON), nil
}

// CreateConversation creates a new chat conversation.
func (r *Repository) CreateConversation(ctx context.Context, conversation *model.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

// CreateConversationParticipant adds a user to a conversation.
func (r *Repository) CreateConversationParticipant(ctx context.Context, participant *model.ConversationParticipant) error {
	return r.db.WithContext(ctx).Create(participant).Error
}

// GetConversationsByUser retrieves all conversations a user is part of.
func (r *Repository) GetConversationsByUser(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	var conversations []model.Conversation
	err := r.db.WithContext(ctx).
		Preload("Participants").
		Preload("Participants.User").
		Preload("Messages"). // Preload for eager loading of last messages.
		Preload("Client").
		Joins("JOIN conversation_participants ON conversation_participants.conversation_id = conversations.id").
		Where("conversation_participants.user_id = ?", userID).
		Group("conversations.id"). // Avoid duplicates due to multiple participants
		Find(&conversations).Error
	return conversations, err
}

// GetConversationByID retrieves a specific conversation by ID
func (r *Repository) GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*model.Conversation, error) {
	var conversation model.Conversation
	err := r.db.WithContext(ctx).
		Preload("Participants").
		Preload("Participants.User").
		Preload("Messages").
		Preload("Client").
		First(&conversation, conversationID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &conversation, nil
}

// -- Task related --
// CreateTask creates a new task.
func (r *Repository) CreateTask(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetTaskByID retrieves a task by its ID.
func (r *Repository) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).
		Preload("AssignedUser").
		Preload("Creator").
		Preload("Client").
		First(&task, taskID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &task, nil
}

// UpdateTask updates an existing task.
func (r *Repository) UpdateTask(ctx context.Context, task *model.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// DeleteTask deletes a task by its ID.
func (r *Repository) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Task{}, taskID).Error
}

// GetTasksByUserID retrieves all tasks assigned to or created by a user.
func (r *Repository) GetTasksByUserID(ctx context.Context, userID uuid.UUID) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.WithContext(ctx).
		Preload("AssignedUser").
		Preload("Creator").
		Preload("Client").
		Where("assigned_to = ? OR created_by = ?", userID, userID).
		Order("due_date ASC"). // Order by due date
		Find(&tasks).Error
	return tasks, err
}

// --- Reminder Methods ---

// CreateReminder creates a new reminder.
func (r *Repository) CreateReminder(ctx context.Context, reminder *model.Reminder) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

// GetReminderByID retrieves a reminder by its ID.
func (r *Repository) GetReminderByID(ctx context.Context, reminderID uuid.UUID) (*model.Reminder, error) {
	var reminder model.Reminder
	err := r.db.WithContext(ctx).Preload("User").First(&reminder, reminderID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &reminder, nil
}

// UpdateReminder updates an existing reminder.
func (r *Repository) UpdateReminder(ctx context.Context, reminder *model.Reminder) error {
	return r.db.WithContext(ctx).Save(reminder).Error
}

// DeleteReminder deletes a reminder by its ID.
func (r *Repository) DeleteReminder(ctx context.Context, reminderID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Reminder{}, reminderID).Error
}

// GetRemindersByUserID retrieves all reminders for a user.
func (r *Repository) GetRemindersByUserID(ctx context.Context, userID uuid.UUID) ([]model.Reminder, error) {
	var reminders []model.Reminder
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("user_id = ?", userID).
		Order("remind_at ASC"). // Order by reminder time
		Find(&reminders).Error
	return reminders, err
}

// GetPendingReminders retrieves all reminders that are due to be triggered.
func (r *Repository) GetPendingReminders(ctx context.Context) ([]model.Reminder, error) {
	var reminders []model.Reminder
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("remind_at <= ? AND triggered = ?", time.Now(), false).
		Order("remind_at ASC").
		Find(&reminders).Error
	return reminders, err
}

// GetAllClients retrieves all clients from db
func (r *Repository) GetAllClients(ctx context.Context) ([]model.Client, error) {
	var clients []model.Client
	err := r.db.WithContext(ctx).
		Order("name ASC"). // Order by due date
		Find(&clients).Error
	return clients, err
}

// internal/repository/repository.go
// Adding GetAllUsers
func (r *Repository) GetAllUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// DeleteConversation implements Repository.
func (r *Repository) DeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	//First Delete associations
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Delete(&model.ConversationParticipant{}).Error
	if err != nil {
		return err
	}
	//Then delete conversation
	result := r.db.WithContext(ctx).Delete(&model.Conversation{}, conversationID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound // No conversation found with the given ID
	}

	return nil
}
