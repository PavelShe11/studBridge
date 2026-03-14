package email

import (
	"context"
	"fmt"
	"math"
	"net/smtp"
	"time"

	"github.com/PavelShe11/studbridge/authMicro/internal/config"
	"github.com/PavelShe11/studbridge/common/translator"
)

type SmtpEmailSender struct {
	config     config.SmtpConfig
	translator *translator.Translator
	codeTTL    time.Duration
}

func NewSmtpEmailSender(cfg config.SmtpConfig, trans *translator.Translator, codeTTL time.Duration) *SmtpEmailSender {
	return &SmtpEmailSender{
		config:     cfg,
		translator: trans,
		codeTTL:    codeTTL,
	}
}

func (s *SmtpEmailSender) SendVerificationCode(_ context.Context, to, code, lang string) error {
	minutes := int(math.Ceil(s.codeTTL.Minutes()))

	params := map[string]interface{}{
		"Code":    code,
		"Minutes": minutes,
	}

	subject := s.translator.Translate("emailVerificationSubject", nil, lang)
	body := s.translator.Translate("emailVerificationBody", params, lang)

	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.config.From, to, subject, body,
	)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message))
}
