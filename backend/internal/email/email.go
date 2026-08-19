// Package email sends transactional email through Resend. When no Resend API
// key is configured it runs in a dev-friendly mode that logs the message (and
// any action link) to stdout instead of sending, so local development needs no
// external credentials.
package email

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/resend/resend-go/v3"
)

// Mailer sends the app's transactional messages.
type Mailer struct {
	client     *resend.Client // nil in dev/log mode
	from       string
	appBaseURL string
}

// New builds a Mailer. If apiKey is empty the mailer logs instead of sending.
func New(apiKey, from, appBaseURL string) *Mailer {
	m := &Mailer{from: from, appBaseURL: appBaseURL}
	if apiKey != "" {
		m.client = resend.NewClient(apiKey)
	} else {
		log.Println("[email] RESEND_API_KEY not set — verification links will be logged to the console, not emailed")
	}
	return m
}

// Enabled reports whether real email delivery is configured.
func (m *Mailer) Enabled() bool { return m.client != nil }

// VerifyLink builds the public verify-email URL for a raw token.
func (m *Mailer) VerifyLink(rawToken string) string {
	return fmt.Sprintf("%s/verify-email?token=%s", m.appBaseURL, url.QueryEscape(rawToken))
}

// SendVerification emails the address a link that verifies their account.
func (m *Mailer) SendVerification(ctx context.Context, to, rawToken string) error {
	link := m.VerifyLink(rawToken)

	if m.client == nil {
		log.Printf("[email:dev] verification for %s -> %s", to, link)
		return nil
	}

	params := &resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: "Verify your SS Tax Engine account",
		Html:    verificationHTML(link),
		Text:    verificationText(link),
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if _, err := m.client.Emails.SendWithContext(ctx, params); err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}

func verificationText(link string) string {
	return "Welcome to SS Tax Engine.\n\n" +
		"Confirm your email address to activate your account:\n" + link +
		"\n\nThis link expires in 24 hours. If you didn't create an account, you can ignore this email."
}

func verificationHTML(link string) string {
	return fmt.Sprintf(`<!doctype html>
<html><body style="margin:0;background:#f5f5f4;font-family:Inter,Arial,sans-serif;color:#1c1917;">
  <div style="max-width:520px;margin:0 auto;padding:32px 24px;">
    <h1 style="font-family:Georgia,'Times New Roman',serif;font-size:22px;color:#0f766e;margin:0 0 16px;">
      Confirm your email
    </h1>
    <p style="font-size:15px;line-height:1.6;margin:0 0 24px;">
      Welcome to SS&nbsp;Tax&nbsp;Engine. Confirm your email address to activate
      your account and start computing Social Security taxability for your clients.
    </p>
    <p style="margin:0 0 28px;">
      <a href="%s" style="display:inline-block;background:#0f766e;color:#ffffff;
         text-decoration:none;font-size:15px;font-weight:600;padding:12px 22px;border-radius:8px;">
        Verify my email
      </a>
    </p>
    <p style="font-size:13px;color:#57534e;line-height:1.6;margin:0 0 8px;">
      Or paste this link into your browser:
    </p>
    <p style="font-size:13px;word-break:break-all;margin:0 0 24px;">
      <a href="%s" style="color:#0f766e;">%s</a>
    </p>
    <p style="font-size:12px;color:#78716c;margin:0;">
      This link expires in 24 hours. If you didn't create an account, ignore this email.
    </p>
  </div>
</body></html>`, link, link, link)
}
