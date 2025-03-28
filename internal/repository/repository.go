// // internal/repository/repository.go
// package repository

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"log/slog" // Import slog for logging within the repo if desired
// 	// "time"

// 	dbmodel "crm-communication-api/internal/model" // Alias for database models

// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// 	"gorm.io/gorm/clause" // Import clause for OnConflict
// )

// // --- Errors ---

// // ErrNotFound indicates that a requested record was not found.
// var ErrNotFound = errors.New("record not found")

// // --- Interfaces ---

// // AuthRepository defines methods for accessing authentication-related data.
// type AuthRepository interface {
// 	GetUserByEmail(ctx context.Context, email string) (*dbmodel.User, error)
// 	GetUserByID(ctx context.Context, id uuid.UUID) (*dbmodel.User, error)
// 	CreateUser(ctx context.Context, user *dbmodel.User) error
// 	UpdateUser(ctx context.Context, user *dbmodel.User) error
// 	CreateRefreshToken(ctx context.Context, token *dbmodel.RefreshToken) error
// 	// Renamed GetRefreshToken -> FindRefreshToken for clarity
// 	FindRefreshToken(ctx context.Context, tokenString string) (*dbmodel.RefreshToken, error)
// 	// Changed DeleteRefreshToken to accept ID for consistency after finding the token
// 	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
// 	GetOAuthProvider(ctx context.Context, userID uuid.UUID, provider string) (*dbmodel.OAuthProvider, error)
// 	// CreateOAuthProvider(ctx context.Context, provider *dbmodel.OAuthProvider) error // Keep if needed separately
// 	// UpdateOAuthProvider(ctx context.Context, provider *dbmodel.OAuthProvider) error // Keep if needed separately
// 	// Add the new combined method
// 	CreateOrUpdateOAuthProvider(ctx context.Context, provider *dbmodel.OAuthProvider) error
// }

// // ConversationRepository defines methods for conversation data access.
// type ConversationRepository interface {
// 	// TODO: Define methods
// }

// // MessageRepository defines methods for message data access.
// type MessageRepository interface {
// 	// TODO: Define methods
// }

// // EmailRepository defines methods for email data access.
// type EmailRepository interface {
// 	// TODO: Define methods
// }

// // ClientRepository defines methods for client data access.
// type ClientRepository interface {
// 	// TODO: Define methods
// }

// // TemplateRepository defines methods for template data access.
// type TemplateRepository interface {
// 	// TODO: Define methods
// }

// // --- Main Repository ---

// // Repository holds the database connection and implements all repository interfaces.
// type Repository struct {
// 	db     *gorm.DB
// 	logger *slog.Logger // Add logger for internal repository logging if needed
// }

// // Assert that Repository implements AuthRepository (and others when defined)
// var _ AuthRepository = (*Repository)(nil)
// // ... other interface assertions

// // NewRepository creates a new Repository instance.
// // Optionally accept a logger.
// func NewRepository(db *gorm.DB /*, logger *slog.Logger*/) *Repository {
// 	// If logger is passed, assign it. Otherwise, use a default discard logger or configure one.
// 	// repoLogger := logger
// 	// if repoLogger == nil {
// 	//     repoLogger = slog.New(slog.NewTextHandler(io.Discard, nil)) // Discard logs by default
// 	// }
// 	return &Repository{
// 		db: db,
// 		// logger: repoLogger.With(slog.String("component", "repository")),
// 	}
// }

// // --- AuthRepository Implementation ---

// func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*dbmodel.User, error) {
// 	var user dbmodel.User
// 	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, ErrNotFound
// 		}
// 		// r.logger.ErrorContext(ctx, "Database error getting user by email", slog.String("email", email), slog.String("error", err.Error()))
// 		return nil, fmt.Errorf("failed to get user by email: %w", err)
// 	}
// 	return &user, nil
// }

