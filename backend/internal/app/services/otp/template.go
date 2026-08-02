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

	subject := "TingTing Soft OTP"
	textBody := strings.Join([]string{
		"Mã đăng nhập của bạn là: " + code,
		"Mã có hiệu lực trong 5 phút.",
		"Nếu bạn không yêu cầu đăng nhập, vui lòng bỏ qua email này và cân nhắc đổi mật khẩu.",
		"",
		"— TingTing Soft",
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
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Mã xác thực đăng nhập</title>
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
            <p style="margin:0 0 8px;color:#64748b;font-size:13px;line-height:20px;font-weight:600;letter-spacing:.02em;text-transform:uppercase;">Xác thực đăng nhập</p>
            <h1 style="margin:0;color:#111827;font-size:24px;line-height:32px;font-weight:700;">Mã đăng nhập TingTing</h1>
            <p style="margin:16px 0 0;color:#334155;font-size:15px;line-height:24px;">Nhập mã bên dưới để hoàn tất đăng nhập. Mã chỉ có hiệu lực trong 5 phút.</p>
            <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin:24px 0;background:#f8fafc;border:1px solid #e2e8f0;border-radius:14px;">
              <tr>
                <td align="center" style="padding:20px 16px;">
                  <div style="display:inline-block;color:#0f172a;font-size:34px;line-height:42px;font-weight:800;letter-spacing:8px;">{{CODE}}</div>
                </td>
              </tr>
            </table>
            <p style="margin:0;color:#64748b;font-size:13px;line-height:21px;">Nếu bạn không yêu cầu đăng nhập, vui lòng bỏ qua email này và cân nhắc đổi mật khẩu.</p>
          </td>
        </tr>
        <tr><td style="padding:16px 8px 0;text-align:center;color:#94a3b8;font-size:12px;line-height:18px;">Email tự động từ TingTing.</td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`
