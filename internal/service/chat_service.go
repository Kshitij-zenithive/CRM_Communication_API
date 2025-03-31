// internal/service/chat_service.go
package service

import (
	"context"
	dbmodel "crm-communication-api/internal/model" // Use alias
	"crm-communication-api/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time" // Needed for reminder parsing
	"crm-communication-api/internal/util" // Assuming util package for GenerateRandomString

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// chatService implements the ChatService interface.
type chatService struct {
	convRepo        repository.ConversationRepository
	msgRepo         repository.MessageRepository
	userRepo        repository.AuthRepository // Needed to validate users/get user info
	timelineService TimelineService           // Interface dependency
	notificationSvc NotificationService       // Interface dependency
	schedulerSvc    SchedulerService          // Interface dependency for commands
	logger          *slog.Logger
	mentionRe       *regexp.Regexp // Store compiled regex
}

// Assert that chatService implements ChatService interface
var _ ChatService = (*chatService)(nil)

// NewChatService creates a new instance of chatService.
func NewChatService(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	userRepo repository.AuthRepository,
	timelineService TimelineService,
	notificationSvc NotificationService,
	schedulerSvc SchedulerService, // Inject scheduler
	logger *slog.Logger,
) ChatService {
	mentionRe := regexp.MustCompile(`@([a-zA-Z0-9_]+)`) // Compile regex once
	return &chatService{
		convRepo:        convRepo,
		msgRepo:         msgRepo,
		userRepo:        userRepo,
		timelineService: timelineService,
		notificationSvc: notificationSvc,
		schedulerSvc:    schedulerSvc, // Store scheduler
		logger:          logger.With(slog.String("service", "ChatService")),
		mentionRe:       mentionRe,
	}
}

// --- Conversation Management ---

func (s *chatService) CreateConversation(ctx context.Context, conversation *dbmodel.Conversation, participantIDs []uuid.UUID) (*dbmodel.Conversation, error) {
	l := s.logger.With(slog.String("method", "CreateConversation"))
	l.Debug("Attempting to create conversation", slog.String("type", conversation.Type))

	// 1. Validate input (basic) - More specific validation added
	if conversation.Type == "" {
		return nil, errors.New("conversation type cannot be empty")
	}
	if len(participantIDs) < 1 { // Must have at least one participant initially
		return nil, errors.New("at least one participant ID is required")
	}
	if conversation.Type == string(dbmodel.ConversationTypeGroup) && (conversation.Name == nil || *conversation.Name == "") {
		return nil, errors.New("group conversation requires a name")
	}
	if conversation.Type == string(dbmodel.ConversationTypeClient) && conversation.ClientID == nil {
		return nil, errors.New("client conversation requires a client ID")
	}
	if conversation.Type == string(dbmodel.ConversationTypeDirect) && len(participantIDs) != 2 {
		return nil, errors.New("direct message conversation requires exactly two participants")
	}


	// 2. Validate participant IDs & Check for duplicates
	uniqueParticipantIDs := make(map[uuid.UUID]struct{})
	validParticipantIDs := make([]uuid.UUID, 0, len(participantIDs))
	for _, id := range participantIDs {
		if _, exists := uniqueParticipantIDs[id]; !exists {
			_, err := s.userRepo.GetUserByID(ctx, id) // Use userRepo now
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					l.WarnContext(ctx, "Participant user ID not found, skipping", slog.String("userID", id.String()))
					continue // Skip non-existent users
				}
				l.ErrorContext(ctx, "Failed to validate participant user ID", slog.String("userID", id.String()), slog.String("error", err.Error()))
				return nil, fmt.Errorf("failed to validate participant %s: %w", id, err)
			}
			uniqueParticipantIDs[id] = struct{}{}
			validParticipantIDs = append(validParticipantIDs, id)
		}
	}

	if len(validParticipantIDs) < 1 { // Check after validation
		return nil, errors.New("at least one valid participant is required")
	}
	if conversation.Type == string(dbmodel.ConversationTypeDirect) && len(validParticipantIDs) != 2 {
		return nil, errors.New("direct message requires exactly two valid participants")
	}

	// 3. Create Conversation in DB
	if err := s.convRepo.CreateConversation(ctx, conversation); err != nil {
		l.ErrorContext(ctx, "Failed to create conversation in repository", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error creating conversation: %w", err)
	}
	l.InfoContext(ctx, "Conversation created", slog.String("conversationID", conversation.ID.String()))

	// 4. Add Participants
	for _, userID := range validParticipantIDs {
		participant := &dbmodel.ConversationParticipant{
			ConversationID: conversation.ID,
			UserID:         userID,
			// JoinedAt is set by default in DB
		}
		if err := s.convRepo.CreateConversationParticipant(ctx, participant); err != nil {
			l.ErrorContext(ctx, "Failed to add participant", slog.String("conversationID", conversation.ID.String()), slog.String("userID", userID.String()), slog.String("error", err.Error()))
			// Attempt to roll back? More complex transaction needed for full rollback.
			// For now, log error and continue, conversation exists but might miss participants.
			// Consider returning partial success or specific error.
			continue
		}
	}

	// 5. Create Timeline Event (optional, but good for tracking)
	timelineEvent := &dbmodel.TimelineEvent{
		UserID:      validParticipantIDs[0], // TODO: Decide who triggers the event (creator?) - Requires creator info passed in
		ClientID:    conversation.ClientID,
		EventType:   "CONVERSATION_CREATED",
		Description: fmt.Sprintf("Conversation '%s' created", conversation.Name), // Use name if available
		RelatedID:   &conversation.ID,
		RelatedType: util.StringPtr("conversation"),
		EventTime:   time.Now(),
	}
	if _, err := s.timelineService.CreateTimelineEvent(ctx, timelineEvent); err != nil {
		l.WarnContext(ctx, "Failed to create timeline event for conversation creation", slog.String("conversationID", conversation.ID.String()), slog.String("error", err.Error()))
		// Non-fatal error
	}

	// 6. Notify Participants (optional)
	// s.notificationSvc.BroadcastConversationUpdate(...) // Needs implementation

	// 7. Return the created conversation (potentially preloading participants if needed by caller)
	// Fetch again to ensure all participants are loaded correctly after potential partial failures
	createdConv, err := s.convRepo.GetConversationByID(ctx, conversation.ID)
	if err != nil {
        l.ErrorContext(ctx, "Failed to fetch newly created conversation with participants", slog.String("conversationID", conversation.ID.String()), slog.String("error", err.Error()))
        return conversation, nil // Return original conversation data as fallback
	}

	return createdConv, nil
}

