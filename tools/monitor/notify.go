package main

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

func sendEmail(cfg *Config, subject, body string) error {
	if len(cfg.Email.To) == 0 {
		return fmt.Errorf("нет получателей email.to")
	}

	from := cfg.Email.From
	to := cfg.Email.To
	msg := buildMessage(from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", cfg.Email.SMTPHost, cfg.Email.SMTPPort)
	auth := smtpAuth(cfg)

	// SendMail: для порта 587/465 автоматически включает STARTTLS при необходимости.
	if auth != nil || cfg.Email.UseTLS {
		return smtp.SendMail(addr, auth, from, to, msg)
	}

	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func buildMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func smtpAuth(cfg *Config) smtp.Auth {
	if cfg.Email.Username == "" {
		return nil
	}
	return smtp.PlainAuth("", cfg.Email.Username, cfg.Email.Password, cfg.Email.SMTPHost)
}

type alertState struct {
	up        bool
	lastAlert time.Time
}

func (s *alertState) shouldNotify(cooldown time.Duration) bool {
	if s.lastAlert.IsZero() {
		return true
	}
	return time.Since(s.lastAlert) >= cooldown
}

func (s *alertState) markNotified() {
	s.lastAlert = time.Now()
}
