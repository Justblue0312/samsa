package email

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		wantErr bool
		errText string
	}{
		{
			name: "valid message with body",
			msg: Message{
				To:      []string{"user@example.com"},
				Subject: "Test Subject",
				Body:    MessagePlain{Text: "Test Body"},
			},
			wantErr: false,
		},
		{
			name: "valid message with template",
			msg: Message{
				To:      []string{"user@example.com"},
				Subject: "Test Subject",
				Body:    MessageTemplate{Name: "test.txt", Data: nil},
			},
			wantErr: false,
		},
		{
			name: "no recipients",
			msg: Message{
				Subject: "Test Subject",
				Body:    MessagePlain{Text: "Test Body"},
			},
			wantErr: true,
			errText: "at least one recipient is required",
		},
		{
			name: "no subject",
			msg: Message{
				To:   []string{"user@example.com"},
				Body: MessagePlain{Text: "Test Body"},
			},
			wantErr: true,
			errText: "subject is required",
		},
		{
			name: "no body and no template",
			msg: Message{
				To:      []string{"user@example.com"},
				Subject: "Test Subject",
			},
			wantErr: true,
			errText: "body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMessageFormatPlainText(t *testing.T) {
	msg := Message{
		To:      []string{"user@example.com"},
		Subject: "Test Subject",
		Body:    MessagePlain{Text: "Test Body"},
	}

	formatted, err := msg.Format("sender@example.com", "Sender Name")
	require.NoError(t, err)

	assert.Contains(t, formatted, "From: Sender Name <sender@example.com>")
	assert.Contains(t, formatted, "To: user@example.com")
	assert.Contains(t, formatted, "Subject: Test Subject")
	assert.Contains(t, formatted, "Content-Type: text/plain; charset=utf-8")
	assert.True(t, strings.HasSuffix(formatted, "\r\nTest Body"))
}

func TestMessageFormatHTML(t *testing.T) {
	msg := Message{
		To:      []string{"user@example.com"},
		Subject: "Test Subject",
		Body:    MessageHTML{Text: "<html><body>Test Body</body></html>"},
	}

	formatted, err := msg.Format("sender@example.com", "Sender Name")
	require.NoError(t, err)

	assert.Contains(t, formatted, "Content-Type: text/html; charset=utf-8")
	assert.True(t, strings.HasSuffix(formatted, "\r\n<html><body>Test Body</body></html>"))
}

func TestMessageFormatWithTemplate(t *testing.T) {
	t.Run("text template", func(t *testing.T) {
		msg := Message{
			To:      []string{"user@example.com"},
			Subject: "Text Template Test",
			Body:    MessageTemplate{Name: "test.txt", Data: map[string]string{"Name": "Alex"}},
		}

		formatted, err := msg.Format("sender@example.com", "")
		require.NoError(t, err)
		assert.Contains(t, formatted, "Content-Type: text/plain; charset=utf-8")
		assert.True(t, strings.HasSuffix(formatted, "\r\nHello, Alex\n"))
	})

	t.Run("html template", func(t *testing.T) {
		msg := Message{
			To:      []string{"user@example.com"},
			Subject: "HTML Template Test",
			Body:    MessageTemplate{Name: "test.html", Data: map[string]string{"Name": "Alex"}},
		}

		formatted, err := msg.Format("sender@example.com", "")
		require.NoError(t, err)

		assert.Contains(t, formatted, "Content-Type: text/html; charset=utf-8")
		assert.True(t, strings.HasSuffix(formatted, "\r\n<html>\n  <body>\n    Hello, Alex\n  </body>\n</html>\n"))
	})

	t.Run("template not found", func(t *testing.T) {
		msg := Message{
			To:      []string{"user@example.com"},
			Subject: "Not Found",
			Body:    MessageTemplate{Name: "nonexistent.txt"},
		}
		_, err := msg.Format("sender@example.com", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read template")
	})
}

func TestMessageFormatMultipleRecipients(t *testing.T) {
	msg := Message{
		To:      []string{"user1@example.com", "user2@example.com"},
		Cc:      []string{"cc1@example.com", "cc2@example.com"},
		Bcc:     []string{"bcc@example.com"},
		Subject: "Test Subject",
		Body:    MessagePlain{Text: "Test Body"},
		ReplyTo: "reply@example.com",
	}

	formatted, err := msg.Format("sender@example.com", "Sender Name")
	require.NoError(t, err)

	assert.Contains(t, formatted, "To: user1@example.com, user2@example.com")
	assert.Contains(t, formatted, "Cc: cc1@example.com, cc2@example.com")
	assert.NotContains(t, formatted, "Bcc:")
	assert.Contains(t, formatted, "Reply-To: reply@example.com")
}

func TestGetAllRecipients(t *testing.T) {
	msg := Message{
		To:   []string{"user1@example.com", "user2@example.com"},
		Cc:   []string{"cc@example.com"},
		Bcc:  []string{"bcc@example.com"},
		Body: MessagePlain{Text: "test"},
	}

	recipients := msg.GetAllRecipients()
	assert.ElementsMatch(t, []string{
		"user1@example.com",
		"user2@example.com",
		"cc@example.com",
		"bcc@example.com",
	}, recipients)
}

func TestMessageBodyTypes(t *testing.T) {
	t.Run("MessagePlain", func(t *testing.T) {
		body := MessagePlain{Text: "Hello, World!"}

		content, err := body.Content()
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World!", content)
		assert.False(t, body.IsHTML())
	})

	t.Run("MessageHTML", func(t *testing.T) {
		body := MessageHTML{Text: "<h1>Hello, World!</h1>"}

		content, err := body.Content()
		assert.NoError(t, err)
		assert.Equal(t, "<h1>Hello, World!</h1>", content)
		assert.True(t, body.IsHTML())
	})

	t.Run("MessageTemplate text", func(t *testing.T) {
		body := MessageTemplate{Name: "test.txt", Data: map[string]string{"Name": "Alice"}}

		content, err := body.Content()
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Alice\n", content)
		assert.False(t, body.IsHTML())
	})

	t.Run("MessageTemplate html", func(t *testing.T) {
		body := MessageTemplate{Name: "test.html", Data: map[string]string{"Name": "Alice"}}

		content, err := body.Content()
		assert.NoError(t, err)
		assert.Equal(t, "<html>\n  <body>\n    Hello, Alice\n  </body>\n</html>\n", content)
		assert.True(t, body.IsHTML())
	})

	t.Run("MessageTemplate not found", func(t *testing.T) {
		body := MessageTemplate{Name: "nonexistent.txt", Data: nil}

		_, err := body.Content()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read template")
	})
}

func TestConstructorFunctions(t *testing.T) {
	t.Run("NewPlainMessage", func(t *testing.T) {
		msg := NewPlainMessage([]string{"test@example.com"}, "Test Subject", "Plain content")

		assert.Equal(t, []string{"test@example.com"}, msg.To)
		assert.Equal(t, "Test Subject", msg.Subject)

		plainBody, ok := msg.Body.(MessagePlain)
		assert.True(t, ok)
		assert.Equal(t, "Plain content", plainBody.Text)
		assert.False(t, plainBody.IsHTML())
	})

	t.Run("NewHTMLMessage", func(t *testing.T) {
		msg := NewHTMLMessage([]string{"test@example.com"}, "Test Subject", "<h1>HTML content</h1>")

		assert.Equal(t, []string{"test@example.com"}, msg.To)
		assert.Equal(t, "Test Subject", msg.Subject)

		htmlBody, ok := msg.Body.(MessageHTML)
		assert.True(t, ok)
		assert.Equal(t, "<h1>HTML content</h1>", htmlBody.Text)
		assert.True(t, htmlBody.IsHTML())
	})

	t.Run("NewTemplateMessage", func(t *testing.T) {
		data := map[string]string{"Name": "Bob"}
		msg := NewTemplateMessage([]string{"test@example.com"}, "Test Subject", "test.txt", data)

		assert.Equal(t, []string{"test@example.com"}, msg.To)
		assert.Equal(t, "Test Subject", msg.Subject)

		templateBody, ok := msg.Body.(MessageTemplate)
		assert.True(t, ok)
		assert.Equal(t, "test.txt", templateBody.Name)
		assert.Equal(t, data, templateBody.Data)
		assert.False(t, templateBody.IsHTML())
	})
}
