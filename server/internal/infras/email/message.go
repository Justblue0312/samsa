package email

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"
)

//go:embed all:templates
var templateFS embed.FS

// MessageBody represents the content of an email message.
type MessageBody interface {
	Content() (string, error)
	IsHTML() bool
}

// MessagePlain represents plain text email content.
type MessagePlain struct {
	Text string
}

// Content returns the plain text content.
func (m MessagePlain) Content() (string, error) {
	return m.Text, nil
}

// IsHTML returns false for plain text content.
func (m MessagePlain) IsHTML() bool {
	return false
}

// MessageHTML represents HTML email content.
type MessageHTML struct {
	Text string
}

// Content returns the HTML content.
func (m MessageHTML) Content() (string, error) {
	return m.Text, nil
}

// IsHTML returns true for HTML content.
func (m MessageHTML) IsHTML() bool {
	return true
}

// MessageTemplate represents template-based email content.
type MessageTemplate struct {
	Name string
	Data any
}

// Content executes the template and returns the generated content.
func (m MessageTemplate) Content() (string, error) {
	var bodyContent bytes.Buffer
	if err := m.executeTemplate(&bodyContent); err != nil {
		return "", err
	}
	return bodyContent.String(), nil
}

// IsHTML determines if the template is HTML based on file extension.
func (m MessageTemplate) IsHTML() bool {
	return strings.HasSuffix(strings.ToLower(m.Name), ".html")
}

// executeTemplate executes the template with the provided data.
func (m MessageTemplate) executeTemplate(w io.Writer) error {
	name := filepath.Base(m.Name)
	t := template.New(name)

	// Read the template file from the embedded filesystem
	templatePath := path.Join("templates", m.Name)
	templateContent, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template '%s': %w", m.Name, err)
	}

	// Parse the template content
	parsedTemplate, err := t.Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template '%s': %w", m.Name, err)
	}

	// Execute the template with the provided data
	if err := parsedTemplate.Execute(w, m.Data); err != nil {
		return fmt.Errorf("failed to execute template '%s': %w", m.Name, err)
	}

	return nil
}

// Message represents an email message to be sent.
type Message struct {
	To      []string // Required: recipient email addresses
	Cc      []string // Optional: carbon copy recipients
	Bcc     []string // Optional: blind carbon copy recipients
	Subject string   // Required: email subject
	ReplyTo string   // Optional: reply-to address

	// Body contains the email content (plain text, HTML, or template-based).
	Body MessageBody
}

// Validate checks that the message has all required fields.
func (m *Message) Validate() error {
	if len(m.To) == 0 {
		return errors.New("at least one recipient is required")
	}
	if m.Subject == "" {
		return errors.New("subject is required")
	}
	if m.Body == nil {
		return errors.New("body is required")
	}
	return nil
}

// Format creates an RFC 5322 formatted email message.
func (m *Message) Format(fromEmail, fromName string) (string, error) {
	var sb strings.Builder

	// From header
	if fromName != "" {
		fmt.Fprintf(&sb, "From: %s <%s>\r\n", fromName, fromEmail)
	} else {
		fmt.Fprintf(&sb, "From: %s\r\n", fromEmail)
	}

	// To, Cc, Reply-To, Subject, Date headers
	fmt.Fprintf(&sb, "To: %s\r\n", strings.Join(m.To, ", "))
	if len(m.Cc) > 0 {
		fmt.Fprintf(&sb, "Cc: %s\r\n", strings.Join(m.Cc, ", "))
	}
	if m.ReplyTo != "" {
		fmt.Fprintf(&sb, "Reply-To: %s\r\n", m.ReplyTo)
	}
	fmt.Fprintf(&sb, "Subject: %s\r\n", m.Subject)
	fmt.Fprintf(&sb, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))

	// MIME headers and body
	sb.WriteString("MIME-Version: 1.0\r\n")

	contentType := "text/plain"
	if m.Body.IsHTML() {
		contentType = "text/html"
	}
	fmt.Fprintf(&sb, "Content-Type: %s; charset=utf-8\r\n", contentType)
	sb.WriteString("\r\n")

	bodyContent, err := m.Body.Content()
	if err != nil {
		return "", err
	}

	sb.WriteString(bodyContent)

	return sb.String(), nil
}

// NewPlainMessage creates a new message with plain text content.
func NewPlainMessage(to []string, subject, content string) *Message {
	return &Message{
		To:      to,
		Subject: subject,
		Body:    MessagePlain{Text: content},
	}
}

// NewHTMLMessage creates a new message with HTML content.
func NewHTMLMessage(to []string, subject, content string) *Message {
	return &Message{
		To:      to,
		Subject: subject,
		Body:    MessageHTML{Text: content},
	}
}

// NewTemplateMessage creates a new message with template-based content.
func NewTemplateMessage(to []string, subject, templateName string, data any) *Message {
	return &Message{
		To:      to,
		Subject: subject,
		Body:    MessageTemplate{Name: templateName, Data: data},
	}
}

// GetAllRecipients returns all recipients (To, Cc, Bcc) as a single slice.
func (m *Message) GetAllRecipients() []string {
	var recipients []string
	recipients = append(recipients, m.To...)
	recipients = append(recipients, m.Cc...)
	recipients = append(recipients, m.Bcc...)
	return recipients
}