// func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*dbmodel.User, error) {
// 	var user dbmodel.User
// 	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, ErrNotFound
// 		}
// 		// r.logger.ErrorContext(ctx, "Database error getting user by ID", slog.String("userID", id.String()), slog.String("error", err.Error()))
// 		return nil, fmt.Errorf("failed to get user by id: %w", err)
// 	}
// 	return &user, nil
// }

// func (r *Repository) CreateUser(ctx context.Context, user *dbmodel.User) error {
// 	err := r.db.WithContext(ctx).Create(user).Error
// 	if err != nil {
// 		// r.logger.ErrorContext(ctx, "Database error creating user", slog.String("email", user.Email), slog.String("error", err.Error()))
// 		// Consider specific DB error checking for duplicate email here
// 		return fmt.Errorf("failed to create user: %w", err)
// 	}
// 	// r.logger.DebugContext(ctx, "User created successfully", slog.String("userID", user.ID.String()))
// 	return nil
// }

// func (r *Repository) UpdateUser(ctx context.Context, user *dbmodel.User) error {
// 	if user.ID == uuid.Nil {
// 		return fmt.Errorf("cannot update user with nil ID")
// 	}
// 	err := r.db.WithContext(ctx).Save(user).Error
// 	if err != nil {
// 		// r.logger.ErrorContext(ctx, "Database error updating user", slog.String("userID", user.ID.String()), slog.String("error", err.Error()))
// 		return fmt.Errorf("failed to update user: %w", err)
// 	}
// 	// r.logger.DebugContext(ctx, "User updated successfully", slog.String("userID", user.ID.String()))
// 	return nil
// }

// func (r *Repository) CreateRefreshToken(ctx context.Context, token *dbmodel.RefreshToken) error {
// 	err := r.db.WithContext(ctx).Create(token).Error
// 	if err != nil {
// 		// r.logger.ErrorContext(ctx, "Database error creating refresh token", slog.String("userID", token.UserID.String()), slog.String("error", err.Error()))
// 		return fmt.Errorf("failed to create refresh token: %w", err)
// 	}
// 	// r.logger.DebugContext(ctx, "Refresh token created successfully", slog.String("userID", token.UserID.String()))
// 	return nil
// }

// // FindRefreshToken retrieves a refresh token by its token string.
// // It does NOT automatically check for expiry here; expiry check is done in AuthService.
// // Renamed from GetRefreshToken for clarity.
// func (r *Repository) FindRefreshToken(ctx context.Context, tokenString string) (*dbmodel.RefreshToken, error) {
// 	var token dbmodel.RefreshToken
// 	err := r.db.WithContext(ctx).
// 		Where("token = ?", tokenString).
// 		First(&token).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, ErrNotFound
// 		}
// 		// r.logger.ErrorContext(ctx, "Database error finding refresh token", slog.String("error", err.Error()))
// 		return nil, fmt.Errorf("failed to find refresh token: %w", err)
// 	}
// 	return &token, nil
// }

// // DeleteRefreshToken deletes a refresh token by its primary key ID.
// // Changed parameter from tokenString to ID for consistency after finding the token.
// func (r *Repository) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
// 	result := r.db.WithContext(ctx).Delete(&dbmodel.RefreshToken{}, id)
// 	if result.Error != nil {
// 		// r.logger.ErrorContext(ctx, "Database error deleting refresh token", slog.String("tokenID", id.String()), slog.String("error", result.Error.Error()))
// 		return fmt.Errorf("failed to delete refresh token: %w", result.Error)
// 	}
// 	// Log if needed, check result.RowsAffected if necessary
// 	// if result.RowsAffected == 0 {
// 	//     r.logger.WarnContext(ctx,"Attempted to delete non-existent refresh token", slog.String("tokenID", id.String()))
// 	//     return ErrNotFound // Or return nil if idempotent deletion is acceptable
// 	// }
// 	// r.logger.DebugContext(ctx, "Refresh token deleted successfully", slog.String("tokenID", id.String()))
// 	return nil
// }

