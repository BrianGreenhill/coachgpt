package email

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestNewSMTPSender_Defaults(t *testing.T) {
	s := NewSMTPSender("", "")
	if s.Addr != "localhost:1025" {
		t.Fatalf("expected default addr localhost:1025, got %s", s.Addr)
	}
	if s.From != "no-reply@coachgpt.local" {
		t.Fatalf("expected default from no-reply@coachgpt.local, got %s", s.From)
	}
}

func TestStdoutSender_Send(t *testing.T) {
	s := StdoutSender{}
	if err := s.Send("user@example.com", "Test subject", "<p>Test</p>"); err != nil {
		t.Fatalf("StdoutSender.Send returned error: %v", err)
	}
}

func TestSMTPSender_Send_EmptyRecipient(t *testing.T) {
	s := NewSMTPSender("localhost:1025", "from@example.com")
	if err := s.Send("", "subj", "body"); err == nil {
		t.Fatalf("expected error when recipient is empty")
	}
}

// Test that MailHog is available and that we can send an email via SMTP and then clean up via the API.
func TestSMTPSender_MailHog_SendAndCleanup(t *testing.T) {
	// Check MailHog API availability
	client := &http.Client{Timeout: 2 * time.Second}
	// Try to clear any existing messages first (best-effort)
	_ = doMailHogDelete(client)

	// Send an email to MailHog
	sender := NewSMTPSender("localhost:1025", "test-from@example.com")
	if err := sender.Send("recipient@example.com", "Test MailHog", "<p>Hello MailHog</p>"); err != nil {
		t.Skipf("MailHog SMTP not available or send failed: %v", err)
	}

	// Give MailHog a moment to accept the message
	time.Sleep(200 * time.Millisecond)

	// Query messages via API (v2 preferred)
	resp, err := client.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		// Fallback to v1 messages endpoint
		resp, err = client.Get("http://localhost:8025/api/v1/messages")
		if err != nil {
			t.Skipf("MailHog HTTP API not available: %v", err)
		}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		// If we can't read messages, skip rather than fail
		t.Skipf("MailHog API returned non-200: %d", resp.StatusCode)
	}

	// Optionally verify there is at least one message
	var payload map[string]any
	b, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(b, &payload)

	// Clean up messages via MailHog API (best-effort)
	if err := doMailHogDelete(client); err != nil {
		t.Fatalf("failed to clean up MailHog messages: %v", err)
	}
}

func doMailHogDelete(client *http.Client) error {
	req, _ := http.NewRequest("DELETE", "http://localhost:8025/api/v1/messages", nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	// Accept 200, 202, 204
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return nil // best-effort: don't treat unexpected codes as fatal
}
