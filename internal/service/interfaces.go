// internal/service/interfaces.go (Revised)
package service

import (
	"context"
	dbmodel "crm-communication-api/internal/model" // Use alias consistently
	"time"

	"github.com/google/uuid"
)

// Handles core chat logic, messages, participants, and command parsing/triggering
type ChatService interface {
	CreateConversation(ctx context.Context, conversation *dbmodel.Conversation, participantIDs []uuid.UUID) (*dbmodel.Conversation, error)
	GetConversation(ctx context.Context, conversationID uuid.UUID) (*dbmodel.Conversation, error)
	ListConversationsForUser(ctx context.Context, userID uuid.UUID) ([]dbmodel.Conversation, error)
	AddParticipant(ctx context.Context, conversationID, userID uuid.UUID) error
	RemoveParticipant(ctx context.Context, conversationID, userID uuid.UUID) error

	GetMessage(ctx context.Context, messageID uuid.UUID) (*dbmodel.Message, error)
	ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID /*, pagination... */) ([]dbmodel.Message, error)
	// SendMessage parses content; if it's a command, it calls ProcessCommand; otherwise, saves TEXT message.
	SendMessage(ctx context.Context, conversationID, senderID uuid.UUID, content string) (*dbmodel.Message, error)
	// ProcessCommand handles command logic, potentially schedules background jobs, returns output msg.
	// Made internal for now, called by SendMessage. Could be exposed if needed.
	// processCommand(ctx context.Context, conversationID, senderID uuid.UUID, command string, args []string) (*dbmodel.Message, error)
	MarkMessageAsRead(ctx context.Context, messageID, userID uuid.UUID) error
	// Add methods to get mentions/read receipts if those models are re-introduced
}

// Handles email sending, syncing, and template operations
type EmailService interface {
	SendEmail(ctx context.Context, email *dbmodel.Email) (*dbmodel.Email, error)
	SendEmailWithTemplate(ctx context.Context, senderID, clientID, templateID uuid.UUID, recipientEmails []string, variables map[string]interface{}) (*dbmodel.Email, error)
	SyncGmail(ctx context.Context, userID string) error // Sync for a specific user
	GetEmails(ctx context.Context, clientID uuid.UUID) ([]dbmodel.Email, error)
	GetEmail(ctx context.Context, emailID uuid.UUID) (*dbmodel.Email, error)
}

// Handles email template CRUD operations
type TemplateService interface {
	CreateEmailTemplate(ctx context.Context, template dbmodel.EmailTemplate) (*dbmodel.EmailTemplate, error)
	GetEmailTemplate(ctx context.Context, id uuid.UUID) (*dbmodel.EmailTemplate, error)
	UpdateEmailTemplate(ctx context.Context, id uuid.UUID, template dbmodel.EmailTemplate) (*dbmodel.EmailTemplate, error)
	DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error
	GetAllEmailTemplates(ctx context.Context) ([]dbmodel.EmailTemplate, error)
	RenderTemplate(template *dbmodel.EmailTemplate, data map[string]interface{}) (subject string, body string, err error) // Utility, might live elsewhere too
}

// Handles creation and retrieval of simplified timeline events
type TimelineService interface {
	CreateTimelineEvent(ctx context.Context, event *dbmodel.TimelineEvent) (*dbmodel.TimelineEvent, error)
	GetTimelineEvents(ctx context.Context, clientID *uuid.UUID, userID *uuid.UUID /*, filters... */) ([]dbmodel.TimelineEvent, error)
	// We no longer need Update/Delete if timeline events are immutable logs
}

// Handles broadcasting real-time events (Needs implementation with Hub)
type NotificationService interface {
	BroadcastNewMessage(ctx context.Context, conversationID uuid.UUID, message *dbmodel.Message) error
	BroadcastMessageRead(ctx context.Context, conversationID uuid.UUID, payload map[string]interface{}) error // <<< ADDED
	// BroadcastConversationUpdate(ctx context.Context, conversation *dbmodel.Conversation) error // Keep if needed
	GetNewMessageChannel(ctx context.Context, conversationID uuid.UUID) (<-chan *dbmodel.Message, error) // For GraphQL Subscription resolver
}

// Simple interfaces for user/client lookup needed by other services
type UserService interface {
	GetUser(ctx context.Context, id uuid.UUID) (*dbmodel.User, error)
}
type ClientService interface {
	GetClient(ctx context.Context, id uuid.UUID) (*dbmodel.Client, error)
}

// Placeholder interface for scheduling background tasks (like reminders)
// The implementation would interact with gocron, Asynq, RabbitMQ, etc.
type SchedulerService interface {
	ScheduleReminder(ctx context.Context, remindAt time.Time, userID uuid.UUID, messageContent string, conversationID uuid.UUID) error
	// ScheduleTaskFollowUp(...)
}
