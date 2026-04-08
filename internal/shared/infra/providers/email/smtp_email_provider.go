package email

import (
	"context"
	"fmt"
	"net/smtp"

	"golang_boilerplate_module/internal/config"
	"golang_boilerplate_module/internal/shared/domain/providers"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("email.provider")

// SMTPEmailProvider sends emails via SMTP using Go's net/smtp package.
type SMTPEmailProvider struct {
	cfg config.EmailConfig
}

// NewSMTPEmailProvider creates a new SMTP-based EmailProvider.
func NewSMTPEmailProvider(cfg *config.Config) providers.EmailProvider {
	return &SMTPEmailProvider{cfg: cfg.Email}
}

func (p *SMTPEmailProvider) Send(ctx context.Context, msg providers.EmailMessage) error {
	ctx, span := tracer.Start(ctx, "SMTPEmailProvider.Send",
		trace.WithAttributes(
			attribute.String("email.to", msg.To),
			attribute.String("email.subject", msg.Subject),
			attribute.String("email.template", msg.TemplateName),
		),
	)
	defer span.End()

	body, err := RenderTemplate(msg.TemplateName, msg.Data)
	if err != nil {
		span.SetStatus(codes.Error, "render template failed")
		span.RecordError(err)
		return fmt.Errorf("render template %s: %w", msg.TemplateName, err)
	}

	mime := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n%s", p.cfg.FromName, p.cfg.FromAddress, msg.To, msg.Subject, body)

	addr := fmt.Sprintf("%s:%d", p.cfg.SMTPHost, p.cfg.SMTPPort)

	var auth smtp.Auth
	if p.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", p.cfg.SMTPUser, p.cfg.SMTPPassword, p.cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, p.cfg.FromAddress, []string{msg.To}, []byte(mime)); err != nil {
		span.SetStatus(codes.Error, "send email failed")
		span.RecordError(err)
		return fmt.Errorf("send email to %s: %w", msg.To, err)
	}

	span.SetStatus(codes.Ok, "email sent")
	return nil
}
