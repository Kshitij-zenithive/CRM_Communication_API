// internal/service/notification_service.go
package service

import (
	"context"
	dbmodel "crm-communication-api/internal/model"
	websocket "crm-communication-api/internal/websocket" // Import websocket package
	"fmt"
	"log/slog"
	"sync" // Needed for the simple broadcaster mutex

	"github.com/google/uuid"
)

// Define the message channel type used by the broadcaster
type messageChan chan *dbmodel.Message

// Simple in-memory broadcaster (Replace with Redis Pub/Sub for scaling)
type inMemoryBroadcaster struct {
	messageSubscribers map[uuid.UUID][]messageChan // Map conversationID to listening channels
	mu                 sync.RWMutex
	logger             *slog.Logger
}

func newInMemoryBroadcaster(logger *slog.Logger) *inMemoryBroadcaster {
	return &inMemoryBroadcaster{
		messageSubscribers: make(map[uuid.UUID][]messageChan),
		logger:             logger.With(slog.String("component", "InMemoryBroadcaster")),
	}
}

// SubscribeMessages creates a channel and registers it for a conversation.
func (b *inMemoryBroadcaster) SubscribeMessages(ctx context.Context, conversationID uuid.UUID) (<-chan *dbmodel.Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(messageChan, 1) // Buffered channel
	b.messageSubscribers[conversationID] = append(b.messageSubscribers[conversationID], ch)
	b.logger.Debug("Channel subscribed", slog.String("conversationID", conversationID.String()), slog.Int("current_subs", len(b.messageSubscribers[conversationID])))

	// Handle client disconnect using context cancellation
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()

		subs := b.messageSubscribers[conversationID]
		for i, subCh := range subs {
			if subCh == ch {
				// Remove channel from slice
				b.messageSubscribers[conversationID] = append(subs[:i], subs[i+1:]...)
				close(ch) // Close the channel
				b.logger.Debug("Channel unsubscribed due to context done", slog.String("conversationID", conversationID.String()), slog.Int("remaining_subs", len(b.messageSubscribers[conversationID])))
				break
			}
		}
		// Clean up map entry if no subscribers left
		if len(b.messageSubscribers[conversationID]) == 0 {
			delete(b.messageSubscribers, conversationID)
			b.logger.Debug("Cleared subscriber map entry for conversation", slog.String("conversationID", conversationID.String()))
		}
	}()

	return ch, nil
}

// broadcast sends a generic websocket message to subscribers of a specific room (conversation).
func (b *inMemoryBroadcaster) broadcast(conversationID uuid.UUID, message websocket.Message) {
	b.mu.RLock()
	subscribers := b.messageSubscribers[conversationID]
	// Create a copy of the subscriber list to avoid holding lock during sends
	subsCopy := make([]messageChan, len(subscribers))
	copy(subsCopy, subscribers)
	b.mu.RUnlock()

	b.logger.Debug("Broadcasting message",
		slog.String("conversationID", conversationID.String()),
		slog.String("messageType", message.Type),
		slog.Int("subscriber_count", len(subsCopy)))

	for _, ch := range subsCopy {
		// Non-blocking send
		select {
		case ch <- message.Payload.(*dbmodel.Message): // Assuming payload is *dbmodel.Message for new_message
			// TODO: Handle different payload types based on message.Type
			// Need type assertion or a different channel type if payload varies
		default:
			// Subscriber channel is full or closed, maybe log this?
			b.logger.Warn("Failed to send to subscriber channel (full or closed)", slog.String("conversationID", conversationID.String()))
		}
	}
}

// broadcastGeneric sends any payload type to a different kind of channel if needed
// This is more complex and might require rethinking the channel/subscription strategy
// For now, let's assume the main channel handles *dbmodel.Message for simplicity
// func (b *inMemoryBroadcaster) broadcastGeneric(conversationID uuid.UUID, message websocket.Message) { ... }

// --- notificationService Implementation ---

type notificationService struct {
	hub         *websocket.Hub       // Direct Hub reference for generic broadcast
	broadcaster *inMemoryBroadcaster // Specific broadcaster for typed channels (optional)
	logger      *slog.Logger
}

// Assert implements interface
var _ NotificationService = (*notificationService)(nil)

// NewNotificationService constructor
func NewNotificationService(hub *websocket.Hub, logger *slog.Logger) NotificationService {
	// Create broadcaster if using typed channels
	// broadcaster := newInMemoryBroadcaster(logger)
	return &notificationService{
		hub: hub, // Store hub
		// broadcaster: broadcaster,
		logger: logger.With(slog.String("service", "NotificationService")),
	}
}

// BroadcastNewMessage sends new messages via the Hub
func (s *notificationService) BroadcastNewMessage(ctx context.Context, conversationID uuid.UUID, message *dbmodel.Message) error {
	l := s.logger.With(slog.String("method", "BroadcastNewMessage"), slog.String("conversationID", conversationID.String()), slog.String("messageID", message.ID.String()))
	l.Debug("Broadcasting new message event")

	wsMsg := websocket.Message{
		Type:    "new_message",
		RoomID:  conversationID.String(),
		Payload: message, // Send the full internal message model
	}

	// Send to the hub's general broadcast channel
	// The hub itself will route it to clients in the correct room (RoomID)
	select {
	case s.hub.Broadcast <- wsMsg:
		l.Debug("Message sent to hub broadcast channel")
	default:
		// Hub broadcast channel might be full, this indicates a bottleneck
		l.Warn("Hub broadcast channel full, message potentially dropped")
		// Consider adding metrics or alternative handling
	}

	// If using the specific broadcaster for typed channels (optional):
	// s.broadcaster.broadcast(conversationID, wsMsg) // Assuming broadcaster handles this type

	return nil
}

// BroadcastMessageRead sends message read status updates via the Hub
func (s *notificationService) BroadcastMessageRead(ctx context.Context, conversationID uuid.UUID, payload map[string]interface{}) error {
	l := s.logger.With(slog.String("method", "BroadcastMessageRead"), slog.String("conversationID", conversationID.String()))
	l.Debug("Broadcasting message read event")

	wsMsg := websocket.Message{
		Type:    "message_read",
		RoomID:  conversationID.String(),
		Payload: payload, // Send the map payload directly
	}

	select {
	case s.hub.Broadcast <- wsMsg:
		l.Debug("Message read event sent to hub broadcast channel")
	default:
		l.Warn("Hub broadcast channel full, message read event potentially dropped")
	}

	// If using a specific broadcaster (less likely for generic map payload):
	// s.broadcaster.broadcastGeneric(conversationID, wsMsg) // Would need a different channel type

	return nil
}

// GetNewMessageChannel provides the channel for GraphQL subscriptions
func (s *notificationService) GetNewMessageChannel(ctx context.Context, conversationID uuid.UUID) (<-chan *dbmodel.Message, error) {
	s.logger.Debug("Subscription request received", slog.String("conversationID", conversationID.String()))
	// This should ideally use the broadcaster's subscribe method if implemented
	// return s.broadcaster.SubscribeMessages(ctx, conversationID)

	// Placeholder returning an error until broadcaster is fully integrated or
	// hub provides direct subscription channels
	return nil, fmt.Errorf("typed subscription channel mechanism not fully implemented")
}
