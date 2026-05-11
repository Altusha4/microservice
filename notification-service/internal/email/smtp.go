package email

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"time"
)

// #######################################
// SMTP ADAPTER (Mailtrap or any standard SMTP server)
// #######################################
//
// Configured via env:
//   SMTP_HOST     (e.g. sandbox.smtp.mailtrap.io)
//   SMTP_PORT     (e.g. 2525)
//   SMTP_USERNAME
//   SMTP_PASSWORD
//   SMTP_FROM     (default no-reply@microservice.local)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) SendOrderConfirmation(ctx context.Context, msg Message) error {
	// SMTP libraries are not context-aware; we enforce a deadline
	// by running the dial in a goroutine and racing it with ctx.
	addr := s.cfg.Host + ":" + s.cfg.Port
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	body := buildEmailBody(s.cfg.From, msg)

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, body)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("[email][smtp] FAIL order=%s to=%s: %v", msg.OrderID, msg.To, err)
			return fmt.Errorf("smtp send: %w", err)
		}
		log.Printf("[email][smtp] OK order=%s to=%s amount=%s", msg.OrderID, msg.To, msg.AmountUS)
		return nil

	case <-ctx.Done():
		// SMTP call may still finish in the background — Mailtrap sandbox
		// won't be hurt by a stray message. Caller already moved on.
		return ctx.Err()
	}
}

// buildEmailBody assembles an RFC-822 message: headers + blank line + body.
func buildEmailBody(from string, msg Message) []byte {
	return []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: Your order %s is confirmed\r\n"+
			"Date: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
			"\r\n"+
			"Hello!\r\n\r\n"+
			"Your order %s has been paid. Amount: %s.\r\n\r\n"+
			"Thank you for your purchase!\r\n",
		from, msg.To, msg.OrderID,
		time.Now().UTC().Format(time.RFC1123Z),
		msg.OrderID, msg.AmountUS,
	))
}