func (s *chatService) GetConversation(ctx context.Context, conversationID uuid.UUID) (*dbmodel.Conversation, error) {
	l := s.logger.With(slog.String("method", "GetConversation"), slog.String("conversationID", conversationID.String()))
	conv, err := s.convRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.WarnContext(ctx, "Conversation not found")
			return nil, repository.ErrNotFound // Propagate not found error
		}
		l.ErrorContext(ctx, "Failed to get conversation", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error getting conversation: %w", err)
	}
	return conv, nil
}

func (s *chatService) ListConversationsForUser(ctx context.Context, userID uuid.UUID) ([]dbmodel.Conversation, error) {
	l := s.logger.With(slog.String("method", "ListConversationsForUser"), slog.String("userID", userID.String()))
	convs, err := s.convRepo.GetConversationsByUser(ctx, userID)
	if err != nil {
		l.ErrorContext(ctx, "Failed to list conversations for user", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error listing conversations: %w", err)
	}
	l.DebugContext(ctx, "Successfully listed conversations", slog.Int("count", len(convs)))
	return convs, nil
}

func (s *chatService) AddParticipant(ctx context.Context, conversationID, userID uuid.UUID) error {
	l := s.logger.With(slog.String("method", "AddParticipant"), slog.String("conversationID", conversationID.String()), slog.String("userID", userID.String()))

	// 1. Validate conversation exists
	_, err := s.convRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		l.WarnContext(ctx, "Conversation not found", slog.String("error", err.Error()))
		return fmt.Errorf("conversation not found: %w", err)
	}
	// 2. Validate user exists
	_, err = s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		l.WarnContext(ctx, "User not found", slog.String("error", err.Error()))
		return fmt.Errorf("user not found: %w", err)
	}

	// 3. Add participant
	participant := &dbmodel.ConversationParticipant{
		ConversationID: conversationID,
		UserID:         userID,
	}
	if err := s.convRepo.CreateConversationParticipant(ctx, participant); err != nil {
		l.ErrorContext(ctx, "Failed to add participant", slog.String("error", err.Error()))
		// Handle potential duplicate entry errors if not using OnConflict in repo
		return fmt.Errorf("database error adding participant: %w", err)
	}

	l.InfoContext(ctx, "Participant added to conversation")

	// 4. Create Timeline Event (optional)
	// ... s.timelineService.CreateTimelineEvent(...) ...

	// 5. Notify Conversation (optional)
	// ... s.notificationSvc.BroadcastConversationUpdate(...) ...

	return nil
}

