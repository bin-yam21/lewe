// Package mail sends transactional email (verification and password reset).
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"time"
)

// Message is a plain-text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer delivers email.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// LogMailer writes emails to the log instead of sending them. It is the
// default when no SMTP server is configured, which suits local development.
type LogMailer struct{}

// Send logs the message.
func (LogMailer) Send(_ context.Context, msg Message) error {
	log.Printf("email (not sent; SMTP_HOST unset) to=%s subject=%q\n%s", msg.To, msg.Subject, msg.Body)
	return nil
}

// SMTPConfig configures an SMTPMailer.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPMailer sends email through an SMTP server, using STARTTLS when the
// server offers it (or implicit TLS on port 465).
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer creates an SMTP mailer.
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

// Send delivers msg, giving up after the context deadline or 15 seconds.
func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	deadline := time.Now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprint(m.cfg.Port))
	dialer := &net.Dialer{Deadline: deadline}

	var conn net.Conn
	var err error
	if m.cfg.Port == 465 {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: m.cfg.Host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	_ = conn.SetDeadline(deadline)

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); ok && m.cfg.Port != 465 {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if m.cfg.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(format(m.cfg.From, msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return c.Quit()
}

// format renders an RFC 5322 message. Header values are stripped of line
// breaks so user-controlled text cannot inject headers.
func format(from string, msg Message) []byte {
	clean := strings.NewReplacer("\r", "", "\n", "").Replace
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", clean(from))
	fmt.Fprintf(&b, "To: %s\r\n", clean(msg.To))
	fmt.Fprintf(&b, "Subject: %s\r\n", clean(msg.Subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(strings.ReplaceAll(msg.Body, "\n", "\r\n"))
	return []byte(b.String())
}

// MemoryMailer records messages instead of sending them. It is safe for
// concurrent use and intended for tests.
type MemoryMailer struct {
	mu   sync.Mutex
	sent []Message
}

// Send records msg.
func (m *MemoryMailer) Send(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

// Sent returns a copy of the recorded messages.
func (m *MemoryMailer) Sent() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Message(nil), m.sent...)
}
