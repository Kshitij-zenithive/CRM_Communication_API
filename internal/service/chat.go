// internal/service/chat.go (Corrected for consistency and added mention handling)

package service

import (
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"crm-communication-api/internal/websocket"
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

type ChatService struct {
	repo      *repository.Repository
	hub       *websocket.Hub
	mentionRe *regexp.Regexp // Compiled regular expression for mentions
}

func NewChatService(repo *repository.Repository, hub *websocket.Hub) *ChatService {
	// Compile the regular expression for mentions (@username)
	mentionRe := regexp.MustCompile(`@([a-zA-Z0-9_]+)`) // Basic username matching
	return &ChatService{
		repo:      repo,
		hub:       hub,
		mentionRe: mentionRe,
	}
}

// CreateConversation creates a new chat conversation.
func (s *ChatService) CreateConversation(ctx context.Context, conversation *model.Conversation, userIDs []string) (*model.Conversation, error) {
	// Validate input
	if conversation.Type != model.ConversationTypeDM && conversation.Type != model.ConversationTypeGroup && conversation.Type != model.ConversationTypeClient {
		return nil, fmt.Errorf("invalid conversation type: %s", conversation.Type)
	}

	if conversation.Type == model.ConversationTypeGroup && conversation.Name == nil {
		return nil, fmt.Errorf("group conversation requires a name")
	}

	if conversation.Type == model.ConversationTypeClient && conversation.ClientID == nil {
		return nil, fmt.Errorf("client conversation requires a client ID")
	}

	// Create the conversation
	if err := s.repo.CreateConversation(ctx, conversation); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Add participants
	for _, userIDStr := range userIDs {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %s", userIDStr)
		}
		participant := &model.ConversationParticipant{
			ConversationID: conversation.ID,
			UserID:         userID,
		}
		if err := s.repo.CreateConversationParticipant(ctx, participant); err != nil {
			// Clean up the created conversation in case of an error
			_ = s.repo.DeleteConversation(ctx, conversation.ID)
			return nil, fmt.Errorf("failed to add participant: %w", err)
		}
	}

	return conversation, nil
}

// GetConversationsByUser retrieves all conversations a user is part of.
func (s *ChatService) GetConversationsByUser(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	conversations, err := s.repo.GetConversationsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}
	return conversations, nil
}

// CreateMessage creates a new message and handles mentions.
func (s *ChatService) CreateMessage(ctx context.Context, message *model.Message) (*model.Message, error) {
	// Create the message
	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Process mentions
	matches := s.mentionRe.FindAllStringSubmatch(message.Content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			mentionedUsername := match[1]
			user, err := s.repo.GetUserByEmail(ctx, mentionedUsername) // Assuming usernames are emails
			if err == nil {
				// Create a mention record
				mention := &model.MessageMention{
					MessageID: message.ID,
					UserID:    user.ID,
				}
				if err := s.repo.CreateMessageMention(ctx, mention); err != nil {
					// Log error but don't fail the entire operation
					fmt.Printf("failed to create mention for user %s: %v\n", mentionedUsername, err)
				}

				// Broadcast a mention notification
				//Corrected
				s.hub.Broadcast <- websocket.Message{
					Type:   "mention",
					RoomID: user.ID.String(), //send only to the mentioned user
					Payload: map[string]interface{}{
						"messageId": message.ID.String(),
						"userId":    user.ID.String(), // Notified user ID
						"senderId":  message.SenderID.String(),
						"content":   message.Content, // Include message
					},
				}

			} else if err != repository.ErrNotFound {
				// Log error, but don't fail message creation
				fmt.Printf("failed to get user %s: %v\n", mentionedUsername, err)
			}
		}
	}

	// Broadcast the new message to all connected clients in the conversation using RoomID
	s.hub.Broadcast <- websocket.Message{
		Type:    "new_message",
		RoomID:  message.ConversationID.String(),
		Payload: message, // Send the entire message object
	}

	return message, nil
}

// GetMessagesByConversationID retrieves messages for a conversation
func (s *ChatService) GetMessagesByConversationID(ctx context.Context, conversationID uuid.UUID) ([]model.Message, error) {
	messages, err := s.repo.GetMessagesByConversationID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, nil
}

// MarkMessageAsRead marks a message as read by a user.
func (s *ChatService) MarkMessageAsRead(ctx context.Context, messageID, userID uuid.UUID) error {
	// Check if a read receipt already exists
	_, err := s.repo.GetMessageReadReceipt(ctx, messageID, userID)
	if err == nil {
		// Read receipt already exists, no need to create a new one
		return nil
	} else if err != repository.ErrNotFound {
		// Some other error occurred
		return fmt.Errorf("failed to check for existing read receipt: %w", err)
	}

	// Create a new read receipt
	readReceipt := &model.MessageReadReceipt{
		MessageID: messageID,
		UserID:    userID,
	}
	if err := s.repo.CreateMessageReadReceipt(ctx, readReceipt); err != nil {
		return fmt.Errorf("failed to create read receipt: %w", err)
	}
	// Broadcast the message read event using RoomID
	//Potentially get Conversation ID for the given message, as the Hub accepts only RoomID now
	message, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil || message == nil {
		return fmt.Errorf("failed to retrieve message, can't broadcast")
	}
	s.hub.Broadcast <- websocket.Message{
		Type:   "message_read",
		RoomID: message.ConversationID.String(),
		Payload: map[string]interface{}{
			"messageId": messageID.String(),
			"userId":    userID.String(),
		},
	}
	return nil
}

// GetConversation retrieves a specific conversation by ID
func (s *ChatService) GetConversation(ctx context.Context, conversationID uuid.UUID) (*model.Conversation, error) {
	return s.repo.GetConversationByID(ctx, conversationID)
}

// DeleteConversation deletes a conversation by its ID.
func (s *ChatService) DeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	return s.repo.DeleteConversation(ctx, conversationID)
}