func (s *chatService) RemoveParticipant(ctx context.Context, conversationID, userID uuid.UUID) error {
	l := s.logger.With(slog.String("method", "RemoveParticipant"), slog.String("conversationID", conversationID.String()), slog.String("userID", userID.String()))

	if err := s.convRepo.RemoveConversationParticipant(ctx, conversationID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.WarnContext(ctx, "Participant or conversation not found")
			return repository.ErrNotFound // Propagate not found
		}
		l.ErrorContext(ctx, "Failed to remove participant", slog.String("error", err.Error()))
		return fmt.Errorf("database error removing participant: %w", err)
	}

	l.InfoContext(ctx, "Participant removed from conversation")

	// Create Timeline Event (optional)
	// Notify Conversation (optional)

	return nil
}

// --- Message Handling ---

func (s *chatService) GetMessage(ctx context.Context, messageID uuid.UUID) (*dbmodel.Message, error) {
	l := s.logger.With(slog.String("method", "GetMessage"), slog.String("messageID", messageID.String()))
	msg, err := s.msgRepo.GetMessageByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			l.WarnContext(ctx, "Message not found")
			return nil, repository.ErrNotFound
		}
		l.ErrorContext(ctx, "Failed to get message", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error getting message: %w", err)
	}
	return msg, nil
}

