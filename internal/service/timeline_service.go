// internal/service/timeline_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	dbmodel "crm-communication-api/internal/model" // Alias for DB model
	"crm-communication-api/internal/repository"

	"github.com/google/uuid"
)

// timelineService implements the TimelineService interface.
type timelineService struct {
	repo   repository.TimelineRepository // Use the specific interface
	logger *slog.Logger
}

// Assert that timelineService implements TimelineService interface
var _ TimelineService = (*timelineService)(nil)

// NewTimelineService creates a new instance of timelineService.
func NewTimelineService(repo repository.TimelineRepository, logger *slog.Logger) TimelineService {
	return &timelineService{
		repo:   repo,
		logger: logger.With(slog.String("service", "TimelineService")),
	}
}

// CreateTimelineEvent validates and saves a new timeline event.
func (s *timelineService) CreateTimelineEvent(ctx context.Context, event *dbmodel.TimelineEvent) (*dbmodel.TimelineEvent, error) {
	l := s.logger.With(slog.String("method", "CreateTimelineEvent"), slog.String("eventType", event.EventType), slog.String("userID", event.UserID.String()))

	// --- Basic Validation ---
	if event.UserID == uuid.Nil {
		l.WarnContext(ctx, "CreateTimelineEvent validation failed: UserID is required")
		return nil, errors.New("user ID is required for timeline event")
	}
	if event.EventType == "" {
		l.WarnContext(ctx, "CreateTimelineEvent validation failed: EventType is required")
		return nil, errors.New("event type is required for timeline event")
	}
	if event.EventTime.IsZero() {
		l.WarnContext(ctx, "CreateTimelineEvent validation failed: EventTime is required")
		return nil, errors.New("event time is required for timeline event")
	}

	// --- Repository Call ---
	err := s.repo.CreateTimelineEvent(ctx, event)
	if err != nil {
		l.ErrorContext(ctx, "Failed to create timeline event in repository", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error creating timeline event: %w", err)
	}

	l.InfoContext(ctx, "Timeline event created successfully", slog.String("eventID", event.ID.String()))
	// The 'event' object passed in should now have its ID populated by GORM's Create method.
	return event, nil
}

// GetTimelineEvents retrieves timeline events based on optional filters.
func (s *timelineService) GetTimelineEvents(ctx context.Context, clientID *uuid.UUID, userID *uuid.UUID /*, pagination... */) ([]dbmodel.TimelineEvent, error) {
	l := s.logger.With(slog.String("method", "GetTimelineEvents"))
	if clientID != nil {
		l = l.With(slog.String("clientID", clientID.String()))
	}
	if userID != nil {
		l = l.With(slog.String("userID", userID.String()))
	}
	l.DebugContext(ctx, "Fetching timeline events")

	// --- Repository Call ---
	events, err := s.repo.GetTimelineEvents(ctx, clientID, userID)
	if err != nil {
		// No need to check for ErrNotFound here usually, repo call should just return empty slice if nothing found.
		// Log only unexpected errors.
		l.ErrorContext(ctx, "Failed to get timeline events from repository", slog.String("error", err.Error()))
		return nil, fmt.Errorf("database error retrieving timeline events: %w", err)
	}

	l.DebugContext(ctx, "Successfully fetched timeline events", slog.Int("count", len(events)))
	return events, nil
}
