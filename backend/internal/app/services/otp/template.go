package otp

import (
	"fmt"
	"strings"

	"api-server/internal/domain"
)

// BuildOTPEmailMessage constructs the Vietnamese-localized OTP login email.
// Only the 6-digit code is placed in the body — never the otp_session_id
// (that is a server session token, not a user-facing value). The template is
// intentionally table-based HTML; the brand banner is referenced by a public
// URL (no inline attachment) so the email stays lightweight and deliverable.
func BuildOTPEmailMessage(code, recipientEmail, recipientName, fromEmail string) (*domain.EmailMessage, error) {
	to, err := domain.ParseEmailAddress(recipientEmail)
	if err != nil {
		return nil, fmt.Errorf("otp: parse recipient email: %w", err)
	}
	from, err := domain.ParseEmailAddress(fromEmail)
	if err != nil {
		return nil, fmt.Errorf("otp: parse from email: %w", err)
	}
	if strings.TrimSpace(recipientName) != "" {
		to.Name = recipientName
	}

	subject := "Ting Ting Soft OTP"
	textBody := strings.Join([]string{
		"Mã đăng nhập của bạn là: " + code,
		"Mã có hiệu lực trong 5 phút.",
		"Nếu bạn không yêu cầu đăng nhập, vui lòng bỏ qua email này và cân nhắc đổi mật khẩu.",
		"",
		"— Ting Ting Soft",
	}, "\n")

	htmlBody := buildOTPEmailHTML(code)

	msg := &domain.EmailMessage{
		Kind:     domain.EmailKindOTP,
		From:     from,
		To:       []domain.EmailAddress{to},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("otp: validate email message: %w", err)
	}
	return msg, nil
}

func buildOTPEmailHTML(code string) string {
	return strings.NewReplacer(
		"{{CODE}}", code,
	).Replace(otpEmailTemplate)
}

const otpEmailTemplate = `<!DOCTYPE html>
<html lang="vi">
<head><meta charset="utf-8"><title>Mã xác thực đăng nhập</title></head>
<body style="margin:0;padding:0;background:#f4f5f7;font-family:Arial,Helvetica,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="padding:24px;">
    <tr><td align="center">
      <table role="presentation" width="420" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;padding:32px 24px;">
        <tr><td style="padding-bottom:16px;"><img src="https://tingting.vip/email-banner.jpg" alt="Ting Ting Soft" width="600" style="display:block;width:100%;max-width:600px;height:auto;border:0;"></td></tr>
        <tr><td style="color:#1a1a1a;font-size:15px;line-height:1.5;padding-bottom:8px;">Mã đăng nhập của bạn là:</td></tr>
        <tr><td align="center" style="padding:16px 0 24px;">
          <div style="display:inline-block;font-size:32px;font-weight:bold;letter-spacing:8px;color:#1a1a1a;background:#f0ebe1;border-radius:6px;padding:12px 24px;">{{CODE}}</div>
        </td></tr>
        <tr><td style="color:#6b6258;font-size:13px;line-height:1.5;">Mã có hiệu lực trong 5 phút. Nếu bạn không yêu cầu đăng nhập, vui lòng bỏ qua email này.</td></tr>
      </table>
      <table role="presentation" width="420" cellpadding="0" cellspacing="0" style="color:#9b9b9b;font-size:12px;text-align:center;padding-top:16px;">
        <tr><td>© Ting Ting Soft — Email tự động, vui lòng không trả lời.</td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
