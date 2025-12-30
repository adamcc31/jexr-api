package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"go-recruitment-backend/config"
	"html/template"
	"net"
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
		fromEmail: cfg.SMTPFromEmail, // Verified sender email, NOT the SMTP login
		toEmail:   cfg.ContactEmailTo,
	}
}

// contactEmailTemplate is the HTML template for contact form emails
const contactEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New Contact Form Submission</title>
    <style>
        /* Reset & Base Styles */
        body { margin: 0; padding: 0; font-family: 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333333; background-color: #f4f4f4; -webkit-text-size-adjust: 100%; -ms-text-size-adjust: 100%; }
        table { border-spacing: 0; width: 100%; }
        td { padding: 0; }
        img { border: 0; }

        /* Container */
        .wrapper { width: 100%; table-layout: fixed; background-color: #f4f4f4; padding-bottom: 40px; }
        .main-container { background-color: #ffffff; margin: 0 auto; max-width: 600px; width: 100%; box-shadow: 0 1px 3px rgba(0,0,0,0.1); border-radius: 4px; overflow: hidden; }

        /* Header */
        .header { background-color: #0066cc; color: #ffffff; padding: 25px 20px; text-align: center; }
        .header h1 { margin: 0; font-size: 24px; font-weight: 600; letter-spacing: 0.5px; }

        /* Content */
        .content { padding: 30px 25px; }
        .field-group { margin-bottom: 20px; }
        .label { font-size: 12px; text-transform: uppercase; letter-spacing: 1px; color: #888888; font-weight: bold; margin-bottom: 5px; }
        .value { font-size: 16px; color: #333333; font-weight: 500; }
        
        /* Message Box */
        .message-box { background-color: #f8f9fa; border-left: 4px solid #0066cc; padding: 15px; margin-top: 5px; font-style: italic; color: #555; white-space: pre-wrap; }

        /* Footer */
        .footer { background-color: #f4f4f4; padding: 20px; text-align: center; font-size: 12px; color: #999999; border-top: 1px solid #e1e1e1; }
        .footer p { margin: 5px 0; }
        .footer strong { color: #666; }
        .company-address { margin: 15px 0; line-height: 1.5; }
        .dev-credit { font-size: 11px; margin-top: 15px; opacity: 0.7; }
        
        /* Links */
        a { color: #0066cc; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .unsubscribe-link { color: #999999; text-decoration: underline; font-size: 11px; }

        /* Mobile Responsive */
        @media screen and (max-width: 600px) {
            .main-container { width: 100% !important; }
            .content { padding: 20px; }
        }
    </style>
</head>
<body>
    <div class="wrapper">
        <br> <div class="main-container">
            <div class="header">
                <h1>New Lead Received</h1>
            </div>

            <div class="content">
                <div class="field-group">
                    <div class="label">From</div>
                    <div class="value">{{.SenderName}} (<a href="mailto:{{.SenderEmail}}">{{.SenderEmail}}</a>)</div>
                </div>

                <div class="field-group">
                    <div class="label">Subject</div>
                    <div class="value">{{.Subject}}</div>
                </div>

                <div class="field-group">
                    <div class="label">Message Content</div>
                    <div class="value message-box">{{.Message}}</div>
                </div>

                <div style="margin-top: 30px; text-align: center;">
                    <a href="mailto:{{.SenderEmail}}?subject=Re: {{.Subject}}" style="background-color: #0066cc; color: white; padding: 12px 25px; text-decoration: none; border-radius: 4px; font-weight: bold; display: inline-block;">Reply to Sender</a>
                </div>
            </div>

            <div class="footer">
                <p>This is an automated notification from your website contact form.</p>
                
                <div class="company-address">
                    <strong>J Expert Recruitment - Part of Exata Group</strong><br>
                    Exata Office Tower, Ruko Rose Garden 5 No. 9<br>
                    Jakasetia, Bekasi Selatan, Kota Bekasi<br>
                    Jawa Barat, Indonesia 17148
                </div>

                <p>&copy; 2025 J Expert Recruitment. All rights reserved.</p>
                <p class="dev-credit">Develop by Noxx Labs</p>

                <p style="margin-top: 20px;">
                    <a href="#" class="unsubscribe-link">Unsubscribe</a> | 
                    <a href="#" class="unsubscribe-link">Privacy Policy</a>
                </p>
            </div>
        </div>
        <br> </div>
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

	// Send via STARTTLS (required by Brevo on port 587)
	err = s.sendMailWithStartTLS(msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// sendMailWithStartTLS sends email using STARTTLS which is required by Brevo
func (s *EmailService) sendMailWithStartTLS(msg []byte) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// Connect to SMTP server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Say hello
	if err = client.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: s.host,
	}
	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	// Authenticate using LOGIN auth (Brevo preference)
	auth := LoginAuth(s.username, s.password)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set sender
	if err = client.Mail(s.fromEmail); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Set recipient
	if err = client.Rcpt(s.toEmail); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Quit
	return client.Quit()
}

// loginAuth implements LOGIN authentication (required by some SMTP servers like Brevo)
type loginAuth struct {
	username, password string
}

// LoginAuth returns an Auth that implements the LOGIN authentication mechanism
func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		}
	}
	return nil, nil
}

// IsConfigured checks if the email service has valid SMTP configuration
func (s *EmailService) IsConfigured() bool {
	return s.host != "" && s.username != "" && s.password != "" && s.fromEmail != ""
}