// func (r *Repository) GetOAuthProvider(ctx context.Context, userID uuid.UUID, provider string) (*dbmodel.OAuthProvider, error) {
// 	var providerInfo dbmodel.OAuthProvider
// 	err := r.db.WithContext(ctx).
// 		Where("user_id = ? AND provider = ?", userID, provider).
// 		First(&providerInfo).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, ErrNotFound
// 		}
// 		// r.logger.ErrorContext(ctx, "Database error getting OAuth provider", slog.String("userID", userID.String()), slog.String("provider", provider), slog.String("error", err.Error()))
// 		return nil, fmt.Errorf("failed to get oauth provider: %w", err)
// 	}
// 	return &providerInfo, nil
// }

// // CreateOrUpdateOAuthProvider inserts a new OAuthProvider record or updates an existing one
// // based on the unique constraint (user_id, provider).
// // This uses GORM's OnConflict clause for an atomic "Upsert" operation.
// func (r *Repository) CreateOrUpdateOAuthProvider(ctx context.Context, provider *dbmodel.OAuthProvider) error {
// 	if provider.UserID == uuid.Nil || provider.Provider == "" {
// 		return fmt.Errorf("cannot create or update oauth provider with empty user ID or provider")
// 	}

// 	// Use Clauses(clause.OnConflict...) to perform an Upsert.
// 	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
// 		// Specify the columns that define the unique constraint.
// 		// Ensure you have a unique index on (user_id, provider) in your database schema.
// 		Columns: []clause.Column{{Name: "user_id"}, {Name: "provider"}},
// 		// Specify which columns to update if a conflict occurs.
// 		// Use AssignmentColumns to list the columns explicitly by name.
// 		// This prevents updating immutable fields like user_id, provider, or created_at.
// 		DoUpdates: clause.AssignmentColumns([]string{
// 			"provider_id",
// 			"access_token",
// 			"refresh_token",
// 			"expires_at",
// 			// Include "updated_at" if you want GORM to manage it automatically on update
// 			// "updated_at",
// 		}),
// 		// Alternatively, use clause.AssignAll to update all fields except the conflict keys and primary key.
// 		// Use with caution: DoUpdates: clause.AssignAll,
// 	}).Create(provider).Error // Create attempts to insert, OnConflict handles the update case.

// 	if err != nil {
// 		// r.logger.ErrorContext(ctx, "Database error creating or updating OAuth provider", slog.String("userID", provider.UserID.String()), slog.String("provider", provider.Provider), slog.String("error", err.Error()))
// 		return fmt.Errorf("failed to create or update oauth provider: %w", err)
// 	}

// 	// r.logger.DebugContext(ctx, "OAuth provider created or updated successfully", slog.String("userID", provider.UserID.String()), slog.String("provider", provider.Provider))
// 	return nil
// }

// // --- Implementations for other repositories go here ---



// internal/repository/repository.go
package repository

import (
	"context"
	"errors"
	"fmt"
	// "time"

	dbmodel "crm-communication-api/internal/model" // Alias for database models

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // Now needed for Preload and OnConflict
)

// --- Errors ---
var ErrNotFound = errors.New("record not found")

// --- Interfaces ---

// AuthRepository definition remains the same as corrected previously
type AuthRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*dbmodel.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*dbmodel.User, error)
	CreateUser(ctx context.Context, user *dbmodel.User) error
	UpdateUser(ctx context.Context, user *dbmodel.User) error
	CreateRefreshToken(ctx context.Context, token *dbmodel.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenString string) (*dbmodel.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error // Changed param to ID
	GetOAuthProvider(ctx context.Context, userID uuid.UUID, provider string) (*dbmodel.OAuthProvider, error)
	CreateOrUpdateOAuthProvider(ctx context.Context, provider *dbmodel.OAuthProvider) error // Upsert logic
}

