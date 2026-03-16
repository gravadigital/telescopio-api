package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
)

// EmailService handles sending email notifications via SMTP.
type EmailService struct {
	cfg *config.Config
	log *log.Logger
}

// NewEmailService creates a new EmailService.
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		cfg: cfg,
		log: logger.Service("email"),
	}
}

// SendStageChangeNotification sends an email to all recipients informing them
// of the event stage change.
func (s *EmailService) SendStageChangeNotification(eventName, newStage string, recipients []string) error {
	if !s.cfg.Email.Enabled {
		s.log.Debug("email disabled, skipping stage change notification",
			"event", eventName, "stage", newStage, "recipients", len(recipients))
		return nil
	}
	if len(recipients) == 0 {
		return nil
	}
	return s.send(recipients, stageChangeSubject(eventName, newStage), stageChangeBody(eventName, newStage))
}

// SendCancellationNotification sends an email to all recipients informing them
// that the event has been cancelled.
func (s *EmailService) SendCancellationNotification(eventName string, recipients []string) error {
	if !s.cfg.Email.Enabled {
		s.log.Debug("email disabled, skipping cancellation notification",
			"event", eventName, "recipients", len(recipients))
		return nil
	}
	if len(recipients) == 0 {
		return nil
	}
	return s.send(recipients, cancellationSubject(eventName), cancellationBody(eventName))
}

// send abre una única conexión SMTP y envía un correo individual a cada destinatario.
// Cada participante solo ve su propio email en el campo To.
func (s *EmailService) send(recipients []string, subject, body string) error {
	client, err := s.newClient()
	if err != nil {
		return err
	}
	defer client.Close()

	sent, failed := 0, 0
	for _, recipient := range recipients {
		if err := s.sendOne(client, recipient, subject, body); err != nil {
			s.log.Warn("failed to send email to recipient", "recipient", recipient, "error", err)
			failed++
		} else {
			sent++
		}
	}

	s.log.Info("email batch completed", "subject", subject, "sent", sent, "failed", failed)
	return nil
}

// newClient abre y autentica una conexión SMTP.
// Soporta SSL/TLS directo (puerto 465, SMTP_SECURE=true) y STARTTLS (puerto 587).
func (s *EmailService) newClient() (*smtp.Client, error) {
	addr := s.cfg.Email.SMTPHost + ":" + s.cfg.Email.SMTPPort
	auth := smtp.PlainAuth("", s.cfg.Email.SMTPUser, s.cfg.Email.SMTPPassword, s.cfg.Email.SMTPHost)

	var client *smtp.Client
	if s.cfg.Email.Secure {
		tlsCfg := &tls.Config{ServerName: s.cfg.Email.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			s.log.Error("failed to connect via TLS", "addr", addr, "error", err)
			return nil, err
		}
		client, err = smtp.NewClient(conn, s.cfg.Email.SMTPHost)
		if err != nil {
			s.log.Error("failed to create SMTP client", "error", err)
			return nil, err
		}
	} else {
		var err error
		client, err = smtp.Dial(addr)
		if err != nil {
			s.log.Error("failed to connect to SMTP server", "addr", addr, "error", err)
			return nil, err
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Email.SMTPHost}); err != nil {
			s.log.Error("STARTTLS failed", "error", err)
			return nil, err
		}
	}

	if err := client.Auth(auth); err != nil {
		s.log.Error("SMTP auth failed", "error", err)
		client.Close()
		return nil, err
	}
	return client, nil
}

// sendOne envía un único correo a un destinatario usando un cliente SMTP ya autenticado.
func (s *EmailService) sendOne(client *smtp.Client, to, subject, body string) error {
	from := fmt.Sprintf("%s <%s>", s.cfg.Email.FromName, s.cfg.Email.FromAddress)
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	if err := client.Mail(s.cfg.Email.FromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, msg); err != nil {
		return err
	}
	return w.Close()
}
