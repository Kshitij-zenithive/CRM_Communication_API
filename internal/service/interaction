// internal/service/interaction.go (Corrected model usage)
package service

import (
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"fmt"

	"github.com/google/uuid"
)

// InteractionService handles interactions between users and clients.
type InteractionService struct {
	repo *repository.Repository
}

// NewInteractionService creates a new InteractionService.
func NewInteractionService(repo *repository.Repository) *InteractionService {
	return &InteractionService{repo: repo}
}

// RecordInteraction creates a timeline event for a given interaction.
func (s *InteractionService) RecordInteraction(ctx context.Context, interaction model.Interaction) error {
	eventableID, err := uuid.Parse(interaction.GetID())
	if err != nil {
		return fmt.Errorf("invalid eventable ID: %w", err)
	}
	// Create a timeline event
	event := model.TimelineEvent{
		UserID:        interaction.GetUser().ID,
		EventableType: interaction.GetType(),
		EventableID:   eventableID,
		EventType:     interaction.GetType(), // You could have more specific event types
		EventTime:     interaction.GetCreatedAt(),
	}

	// Handle cases where Client might be nil
	if interaction.GetClient() != nil {
		event.ClientID = &interaction.GetClient().ID
	}

	switch interaction.GetType() {
	case string(model.InteractionTypeChatMessage):
		msg := interaction.(*model.Message) // Assert to Message
		conversation, err := s.repo.GetConversationByID(ctx, msg.ConversationID)
		if err != nil {
			// Decide how to continue in case of error, log it, but don't stop saving the timeline:
			fmt.Println("Failed to fetch conversation", err)
		}
		if conversation != nil && conversation.Type == model.ConversationTypeClient {
			event.Title = fmt.Sprintf("Chat message with %s", conversation.Client.Name) // Access Client via Conversation
		} else if conversation != nil && conversation.Type == model.ConversationTypeGroup {
			event.Title = fmt.Sprintf("Chat message in group: %s", *conversation.Name) // Access Client via Conversation
		} else {
			event.Title = "Chat message"
		}
		event.Content = msg.Content

	case string(model.InteractionTypeEmailSent), string(model.InteractionTypeEmailReceived):
		email := interaction.(*model.Email) // Assert to Email
		if email.GetType() == string(model.InteractionTypeEmailSent) {
			event.Title = fmt.Sprintf("Email sent to %s", email.To)
		} else {
			event.Title = fmt.Sprintf("Email received from %s", email.From)
		}
		event.Content = fmt.Sprintf("Subject: %s, Snippet: %s", email.Subject, email.Snippet)

	case string(model.InteractionTypeTaskCreated):
		task := interaction.(*model.Task) // Assert to Task
		event.Title = fmt.Sprintf("Task created: %s", task.Title)
		event.Content = task.Description

	case string(model.InteractionTypeTaskUpdated):
		task := interaction.(*model.Task)
		event.Title = fmt.Sprintf("Task updated: %s", task.Title)
		event.Content = task.Description //  include details about what was updated

	case string(model.InteractionTypeTaskCompleted):
		task := interaction.(*model.Task)
		event.Title = fmt.Sprintf("Task completed: %s", task.Title)
		event.Content = task.Description

	case string(model.InteractionTypeReminder):
		reminder := interaction.(*model.Reminder) // Assert to Reminder
		event.Title = fmt.Sprintf("Reminder: %s", reminder.Content)
		event.Content = reminder.Content

	default:
		return fmt.Errorf("unknown interaction type: %s", interaction.GetType())
	}

	// Save the timeline event
	return s.repo.CreateTimelineEvent(ctx, &event)
}

// GetTimeline retrieves the timeline events for a specific client.
func (s *InteractionService) GetTimeline(ctx context.Context, clientID uuid.UUID) ([]model.TimelineEvent, error) {
	return s.repo.GetTimelineEventsByClientID(ctx, clientID)
}