// ConversationRepository defines methods for conversation data access.
type ConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *dbmodel.Conversation) error
	CreateConversationParticipant(ctx context.Context, participant *dbmodel.ConversationParticipant) error
	GetConversationByID(ctx context.Context, id uuid.UUID) (*dbmodel.Conversation, error)            // Preloads Participants.User and Client
	GetConversationsByUser(ctx context.Context, userID uuid.UUID) ([]dbmodel.Conversation, error) // Preloads Participants.User and Client
	DeleteConversation(ctx context.Context, id uuid.UUID) error
	GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]dbmodel.ConversationParticipant, error) // Preloads User
	RemoveConversationParticipant(ctx context.Context, conversationID, userID uuid.UUID) error
}

// MessageRepository defines methods for message data access.
type MessageRepository interface {
	CreateMessage(ctx context.Context, message *dbmodel.Message) error
	GetMessageByID(ctx context.Context, id uuid.UUID) (*dbmodel.Message, error)                                        // Preloads Sender and Conversation
	GetMessagesByConversationID(ctx context.Context, conversationID uuid.UUID /*, limit, offset */) ([]dbmodel.Message, error) // Preloads Sender
	// GetMessageMentions is removed as the model was removed
	GetMessageReadBy(ctx context.Context, messageID uuid.UUID) ([]dbmodel.User, error) // Gets users who read the message
}

// EmailRepository defines methods for email data access.
type EmailRepository interface {
	CreateEmail(ctx context.Context, email *dbmodel.Email) error
	GetEmailByID(ctx context.Context, id uuid.UUID) (*dbmodel.Email, error)                // Preloads Client, User, Attachments
	GetEmailsByClientID(ctx context.Context, clientID uuid.UUID) ([]dbmodel.Email, error)   // Preloads Client, User
	GetEmailsByUserID(ctx context.Context, userID uuid.UUID) ([]dbmodel.Email, error)       // Preloads Client, User
	GetEmailByProviderID(ctx context.Context, provider, providerID string) (*dbmodel.Email, error) // For sync checks
	CreateEmailAttachment(ctx context.Context, attachment *dbmodel.EmailAttachment) error
}

// ClientRepository defines methods for client data access.
type ClientRepository interface {
	CreateClient(ctx context.Context, client *dbmodel.Client) error
	GetClientByID(ctx context.Context, id uuid.UUID) (*dbmodel.Client, error)
	GetClientByEmail(ctx context.Context, email string) (*dbmodel.Client, error)
	GetAllClients(ctx context.Context) ([]dbmodel.Client, error)
	UpdateClient(ctx context.Context, client *dbmodel.Client) error // Added Update
	// DeleteClient(ctx context.Context, id uuid.UUID) error // Optional Delete
}

// TemplateRepository defines methods for template data access.
type TemplateRepository interface {
	CreateEmailTemplate(ctx context.Context, template *dbmodel.EmailTemplate) error
	GetEmailTemplateByID(ctx context.Context, id uuid.UUID) (*dbmodel.EmailTemplate, error)
	GetEmailTemplateByName(ctx context.Context, name string) (*dbmodel.EmailTemplate, error)
	GetAllEmailTemplates(ctx context.Context) ([]dbmodel.EmailTemplate, error)
	UpdateEmailTemplate(ctx context.Context, template *dbmodel.EmailTemplate) error
	DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error
}

// TimelineRepository defines methods for timeline event data access.
type TimelineRepository interface {
	CreateTimelineEvent(ctx context.Context, event *dbmodel.TimelineEvent) error
	GetTimelineEvents(ctx context.Context, clientID *uuid.UUID, userID *uuid.UUID /*, limit, offset */) ([]dbmodel.TimelineEvent, error) // Preloads User, Client
}


// --- Main Repository ---

// Repository holds the database connection and implements all repository interfaces.
type Repository struct {
	db *gorm.DB
	// logger *slog.Logger // Optional logger
}

