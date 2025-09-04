package email

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

type Sender interface {
	Send(to, subject, html string) error
}

// StdoutSender prints emails to stdout (used in tests/dev)
type StdoutSender struct{}

func (StdoutSender) Send(to, subject, html string) error {
	log.Printf("EMAIL to=%s subject=%s\n%s", to, subject, html)
	return nil
}

// SMTPSender sends emails over SMTP. MailHog listens on localhost:1025 by default.
// This is a simple implementation without auth, suitable for local use with MailHog.
// From should be a valid email address (e.g., "no-reply@coachgpt.local").
type SMTPSender struct {
	Addr string // e.g. "localhost:1025"
	From string
}

func NewSMTPSender(addr, from string) *SMTPSender {
	if addr == "" {
		addr = "localhost:1025"
	}
	if from == "" {
		from = "no-reply@coachgpt.local"
	}
	return &SMTPSender{Addr: addr, From: from}
}

func (s *SMTPSender) Send(to, subject, html string) error {
	if to == "" {
		return fmt.Errorf("recipient empty")
	}

	// Build a minimal RFC 822 message with HTML body
	header := make(map[string]string)
	header["From"] = s.From
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"utf-8\""

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(html)

	// No auth, send directly to MailHog
	if err := smtp.SendMail(s.Addr, nil, s.From, []string{to}, []byte(msg.String())); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