func (s *chatService) ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID /*, pagination... */) ([]dbmodel.Message, error) {
	l := s.logger.With(slog.String("method", "ListMessagesByConversation"), slog.String("conversationID", conversationID.String()))
	// TODO: Add permission check: Does the requesting user belong to this conversation?

	msgs, err := s.msgRepo.GetMessagesByConversationID(ctx, conversationID /*, limit, offset */)
	if err != nil {
		l.ErrorContext(ctx, "Failed to list messages", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error listing messages: %w", err)
	}
	l.DebugContext(ctx, "Successfully listed messages", slog.Int("count", len(msgs)))
	return msgs, nil
}

func (s *chatService) SendMessage(ctx context.Context, conversationID, senderID uuid.UUID, content string) (*dbmodel.Message, error) {
	l := s.logger.With(slog.String("method", "SendMessage"), slog.String("conversationID", conversationID.String()), slog.String("senderID", senderID.String()))

	// 1. TODO: Validate sender is part of the conversation

	// 2. Check for and process commands
	if strings.HasPrefix(content, "/") {
		l.InfoContext(ctx, "Processing command")
		return s.processCommand(ctx, conversationID, senderID, content)
	}

	// 3. Handle regular text message
	l.DebugContext(ctx, "Processing regular text message")
	message := &dbmodel.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		Type:           dbmodel.MessageTypeText,
	}

	if err := s.msgRepo.CreateMessage(ctx, message); err != nil {
		l.ErrorContext(ctx, "Failed to create text message", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error creating message: %w", err)
	}
	l.InfoContext(ctx, "Text message created", slog.String("messageID", message.ID.String()))

	// 4. Trigger Timeline Event
	timelineEvent := &dbmodel.TimelineEvent{
		UserID:      senderID,
		ClientID:    nil, // TODO: Get ClientID from conversation if applicable
		EventType:   string(dbmodel.InteractionTypeChatMessage),
		Description: fmt.Sprintf("Sent message in conversation %s", conversationID),
		RelatedID:   &message.ID,
		RelatedType: util.StringPtr("message"),
		EventTime:   message.CreatedAt, // Use message creation time
	}
	// Fetch conversation to get client ID if needed for timeline
	conv, err := s.convRepo.GetConversationByID(ctx, conversationID)
	if err == nil && conv.ClientID != nil {
		timelineEvent.ClientID = conv.ClientID
	} else if err != nil {
         l.WarnContext(ctx, "Failed to get conversation for timeline event", slog.String("error", err.Error()))
    }

	if _, err := s.timelineService.CreateTimelineEvent(ctx, timelineEvent); err != nil {
		l.WarnContext(ctx, "Failed to create timeline event for sent message", slog.String("messageID", message.ID.String()), slog.String("error", err.Error()))
		// Non-fatal error
	}

	// 5. Broadcast via Notification Service
	if err := s.notificationSvc.BroadcastNewMessage(ctx, conversationID, message); err != nil {
		l.WarnContext(ctx, "Failed to broadcast new message", slog.String("messageID", message.ID.String()), slog.String("error", err.Error()))
		// Non-fatal error
	}

	return message, nil
}

// processCommand is an internal helper to handle slash commands.
func (s *chatService) processCommand(ctx context.Context, conversationID, senderID uuid.UUID, rawContent string) (*dbmodel.Message, error) {
	l := s.logger.With(slog.String("method", "processCommand"), slog.String("conversationID", conversationID.String()), slog.String("senderID", senderID.String()))
	parts := strings.Fields(rawContent)
	if len(parts) == 0 {
		return nil, errors.New("empty command")
	}
	command := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
	args := parts[1:]

	var outputMessage *dbmodel.Message // Message to potentially send back to chat

	switch command {
	case "reminder", "remind":
		// Example: /remind @user 1h Check report
		// Example: /remind me 10m Call back client X
		targetUserID := senderID // Default to self
		timeArgIndex := 0

		if len(args) > 0 && strings.HasPrefix(args[0], "@") {
			// Mentioning another user
			mentionedUsername := strings.TrimPrefix(args[0], "@")
			targetUser, err := s.userRepo.GetUserByEmail(ctx, mentionedUsername+"@example.com") // HACK: Assume domain, fix this!
			if err != nil {
				errMsg := fmt.Sprintf("User '%s' not found.", mentionedUsername)
				outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, errMsg, nil)
				l.WarnContext(ctx, errMsg)
				break // Exit switch
			}
			targetUserID = targetUser.ID
			timeArgIndex = 1 // Time argument is next
		} else if len(args) > 0 && strings.ToLower(args[0]) == "me" {
			// Explicitly reminding self
			targetUserID = senderID
			timeArgIndex = 1
		}

		if len(args) <= timeArgIndex {
			outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, "Usage: /remind [@user|me] <when> <message>", nil)
			l.WarnContext(ctx, "Invalid reminder command usage: missing time/message")
			break
		}

		// Parse time duration (simple example, needs robust parsing)
		durationStr := args[timeArgIndex]
		duration, err := time.ParseDuration(durationStr) // e.g., "1h", "10m", "30s"
		if err != nil {
			outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, fmt.Sprintf("Invalid time format: '%s'. Use format like '1h', '30m'.", durationStr), nil)
			l.WarnContext(ctx, "Invalid time format in reminder", slog.String("input", durationStr))
			break
		}
		remindAt := time.Now().Add(duration)

		// Get reminder message content
		reminderContent := strings.Join(args[timeArgIndex+1:], " ")
		if reminderContent == "" {
			outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, "Reminder message cannot be empty.", nil)
			l.WarnContext(ctx, "Empty reminder message")
			break
		}

		// Schedule the reminder using the Scheduler Service
		err = s.schedulerSvc.ScheduleReminder(ctx, remindAt, targetUserID, reminderContent, conversationID)
		if err != nil {
			l.ErrorContext(ctx, "Failed to schedule reminder", slog.String("error", err.Error()))
			outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, "Error setting reminder.", nil)
			break
		}

		// Create confirmation message
		confirmation := fmt.Sprintf("Reminder set for %s at %s.", args[0], remindAt.Format(time.Kitchen))
		if targetUserID != senderID {
			confirmation = fmt.Sprintf("Reminder set for user %s at %s.", args[0], remindAt.Format(time.Kitchen))
		}
		outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, confirmation, map[string]interface{}{
			"targetUserID": targetUserID.String(),
			"remindAt":     remindAt,
			"content":      reminderContent,
		})
		l.InfoContext(ctx, "Reminder scheduled successfully")

	// case "task":
	// TODO: Implement task command parsing and scheduling/creation logic

	default:
		outputMessage = s.createCommandOutputMessage(conversationID, senderID, dbmodel.MessageTypeCommandOutput, fmt.Sprintf("Unknown command: '%s'", command), nil)
		l.WarnContext(ctx, "Unknown command received", slog.String("command", command))
	}

	// Save and broadcast the output message if one was generated
	if outputMessage != nil {
		if err := s.msgRepo.CreateMessage(ctx, outputMessage); err != nil {
			l.ErrorContext(ctx, "Failed to save command output message", slog.String("error", err.Error()))
			return nil, fmt.Errorf("failed to save command output: %w", err) // Return error if saving output fails
		}
		l.DebugContext(ctx, "Command output message saved", slog.String("messageID", outputMessage.ID.String()))

		// Create Timeline Event for the command output
        // Timeline event logic might differ based on command success/failure
        // ...

		// Broadcast the output message
		if err := s.notificationSvc.BroadcastNewMessage(ctx, conversationID, outputMessage); err != nil {
			l.WarnContext(ctx, "Failed to broadcast command output message", slog.String("messageID", outputMessage.ID.String()), slog.String("error", err.Error()))
		}
		return outputMessage, nil // Return the output message
	}

	return nil, errors.New("command processed but generated no output") // Or return nil, nil if no output is expected
}