// Assert that Repository implements all interfaces
var _ AuthRepository         = (*Repository)(nil)
var _ ConversationRepository = (*Repository)(nil)
var _ MessageRepository      = (*Repository)(nil)
var _ EmailRepository        = (*Repository)(nil)
var _ ClientRepository       = (*Repository)(nil)
var _ TemplateRepository     = (*Repository)(nil)
var _ TimelineRepository     = (*Repository)(nil)

// NewRepository creates a new Repository instance.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- AuthRepository Implementation (from previous step, verified) ---

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*dbmodel.User, error) {
	var user dbmodel.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}
func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*dbmodel.User, error) {
	var user dbmodel.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &user, nil
}
func (r *Repository) CreateUser(ctx context.Context, user *dbmodel.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil { return fmt.Errorf("failed to create user: %w", err) }
	return nil
}
func (r *Repository) UpdateUser(ctx context.Context, user *dbmodel.User) error {
	if user.ID == uuid.Nil { return fmt.Errorf("cannot update user with nil ID") }
	err := r.db.WithContext(ctx).Save(user).Error
	if err != nil { return fmt.Errorf("failed to update user: %w", err) }
	return nil
}
func (r *Repository) CreateRefreshToken(ctx context.Context, token *dbmodel.RefreshToken) error {
	err := r.db.WithContext(ctx).Create(token).Error
	if err != nil { return fmt.Errorf("failed to create refresh token: %w", err) }
	return nil
}
func (r *Repository) FindRefreshToken(ctx context.Context, tokenString string) (*dbmodel.RefreshToken, error) {
	var token dbmodel.RefreshToken
	// Expiry check moved to service layer for clarity, repo just finds the token
	err := r.db.WithContext(ctx).Where("token = ?", tokenString).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	return &token, nil
}
func (r *Repository) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&dbmodel.RefreshToken{}, id)
	if result.Error != nil { return fmt.Errorf("failed to delete refresh token: %w", result.Error) }
	// Optional: Check result.RowsAffected == 0 and return ErrNotFound if needed
	return nil
}
func (r *Repository) GetOAuthProvider(ctx context.Context, userID uuid.UUID, provider string) (*dbmodel.OAuthProvider, error) {
	var providerInfo dbmodel.OAuthProvider
	err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).First(&providerInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get oauth provider: %w", err)
	}
	return &providerInfo, nil
}
func (r *Repository) CreateOrUpdateOAuthProvider(ctx context.Context, provider *dbmodel.OAuthProvider) error {
	if provider.UserID == uuid.Nil || provider.Provider == "" {
		return fmt.Errorf("cannot create or update oauth provider with empty user ID or provider")
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "provider"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider_id", "access_token", "refresh_token", "expires_at", "updated_at"}),
	}).Create(provider).Error
	if err != nil {
		return fmt.Errorf("failed to create or update oauth provider: %w", err)
	}
	return nil
}

// --- ConversationRepository Implementation ---

func (r *Repository) CreateConversation(ctx context.Context, conversation *dbmodel.Conversation) error {
	err := r.db.WithContext(ctx).Create(conversation).Error
	if err != nil { return fmt.Errorf("failed to create conversation: %w", err) }
	return nil
}

func (r *Repository) CreateConversationParticipant(ctx context.Context, participant *dbmodel.ConversationParticipant) error {
	// Using OnConflict here prevents errors if participant already exists, effectively making it idempotent
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "conversation_id"}, {Name: "user_id"}},
		DoNothing: true, // If participant exists, do nothing
		// Or use DoUpdates if you want to update 'joined_at' or other fields on re-add
		// DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(participant).Error
	if err != nil { return fmt.Errorf("failed to create conversation participant: %w", err) }
	return nil
}

