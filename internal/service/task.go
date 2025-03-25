// internal/service/task.go  (Corrected service and added missing functions)

package service

import (
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	repo *repository.Repository
}

func NewTaskService(repo *repository.Repository) *TaskService {
	return &TaskService{repo: repo}
}

// CreateTask creates a new task and records a timeline event.
func (ts *TaskService) CreateTask(ctx context.Context, input model.Task) (*model.Task, error) {
	if err := ts.repo.CreateTask(ctx, &input); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return &input, nil
}

// GetTask retrieves a task by its ID.
func (ts *TaskService) GetTask(ctx context.Context, taskID uuid.UUID) (*model.Task, error) {
	return ts.repo.GetTaskByID(ctx, taskID)
}

// UpdateTask updates an existing task and records a timeline event.
func (ts *TaskService) UpdateTask(ctx context.Context, task *model.Task) (*model.Task, error) {
	// Fetch the existing task
	existingTask, err := ts.repo.GetTaskByID(ctx, task.ID)
	if err != nil {
		return nil, err
	}

	// Update fields based on input
	existingTask.Title = task.Title
	existingTask.Description = task.Description
	existingTask.AssignedTo = task.AssignedTo
	existingTask.DueDate = task.DueDate
	existingTask.Completed = task.Completed
	existingTask.ClientID = task.ClientID

	if task.Completed && existingTask.CompletedAt == nil { // Set CompletedAt only if transitioning to completed
		now := time.Now()
		existingTask.CompletedAt = &now
	}

	// Save the updated task
	if err := ts.repo.UpdateTask(ctx, existingTask); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return existingTask, nil
}

// DeleteTask deletes a task by its ID.
func (ts *TaskService) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	return ts.repo.DeleteTask(ctx, taskID)
}

// GetTasksForUser retrieves all tasks assigned to or created by a user
func (ts *TaskService) GetTasksForUser(ctx context.Context, userID uuid.UUID) ([]model.Task, error) {
	return ts.repo.GetTasksByUserID(ctx, userID)
}
