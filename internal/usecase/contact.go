package usecase

import (
	"context"
	"fmt"
	"go-recruitment-backend/internal/domain"
	"go-recruitment-backend/pkg/email"
	"strings"
)

type contactUsecase struct {
	emailService *email.EmailService
}

// NewContactUsecase creates a new contact usecase
func NewContactUsecase(emailService *email.EmailService) domain.ContactUsecase {
	return &contactUsecase{
		emailService: emailService,
	}
}

// SendContactMessage validates the contact request and sends the email
func (uc *contactUsecase) SendContactMessage(ctx context.Context, req *domain.ContactRequest) error {
	// Validate input (additional validation beyond binding)
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Email) == "" {
		return fmt.Errorf("email is required")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return fmt.Errorf("subject is required")
	}
	if strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("message is required")
	}

	// Check if email service is configured
	if !uc.emailService.IsConfigured() {
		return fmt.Errorf("email service is not configured")
	}

	// Prepare email data
	emailData := email.ContactEmailData{
		SenderName:  strings.TrimSpace(req.Name),
		SenderEmail: strings.TrimSpace(req.Email),
		Subject:     strings.TrimSpace(req.Subject),
		Message:     strings.TrimSpace(req.Message),
	}

	// Send the email
	if err := uc.emailService.SendContactEmail(emailData); err != nil {
		return fmt.Errorf("failed to send contact email: %w", err)
	}

	return nil
}