func (r *Repository) GetConversationByID(ctx context.Context, id uuid.UUID) (*dbmodel.Conversation, error) {
	var conversation dbmodel.Conversation
	err := r.db.WithContext(ctx).
		Preload("Participants.User"). // Eager load participant user details
		Preload("Client").            // Eager load client details if ClientID is set
		Where("id = ?", id).
		First(&conversation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get conversation by id: %w", err)
	}
	return &conversation, nil
}

func (r *Repository) GetConversationsByUser(ctx context.Context, userID uuid.UUID) ([]dbmodel.Conversation, error) {
	var conversations []dbmodel.Conversation
	// Find conversation IDs the user participates in
	var participantEntries []dbmodel.ConversationParticipant
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&participantEntries).Error; err != nil {
		return nil, fmt.Errorf("failed to get user conversation participation: %w", err)
	}
	if len(participantEntries) == 0 {
		return []dbmodel.Conversation{}, nil // Return empty slice, not error
	}

	conversationIDs := make([]uuid.UUID, len(participantEntries))
	for i, p := range participantEntries {
		conversationIDs[i] = p.ConversationID
	}

	// Fetch those conversations with necessary preloads
	err := r.db.WithContext(ctx).
		Preload("Participants.User").
		Preload("Client").
		Where("id IN ?", conversationIDs).
		Order("updated_at DESC"). // Order by most recently updated
		Find(&conversations).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get conversations by user: %w", err)
	}
	return conversations, nil
}

func (r *Repository) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	// GORM automatically handles associated records based on constraints defined in models
	// If using `constraint:OnDelete:CASCADE`, deleting Conversation might cascade.
	// If not, delete participants and messages manually first in a transaction.
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction for deleting conversation: %w", tx.Error)
	}

	// 1. Delete Messages (if not cascaded)
	if err := tx.Where("conversation_id = ?", id).Delete(&dbmodel.Message{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete messages for conversation %s: %w", id, err)
	}

	// 2. Delete Participants (if not cascaded)
	if err := tx.Where("conversation_id = ?", id).Delete(&dbmodel.ConversationParticipant{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete participants for conversation %s: %w", id, err)
	}

	// 3. Delete Conversation itself
	result := tx.Delete(&dbmodel.Conversation{}, id)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete conversation %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		tx.Rollback() // Nothing was deleted
		return ErrNotFound
	}

	return tx.Commit().Error // Commit transaction
}

func (r *Repository) GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]dbmodel.ConversationParticipant, error) {
	var participants []dbmodel.ConversationParticipant
	err := r.db.WithContext(ctx).
		Preload("User"). // Preload the User details for each participant
		Where("conversation_id = ?", conversationID).
		Find(&participants).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation participants: %w", err)
	}
	return participants, nil
}

func (r *Repository) RemoveConversationParticipant(ctx context.Context, conversationID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&dbmodel.ConversationParticipant{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove participant: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound // Participant wasn't found
	}
	return nil
}

// --- MessageRepository Implementation ---

func (r *Repository) CreateMessage(ctx context.Context, message *dbmodel.Message) error {
	err := r.db.WithContext(ctx).Create(message).Error
	if err != nil { return fmt.Errorf("failed to create message: %w", err) }
	// Need to update Conversation's UpdatedAt? GORM might handle this with association hooks,
	// or do it manually here or in the service.
	// r.db.Model(&dbmodel.Conversation{}).Where("id = ?", message.ConversationID).Update("updated_at", time.Now())
	return nil
}

func (r *Repository) GetMessageByID(ctx context.Context, id uuid.UUID) (*dbmodel.Message, error) {
	var message dbmodel.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Conversation").
		First(&message, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get message by id: %w", err)
	}
	return &message, nil
}

func (r *Repository) GetMessagesByConversationID(ctx context.Context, conversationID uuid.UUID /*, limit, offset */) ([]dbmodel.Message, error) {
	var messages []dbmodel.Message
	query := r.db.WithContext(ctx).
		Preload("Sender"). // Eager load sender details
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC") // Order messages chronologically

	// Add pagination if limit/offset are implemented
	// if limit > 0 { query = query.Limit(limit) }
	// if offset > 0 { query = query.Offset(offset) }

	err := query.Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get messages by conversation id: %w", err)
	}
	return messages, nil
}

