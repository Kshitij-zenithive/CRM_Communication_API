// internal/service/template.go (NEW FILE)
package service

import (
	"bytes"
	"context"
	"crm-communication-api/internal/model"
	"crm-communication-api/internal/repository"
	"fmt"
	htmlTemplate "html/template" // Use aliases to avoid naming conflicts
	"log"
	textTemplate "text/template"

	"github.com/google/uuid"
)

type TemplateService struct {
	repo *repository.Repository
}

func NewTemplateService(repo *repository.Repository) *TemplateService {
	return &TemplateService{repo: repo}
}

// CreateEmailTemplate creates a new email template.
func (ts *TemplateService) CreateEmailTemplate(ctx context.Context, input model.EmailTemplate) (*model.EmailTemplate, error) {

	if err := ts.repo.CreateEmailTemplate(ctx, &input); err != nil {
		return nil, fmt.Errorf("failed to create email template: %w", err)
	}
	return &input, nil
}

// GetEmailTemplate retrieves an email template by ID.
func (ts *TemplateService) GetEmailTemplate(ctx context.Context, id uuid.UUID) (*model.EmailTemplate, error) {
	return ts.repo.GetEmailTemplate(ctx, id)
}

// GetAllEmailTemplates retrieves all email templates.
func (ts *TemplateService) GetAllEmailTemplates(ctx context.Context) ([]model.EmailTemplate, error) {
	return ts.repo.GetAllEmailTemplates(ctx)
}

// UpdateEmailTemplate updates an existing email template.
func (ts *TemplateService) UpdateEmailTemplate(ctx context.Context, id uuid.UUID, input model.EmailTemplate) (*model.EmailTemplate, error) {
	template, err := ts.repo.GetEmailTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	template.Name = input.Name
	template.Subject = input.Subject
	template.Body = input.Body

	if err := ts.repo.UpdateEmailTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to update email template: %w", err)
	}
	return template, nil
}

// DeleteEmailTemplate deletes an email template.
func (ts *TemplateService) DeleteEmailTemplate(ctx context.Context, id uuid.UUID) error {
	return ts.repo.DeleteEmailTemplate(ctx, id)
}

// RenderTemplate renders an email template with the provided data.
func (ts *TemplateService) RenderTemplate(template *model.EmailTemplate, data map[string]interface{}) (subject string, body string, err error) {
	// Render subject
	subjectTmpl, err := textTemplate.New("subject").Parse(template.Subject)
	if err != nil {
		log.Printf("Error parsing subject template: %v", err) // Log and continue
		return "", "", fmt.Errorf("error parsing subject template: %w", err)
	}
	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		log.Printf("Error executing subject template: %v", err)
		return "", "", fmt.Errorf("error executing subject template: %w", err)
	}
	subject = subjectBuf.String()

	// Render body (HTML template)
	bodyTmpl, err := htmlTemplate.New("body").Parse(template.Body) // Use html/template
	if err != nil {
		log.Printf("Error parsing body template: %v", err)
		return "", "", fmt.Errorf("error parsing body template: %w", err)
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		log.Printf("Error executing body template: %v", err)
		return "", "", fmt.Errorf("error executing body template: %w", err)
	}
	body = bodyBuf.String()

	return subject, body, nil
}
