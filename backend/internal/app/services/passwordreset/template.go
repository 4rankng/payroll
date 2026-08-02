package passwordreset

import (
	"fmt"
	"net/url"
	"strings"

	"api-server/internal/domain"
)

// BuildResetEmailMessage constructs the Vietnamese-localized password-reset
// email containing a single-use magic-link button. Mirrors the structure of
// otp.BuildOTPEmailMessage. Only the token is embedded in the CTA link — it is
// single-use with a 30-min TTL.
func BuildResetEmailMessage(token, recipientEmail, recipientName, fromEmail, resetURL string) (*domain.EmailMessage, error) {
	to, err := domain.ParseEmailAddress(recipientEmail)
	if err != nil {
		return nil, fmt.Errorf("passwordreset: parse recipient email: %w", err)
	}
	from, err := domain.ParseEmailAddress(fromEmail)
	if err != nil {
		return nil, fmt.Errorf("passwordreset: parse from email: %w", err)
	}
	if strings.TrimSpace(recipientName) != "" {
		to.Name = recipientName
	}

	// base64.RawURLEncoding has no chars needing escape, but be defensive.
	link := resetURL + "?token=" + url.QueryEscape(token)

	subject := "Đặt lại mật khẩu TingTing"
	textBody := strings.Join([]string{
		"Chào " + strings.TrimSpace(recipientName) + ",",
		"",
		"Chúng tôi đã nhận được yêu cầu đặt lại mật khẩu cho tài khoản TingTing của bạn.",
		"Nhấn vào liên kết bên dưới để đặt mật khẩu mới (liên kết có hiệu lực trong 30 phút):",
		"",
		link,
		"",
		"Nếu bạn không yêu cầu đặt lại mật khẩu, vui lòng bỏ qua email này và cân nhắc đổi mật khẩu.",
		"",
		"— TingTing Soft",
	}, "\n")

	htmlBody := buildResetEmailHTML(link, recipientName)

	msg := &domain.EmailMessage{
		Kind:     domain.EmailKindPasswordReset,
		From:     from,
		To:       []domain.EmailAddress{to},
		Subject:  subject,
		TextBody: textBody,
		HTMLBody: htmlBody,
	}
	if err := msg.Validate(); err != nil {
		return nil, fmt.Errorf("passwordreset: validate email message: %w", err)
	}
	return msg, nil
}

func buildResetEmailHTML(link, recipientName string) string {
	greeting := "Xin chào,"
	if n := strings.TrimSpace(recipientName); n != "" {
		greeting = "Xin chào " + n + ","
	}
	return strings.NewReplacer(
		"{{GREETING}}", greeting,
		"{{LINK}}", link,
	).Replace(resetEmailTemplate)
}

const resetEmailTemplate = `<!DOCTYPE html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Đặt lại mật khẩu</title>
</head>
<body style="margin:0;padding:0;background:#f5f7fb;color:#172033;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f7fb;margin:0;padding:28px 12px;">
    <tr><td align="center">
      <table role="presentation" width="520" cellpadding="0" cellspacing="0" style="width:100%;max-width:520px;border-collapse:separate;border-spacing:0;">
        <tr>
          <td style="padding:0 0 14px;">
            <img src="https://tingting.vip/email-banner.jpg?v=20260709" alt="TingTing Soft" width="520" style="display:block;width:100%;max-width:520px;height:auto;border:0;border-radius:16px;">
          </td>
        </tr>
        <tr>
          <td style="background:#ffffff;border:1px solid #e6eaf2;border-radius:16px;padding:30px 32px;box-shadow:0 10px 30px rgba(15,23,42,0.06);">
            <p style="margin:0 0 8px;color:#64748b;font-size:13px;line-height:20px;font-weight:600;letter-spacing:.02em;text-transform:uppercase;">Đặt lại mật khẩu</p>
            <h1 style="margin:0;color:#111827;font-size:24px;line-height:32px;font-weight:700;">Đặt lại mật khẩu TingTing</h1>
            <p style="margin:16px 0 0;color:#334155;font-size:15px;line-height:24px;">{{GREETING}}</p>
            <p style="margin:8px 0 0;color:#334155;font-size:15px;line-height:24px;">Chúng tôi đã nhận được yêu cầu đặt lại mật khẩu cho tài khoản của bạn. Nhấn vào nút bên dưới để đặt mật khẩu mới. Liên kết chỉ có hiệu lực trong 30 phút.</p>
            <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:24px 0;">
              <tr>
                <td align="center">
                  <a href="{{LINK}}" style="display:inline-block;background:#0a7d3c;color:#ffffff;font-size:15px;font-weight:700;text-decoration:none;padding:12px 28px;border-radius:10px;">Đặt lại mật khẩu</a>
                </td>
              </tr>
            </table>
            <p style="margin:0;color:#64748b;font-size:13px;line-height:21px;">Nếu bạn không yêu cầu đặt lại mật khẩu, vui lòng bỏ qua email này và cân nhắc đổi mật khẩu.</p>
          </td>
        </tr>
        <tr><td style="padding:16px 8px 0;text-align:center;color:#94a3b8;font-size:12px;line-height:18px;">Email tự động từ TingTing.</td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