// GetMessageReadBy retrieves users who have read a specific message.
func (r *Repository) GetMessageReadBy(ctx context.Context, messageID uuid.UUID) ([]dbmodel.User, error) {
    var users []dbmodel.User
    // Find user IDs from the receipts table
    var readReceipts []dbmodel.MessageReadReceipt
    if err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Find(&readReceipts).Error; err != nil {
        return nil, fmt.Errorf("failed to get read receipts for message %s: %w", messageID, err)
    }
    if len(readReceipts) == 0 {
        return []dbmodel.User{}, nil // No one read it yet
    }

    userIDs := make([]uuid.UUID, len(readReceipts))
    for i, receipt := range readReceipts {
        userIDs[i] = receipt.UserID
    }

    // Fetch user details for those IDs
    if err := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
        return nil, fmt.Errorf("failed to get users who read message %s: %w", messageID, err)
    }
    return users, nil
}


// --- EmailRepository Implementation ---

func (r *Repository) CreateEmail(ctx context.Context, email *dbmodel.Email) error {
	err := r.db.WithContext(ctx).Create(email).Error
	if err != nil { return fmt.Errorf("failed to create email: %w", err) }
	return nil
}

func (r *Repository) GetEmailByID(ctx context.Context, id uuid.UUID) (*dbmodel.Email, error) {
	var email dbmodel.Email
	err := r.db.WithContext(ctx).
		Preload("Client").
		Preload("User").
		Preload("Attachments").
		First(&email, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get email by id: %w", err)
	}
	return &email, nil
}

func (r *Repository) GetEmailsByClientID(ctx context.Context, clientID uuid.UUID) ([]dbmodel.Email, error) {
	var emails []dbmodel.Email
	err := r.db.WithContext(ctx).
		Preload("Client"). // Optional, client is known
		Preload("User").   // Preload associated internal user
		Where("client_id = ?", clientID).
		Order("created_at DESC"). // Or received_at/sent_at
		Find(&emails).Error
	if err != nil { return nil, fmt.Errorf("failed to get emails by client id: %w", err) }
	return emails, nil
}

func (r *Repository) GetEmailsByUserID(ctx context.Context, userID uuid.UUID) ([]dbmodel.Email, error) {
	var emails []dbmodel.Email
	err := r.db.WithContext(ctx).
		Preload("Client"). // Preload associated client
		Preload("User").   // Optional, user is known
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&emails).Error
	if err != nil { return nil, fmt.Errorf("failed to get emails by user id: %w", err) }
	return emails, nil
}

func (r *Repository) GetEmailByProviderID(ctx context.Context, provider, providerID string) (*dbmodel.Email, error) {
	var email dbmodel.Email
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_id = ?", provider, providerID).
		First(&email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get email by provider id: %w", err)
	}
	return &email, nil
}

func (r *Repository) CreateEmailAttachment(ctx context.Context, attachment *dbmodel.EmailAttachment) error {
	err := r.db.WithContext(ctx).Create(attachment).Error
	if err != nil { return fmt.Errorf("failed to create email attachment: %w", err) }
	return nil
}

// --- ClientRepository Implementation ---

