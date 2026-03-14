package port

import "context"

type EmailSender interface {
	SendVerificationCode(ctx context.Context, to, code, lang string) error
}
