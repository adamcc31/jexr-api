package domain

import "context"

// ContactRequest represents a contact form submission
type ContactRequest struct {
	Name    string `json:"name" binding:"required"`
	Email   string `json:"email" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// ContactUsecase defines the interface for contact form operations
type ContactUsecase interface {
	// SendContactMessage validates and sends a contact form message
	SendContactMessage(ctx context.Context, req *ContactRequest) error
}
