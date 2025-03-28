// internal/service/reminder.go (Corrected for consistency)

package service

import (
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ReminderService struct {
	repo *repository.Repository
}

func NewReminderService(repo *repository.Repository) *ReminderService {
	return &ReminderService{repo: repo}
}

// CreateReminder creates a new reminder.
func (s *ReminderService) CreateReminder(ctx context.Context, reminder *model.Reminder) (*model.Reminder, error) {
	if err := s.repo.CreateReminder(ctx, reminder); err != nil {
		return nil, fmt.Errorf("failed to create reminder: %w", err)
	}
	return reminder, nil
}

// GetReminder retrieves a reminder by its ID.
func (s *ReminderService) GetReminder(ctx context.Context, reminderID uuid.UUID) (*model.Reminder, error) {
	reminder, err := s.repo.GetReminderByID(ctx, reminderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminder: %w", err)
	}
	return reminder, nil
}

// UpdateReminder updates an existing reminder.
func (s *ReminderService) UpdateReminder(ctx context.Context, reminder *model.Reminder) (*model.Reminder, error) {
	// Check if reminder exist before
	existingReminder, err := s.repo.GetReminderByID(ctx, reminder.ID)
	if err != nil {
		return nil, err
	}

	existingReminder.Content = reminder.Content
	existingReminder.RemindAt = reminder.RemindAt
	existingReminder.Triggered = reminder.Triggered
	existingReminder.UpdatedAt = time.Now() // Update the UpdatedAt time

	if err := s.repo.UpdateReminder(ctx, existingReminder); err != nil {
		return nil, fmt.Errorf("failed to update reminder: %w", err)
	}
	return existingReminder, nil
}

// DeleteReminder deletes a reminder.
func (s *ReminderService) DeleteReminder(ctx context.Context, reminderID uuid.UUID) error {
	if err := s.repo.DeleteReminder(ctx, reminderID); err != nil {
		return fmt.Errorf("failed to delete reminder: %w", err)
	}
	return nil
}

// GetReminders retrieves all reminders for a user.
func (s *ReminderService) GetReminders(ctx context.Context, userID uuid.UUID) ([]model.Reminder, error) {
	reminders, err := s.repo.GetRemindersByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminders: %w", err)
	}
	return reminders, nil
}

// GetPendingReminders retrieves reminders that are due to be triggered
func (s *ReminderService) GetPendingReminders(ctx context.Context) ([]model.Reminder, error) {
	reminders, err := s.repo.GetPendingReminders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get reminders: %w", err)
	}
	return reminders, nil
}

// TriggerReminder send notification to user, and mark reminder as triggered
func (s *ReminderService) TriggerReminder(ctx context.Context, reminderID uuid.UUID) error {
	// Retrieve Reminder from db
	reminder, err := s.GetReminder(ctx, reminderID)
	if err != nil {
		return err
	}

	// Check if the reminder has already been triggered
	if reminder.Triggered {
		return fmt.Errorf("reminder with ID %s has already been triggered", reminderID)
	}

	// Update Trigger state, and last updated time
	reminder.Triggered = true
	reminder.UpdatedAt = time.Now()

	// Save reminder to db
	_, err = s.UpdateReminder(ctx, reminder)
	if err != nil {
		return err
	}

	// Implement your notification logic here, for now is just printing.
	fmt.Printf("Reminder for user %s: %s\n", reminder.UserID, reminder.Content)

	return nil
}
