package mail

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/a7medalyapany/GoBank.git/util"
	"github.com/stretchr/testify/require"
	gomail "github.com/wneessen/go-mail"
)

type mockSMTPClient struct {
	err      error
	messages []*gomail.Msg
}

func (c *mockSMTPClient) DialAndSend(messages ...*gomail.Msg) error {
	c.messages = append(c.messages, messages...)
	return c.err
}

func TestNewGmailSender(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		sender, err := NewGmailSender("Go Bank", "sender@example.com", "app-password")
		require.NoError(t, err)
		require.NotNil(t, sender)
		require.Equal(t, "Go Bank", sender.name)
		require.Equal(t, "sender@example.com", sender.fromEmailAddress)
		require.Equal(t, "app-password", sender.fromEmailPassword)
		require.NotNil(t, sender.newClient)
	})

	t.Run("empty sender name", func(t *testing.T) {
		t.Parallel()

		sender, err := NewGmailSender("", "sender@example.com", "app-password")
		require.Error(t, err)
		require.Nil(t, sender)
		require.Contains(t, err.Error(), "sender name cannot be empty")
	})

	t.Run("invalid sender email", func(t *testing.T) {
		t.Parallel()

		sender, err := NewGmailSender("Go Bank", "not-an-email", "app-password")
		require.Error(t, err)
		require.Nil(t, sender)
		require.Contains(t, err.Error(), "invalid sender email address")
	})

	t.Run("empty sender password", func(t *testing.T) {
		t.Parallel()

		sender, err := NewGmailSender("Go Bank", "sender@example.com", "")
		require.Error(t, err)
		require.Nil(t, sender)
		require.Contains(t, err.Error(), "sender password cannot be empty")
	})
}

func TestGmailSenderSendEmail(t *testing.T) {
	t.Parallel()

	t.Run("requires at least one recipient", func(t *testing.T) {
		t.Parallel()

		sender := &GmailSender{
			name:              "Go Bank",
			fromEmailAddress:  "sender@example.com",
			fromEmailPassword: "app-password",
		}

		err := sender.SendEmail("subject", "content", nil, nil, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one recipient is required")
	})

	t.Run("returns client creation error", func(t *testing.T) {
		t.Parallel()

		sender := &GmailSender{
			name:              "Go Bank",
			fromEmailAddress:  "sender@example.com",
			fromEmailPassword: "app-password",
			newClient: func(fromEmailAddress, fromEmailPassword string) (smtpClient, error) {
				return nil, errors.New("dial config failed")
			},
		}

		err := sender.SendEmail("subject", "content", []string{"to@example.com"}, nil, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create smtp client")
		require.Contains(t, err.Error(), "dial config failed")
	})

	t.Run("returns send error", func(t *testing.T) {
		t.Parallel()

		client := &mockSMTPClient{err: errors.New("send failed")}
		sender := &GmailSender{
			name:              "Go Bank",
			fromEmailAddress:  "sender@example.com",
			fromEmailPassword: "app-password",
			newClient: func(fromEmailAddress, fromEmailPassword string) (smtpClient, error) {
				return client, nil
			},
		}

		err := sender.SendEmail("subject", "content", []string{"to@example.com"}, nil, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to send email")
		require.Contains(t, err.Error(), "send failed")
		require.Len(t, client.messages, 1)
	})

	t.Run("returns attachment error for missing file", func(t *testing.T) {
		t.Parallel()

		sender := &GmailSender{
			name:              "Go Bank",
			fromEmailAddress:  "sender@example.com",
			fromEmailPassword: "app-password",
		}

		err := sender.SendEmail(
			"subject",
			"content",
			[]string{"to@example.com"},
			nil,
			nil,
			[]string{"does-not-exist.txt"},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to access attachment file")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		attachmentPath := filepath.Join(t.TempDir(), "sample.txt")
		require.NoError(t, os.WriteFile(attachmentPath, []byte("hello"), 0o600))

		client := &mockSMTPClient{}
		sender := &GmailSender{
			name:              "Go Bank",
			fromEmailAddress:  "sender@example.com",
			fromEmailPassword: "app-password",
			newClient: func(fromEmailAddress, fromEmailPassword string) (smtpClient, error) {
				return client, nil
			},
		}

		err := sender.SendEmail(
			"subject",
			"content",
			[]string{"to@example.com"},
			[]string{"cc@example.com"},
			[]string{"bcc@example.com"},
			[]string{attachmentPath},
		)
		require.NoError(t, err)
		require.Len(t, client.messages, 1)

		msg := client.messages[0]
		require.ElementsMatch(t, []string{"<to@example.com>"}, msg.GetToString())
		require.ElementsMatch(t, []string{"<cc@example.com>"}, msg.GetCcString())
		require.ElementsMatch(t, []string{"<bcc@example.com>"}, msg.GetBccString())
		require.Len(t, msg.GetAttachments(), 1)
	})
}


// TestSendRealEmail is an integration test that sends an actual email via Gmail.
// Skipped by default — run explicitly with:
//
//	go test -v -run TestSendRealEmail ./mail/
//
// Requires app.env at the project root with:
//
//	EMAIL_SENDER_NAME=...
//	EMAIL_SENDER_ADDRESS=...
//	EMAIL_SENDER_PASSWORD=...   ← Gmail App Password, not your real password
func TestSendRealEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
 
	config, err := util.LoadConfig("..")
	require.NoError(t, err)
 
	sender, err := NewGmailSender(
		config.EMAIL_SENDER_NAME,
		config.EMAIL_SENDER_ADDRESS,
		config.EMAIL_SENDER_PASSWORD,
	)
	require.NoError(t, err)
 
	err = sender.SendEmail(
		"GoBank — integration test",
		"If you received this, the Gmail SMTP sender is working correctly.",
		[]string{config.EMAIL_SENDER_ADDRESS}, // send to yourself
		nil,
		nil,
		nil,
	)
	require.NoError(t, err)
}