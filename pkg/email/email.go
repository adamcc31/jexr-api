package email

import (
	"bytes"
	"fmt"
	"go-recruitment-backend/config"
	"html/template"
	"net/smtp"
)

// EmailService handles sending emails via SMTP
type EmailService struct {
	host      string
	port      string
	username  string
	password  string
	fromEmail string
	toEmail   string
}

// ContactEmailData holds the data for contact form emails
type ContactEmailData struct {
	SenderName  string
	SenderEmail string
	Subject     string
	Message     string
}

// NewEmailService creates a new email service with Brevo SMTP configuration
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		host:      cfg.SMTPHost,
		port:      cfg.SMTPPort,
		username:  cfg.SMTPUsername,
		password:  cfg.SMTPPassword,
		fromEmail: cfg.SMTPUsername, // Brevo uses login email as from address
		toEmail:   cfg.ContactEmailTo,
	}
}

// contactEmailTemplate is the HTML template for contact form emails
const contactEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>New Contact Form Submission</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #0066cc; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .field { margin-bottom: 15px; }
        .label { font-weight: bold; color: #555; }
        .value { margin-top: 5px; }
        .message-box { background: white; padding: 15px; border-left: 4px solid #0066cc; margin-top: 10px; }
        .footer { text-align: center; padding: 20px; color: #888; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>New Contact Form Submission</h1>
        </div>
        <div class="content">
            <div class="field">
                <div class="label">From:</div>
                <div class="value">{{.SenderName}} ({{.SenderEmail}})</div>
            </div>
            <div class="field">
                <div class="label">Subject:</div>
                <div class="value">{{.Subject}}</div>
            </div>
            <div class="field">
                <div class="label">Message:</div>
                <div class="message-box">{{.Message}}</div>
            </div>
        </div>
        <div class="footer">
            <p>This email was sent from the J Expert Recruitment contact form.</p>
            <p>To reply, send an email to: {{.SenderEmail}}</p>
        </div>
    </div>
</body>
</html>`

// SendContactEmail sends a contact form email to the configured recipient
func (s *EmailService) SendContactEmail(data ContactEmailData) error {
	// Parse and execute the template
	tmpl, err := template.New("contact").Parse(contactEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Build the email message
	subject := fmt.Sprintf("Contact Form: %s", data.Subject)

	// Construct MIME message
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Reply-To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s",
		s.fromEmail,
		s.toEmail,
		data.SenderEmail,
		subject,
		body.String(),
	))

	// Setup SMTP authentication
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// Send the email
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	err = smtp.SendMail(addr, auth, s.fromEmail, []string{s.toEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// IsConfigured checks if the email service has valid SMTP configuration
func (s *EmailService) IsConfigured() bool {
	return s.host != "" && s.username != "" && s.password != ""
}