// createCommandOutputMessage is a helper to create a message struct for command responses.
func (s *chatService) createCommandOutputMessage(convID, senderID uuid.UUID, msgType dbmodel.MessageType, content string, metadata map[string]interface{}) *dbmodel.Message {
	msg := &dbmodel.Message{
		ConversationID: convID,
		SenderID:       senderID, // Or a dedicated System User ID? For now, use sender.
		Content:        content,
		Type:           msgType,
	}
	if metadata != nil {
		// Marshal metadata map to JSON for storage
		metaJSON, err := json.Marshal(metadata)
		if err == nil {
			msg.Metadata = datatypes.JSON(metaJSON)
		} else {
			s.logger.ErrorContext(context.Background(), "Failed to marshal command output metadata", slog.String("error", err.Error()))
            // Store raw content in metadata as fallback? Or ignore?
            metaStr := fmt.Sprintf("%+v", metadata)
            msg.Metadata = datatypes.JSON(fmt.Sprintf(`{"raw":"%s"}`, metaStr))
		}
	}
	return msg
}


// internal/service/chat_service.go (Corrected MarkMessageAsRead)

func (s *chatService) MarkMessageAsRead(ctx context.Context, messageID, userID uuid.UUID) error {
	l := s.logger.With(slog.String("method", "MarkMessageAsRead"), slog.String("messageID", messageID.String()), slog.String("userID", userID.String()))

	// 1. TODO: Add permission check: Does the userID belong to the message's conversation?
	//    (Requires fetching message/conversation first)
    //    msg, err := s.msgRepo.GetMessageByID(ctx, messageID)
    //    if err != nil { ... handle error ... }
    //    isParticipant := /* check if userID is in msg.Conversation.Participants */
    //    if !isParticipant { return errors.New("user is not part of this conversation")}


	// 2. Check if already read using msgRepo
	_, err := s.msgRepo.GetMessageReadReceipt(ctx, messageID, userID) // CORRECTED: Use s.msgRepo
	if err == nil {
		l.DebugContext(ctx, "Message already marked as read by user")
		return nil // Already read, no error
	} else if !errors.Is(err, repository.ErrNotFound) {
		l.ErrorContext(ctx, "Failed to check existing read receipt", slog.String("error", err.Error()))
		return fmt.Errorf("database error checking read receipt: %w", err)
	}

	// 3. Create Read Receipt using msgRepo
	receipt := &dbmodel.MessageReadReceipt{
		MessageID: messageID,
		UserID:    userID,
		// ReadAt handled by DB default
	}
	if err := s.msgRepo.CreateMessageReadReceipt(ctx, receipt); err != nil { // CORRECTED: Use s.msgRepo
		l.ErrorContext(ctx, "Failed to create read receipt", slog.String("error", err.Error()))
		return fmt.Errorf("database error creating read receipt: %w", err)
	}
	l.InfoContext(ctx, "Message marked as read")

	// 4. Broadcast notification (optional)
    message, err := s.msgRepo.GetMessageByID(ctx, messageID) // Fetch message to get ConversationID
    if err != nil {
        l.WarnContext(ctx, "Failed to get message for broadcasting read status", slog.String("error", err.Error()))
        // Continue even if broadcast fails? Or return error? Depends on requirements.
    } else {
        payload := map[string]interface{}{
            "messageId": messageID.String(),
            "userId":    userID.String(),
            "readAt":    receipt.ReadAt, // Include timestamp if needed by frontend
        }
        if broadcastErr := s.notificationSvc.BroadcastMessageRead(ctx, message.ConversationID, payload); broadcastErr != nil { // Assuming BroadcastMessageRead exists
            l.WarnContext(ctx, "Failed to broadcast message read status", slog.String("error", broadcastErr.Error()))
        }
    }


	return nil
}

// Need to add BroadcastMessageRead to NotificationService interface and implementation
// Example interface update:
// type NotificationService interface {
// 	 BroadcastNewMessage(ctx context.Context, conversationID uuid.UUID, message *dbmodel.Message) error
// 	 BroadcastMessageRead(ctx context.Context, conversationID uuid.UUID, payload map[string]interface{}) error // Added
// }
