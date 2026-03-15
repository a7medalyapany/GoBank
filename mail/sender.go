package mail

import (
	"fmt"
	stdmail "net/mail"
	"os"
	"strings"

	gomail "github.com/wneessen/go-mail"
)

const (
	gmailSMTPHost = "smtp.gmail.com"
	gmailSMTPPort = 587
)

type EmailSender interface {
	SendEmail(
		subject string,
		content string,
		to []string,
		cc []string,
		bcc []string,
		attachFiles []string,
	) error
}

type smtpClient interface {
	DialAndSend(messages ...*gomail.Msg) error
}

type smtpClientFactory func(fromEmailAddress, fromEmailPassword string) (smtpClient, error)

type GmailSender struct {
	name              string
	fromEmailAddress  string
	fromEmailPassword string
	newClient         smtpClientFactory
}

func NewGmailSender(name, fromEmailAddress, fromEmailPassword string) (*GmailSender, error) {
	name = strings.TrimSpace(name)
	fromEmailAddress = strings.TrimSpace(fromEmailAddress)
	fromEmailPassword = strings.TrimSpace(fromEmailPassword)

	if name == "" {
		return nil, fmt.Errorf("sender name cannot be empty")
	}

	if _, err := stdmail.ParseAddress(fromEmailAddress); err != nil {
		return nil, fmt.Errorf("invalid sender email address: %w", err)
	}

	if fromEmailPassword == "" {
		return nil, fmt.Errorf("sender password cannot be empty")
	}

	return &GmailSender{
		name:              name,
		fromEmailAddress:  fromEmailAddress,
		fromEmailPassword: fromEmailPassword,
		newClient:         newGmailClient,
	}, nil
}

func newGmailClient(fromEmailAddress, fromEmailPassword string) (smtpClient, error) {
	return gomail.NewClient(
		gmailSMTPHost,
		gomail.WithPort(gmailSMTPPort),
		gomail.WithTLSPolicy(gomail.TLSMandatory),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(fromEmailAddress),
		gomail.WithPassword(fromEmailPassword),
	)
}

func (sender *GmailSender) SendEmail(
	subject string,
	content string,
	to []string,
	cc []string,
	bcc []string,
	attachFiles []string,
) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	e := gomail.NewMsg()

	if err := e.FromFormat(sender.name, sender.fromEmailAddress); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}

	if err := e.To(to...); err != nil {
		return fmt.Errorf("failed to set to recipients: %w", err)
	}

	if len(cc) > 0 {
		if err := e.Cc(cc...); err != nil {
			return fmt.Errorf("failed to set cc recipients: %w", err)
		}
	}

	if len(bcc) > 0 {
		if err := e.Bcc(bcc...); err != nil {
			return fmt.Errorf("failed to set bcc recipients: %w", err)
		}
	}

	e.Subject(subject)
	e.SetBodyString(gomail.TypeTextPlain, content)

	for _, attachFile := range attachFiles {
		attachFile = strings.TrimSpace(attachFile)
		if attachFile == "" {
			return fmt.Errorf("attachment file path cannot be empty")
		}

		if _, err := os.Stat(attachFile); err != nil {
			return fmt.Errorf("failed to access attachment file %s: %w", attachFile, err)
		}

		e.AttachFile(attachFile)
	}

	newClient := sender.newClient
	if newClient == nil {
		newClient = newGmailClient
	}

	d, err := newClient(sender.fromEmailAddress, sender.fromEmailPassword)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %w", err)
	}

	if err := d.DialAndSend(e); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