func (r *Repository) CreateClient(ctx context.Context, client *dbmodel.Client) error {
	err := r.db.WithContext(ctx).Create(client).Error
	if err != nil { return fmt.Errorf("failed to create client: %w", err) }
	return nil
}
func (r *Repository) GetClientByID(ctx context.Context, id uuid.UUID) (*dbmodel.Client, error) {
	var client dbmodel.Client
	err := r.db.WithContext(ctx).First(&client, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get client by id: %w", err)
	}
	return &client, nil
}
func (r *Repository) GetClientByEmail(ctx context.Context, email string) (*dbmodel.Client, error) {
	var client dbmodel.Client
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get client by email: %w", err)
	}
	return &client, nil
}
func (r *Repository) GetAllClients(ctx context.Context) ([]dbmodel.Client, error) {
	var clients []dbmodel.Client
	err := r.db.WithContext(ctx).Order("name ASC").Find(&clients).Error
	if err != nil { return nil, fmt.Errorf("failed to get all clients: %w", err) }
	return clients, nil
}
func (r *Repository) UpdateClient(ctx context.Context, client *dbmodel.Client) error {
	if client.ID == uuid.Nil { return fmt.Errorf("cannot update client with nil ID") }
	err := r.db.WithContext(ctx).Save(client).Error
	if err != nil { return fmt.Errorf("failed to update client: %w", err) }
	return nil
}

// --- TemplateRepository Implementation ---

func (r *Repository) CreateEmailTemplate(ctx context.Context, template *dbmodel.EmailTemplate) error {
	err := r.db.WithContext(ctx).Create(template).Error
	if err != nil { return fmt.Errorf("failed to create email template: %w", err) }
	return nil
}

func (r *Repository) GetEmailTemplateByID(ctx context.Context, id uuid.UUID) (*dbmodel.EmailTemplate, error) {
	var template dbmodel.EmailTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get email template by id: %w", err)
	}
	return &template, nil
}

func (r *Repository) GetEmailTemplateByName(ctx context.Context, name string) (*dbmodel.EmailTemplate, error) {
	var template dbmodel.EmailTemplate
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrNotFound }
		return nil, fmt.Errorf("failed to get email template by name: %w", err)
	}
	return &template, nil
}

func (r *Repository) GetAllEmailTemplates(ctx context.Context) ([]dbmodel.EmailTemplate, error) {
	var templates []dbmodel.EmailTemplate
	err := r.db.WithContext(ctx).Order("name ASC").Find(&templates).Error
	if err != nil { return nil, fmt.Errorf("failed to get all email templates: %w", err) }
	return templates, nil
}

func (r *Repository) UpdateEmailTemplate(ctx context.Context, template *dbmodel.EmailTemplate) error {
	if template.ID == uuid.Nil { return fmt.Errorf("cannot update email template with nil ID") }
	err := r.db.WithContext(ctx).Save(template).Error
	if err != nil { return fmt.Errorf("failed to update email template: %w", err) }
	return nil
}

func (r *Repository) DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&dbmodel.EmailTemplate{}, id)
	if result.Error != nil { return fmt.Errorf("failed to delete email template: %w", result.Error) }
	if result.RowsAffected == 0 { return ErrNotFound }
	return nil
}

// --- TimelineRepository Implementation ---

func (r *Repository) CreateTimelineEvent(ctx context.Context, event *dbmodel.TimelineEvent) error {
	err := r.db.WithContext(ctx).Create(event).Error
	if err != nil { return fmt.Errorf("failed to create timeline event: %w", err) }
	return nil
}

func (r *Repository) GetTimelineEvents(ctx context.Context, clientID *uuid.UUID, userID *uuid.UUID /*, limit, offset */) ([]dbmodel.TimelineEvent, error) {
	var events []dbmodel.TimelineEvent
	query := r.db.WithContext(ctx).Preload("User").Preload("Client") // Eager load User and Client

	if clientID != nil {
		query = query.Where("client_id = ?", clientID)
	}
	if userID != nil {
		query = query.Where("user_id = ?", userID)
	}

	// Add pagination if needed
	// if limit > 0 { query = query.Limit(limit) }
	// if offset > 0 { query = query.Offset(offset) }

	err := query.Order("event_time DESC").Find(&events).Error // Order by event time, descending
	if err != nil { return nil, fmt.Errorf("failed to get timeline events: %w", err) }
	return events, nil
}