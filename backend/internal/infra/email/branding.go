package email

import (
	"html"
	"regexp"
	"strings"

	"api-server/internal/domain"
)

const publicEmailBannerURL = "https://tingting.vip/email-banner.jpg?v=20260709"
const publicEmailBannerPath = "tingting.vip/email-banner.jpg"

const publicEmailBannerHTML = `<img src="https://tingting.vip/email-banner.jpg?v=20260709" alt="TingTing Soft" width="640" style="display:block;width:100%;max-width:640px;height:auto;border:0;border-radius:16px;">`

var publicEmailBannerImagePattern = regexp.MustCompile(`(?is)<img\b[^>]*\bsrc\s*=\s*(?:"[^"]*tingting\.vip/email-banner\.jpg[^"]*"|'[^']*tingting\.vip/email-banner\.jpg[^']*'|[^\s>]*tingting\.vip/email-banner\.jpg[^\s>]*)[^>]*>`)

func withPublicEmailBanner(htmlBody, textBody string) string {
	body := strings.TrimSpace(htmlBody)
	if body == "" {
		body = textBodyToHTML(textBody)
	}
	if body == "" {
		return body
	}
	lowerBody := strings.ToLower(body)
	if !strings.Contains(lowerBody, "<html") && !strings.Contains(lowerBody, "<body") {
		// Fragments are wrapped in the maintained shell, which supplies the
		// canonical banner. Remove any pasted copies before wrapping.
		body = publicEmailBannerImagePattern.ReplaceAllString(body, "")
		return brandedEmailShell(body)
	}

	// Complete templates already position their banner within an email-safe
	// table layout. Canonicalize the first banner in place and remove duplicates
	// instead of detaching it above the table, which adds empty vertical space
	// and can push content behind fixed controls in mobile mail clients.
	if matches := publicEmailBannerImagePattern.FindAllStringIndex(body, -1); len(matches) > 0 {
		var normalized strings.Builder
		normalized.Grow(len(body) + len(publicEmailBannerHTML))
		cursor := 0
		for i, match := range matches {
			normalized.WriteString(body[cursor:match[0]])
			if i == 0 {
				normalized.WriteString(publicEmailBannerHTML)
			}
			cursor = match[1]
		}
		normalized.WriteString(body[cursor:])
		return normalized.String()
	}

	if bodyIdx := strings.Index(lowerBody, "<body"); bodyIdx >= 0 {
		if closeIdx := strings.Index(lowerBody[bodyIdx:], ">"); closeIdx >= 0 {
			insertPos := bodyIdx + closeIdx + 1
			return body[:insertPos] + "\n" + `<div style="max-width:640px;margin:0 auto 16px;">` + publicEmailBannerHTML + `</div>` + "\n" + body[insertPos:]
		}
	}

	return brandedEmailShell(body)
}

func textBodyToHTML(textBody string) string {
	text := strings.TrimSpace(textBody)
	if text == "" {
		return ""
	}

	escaped := html.EscapeString(text)
	escaped = strings.ReplaceAll(escaped, "\r\n", "\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\n")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
	return `<p style="margin:0;color:#334155;font-size:15px;line-height:24px;">` + escaped + `</p>`
}

func brandedEmailShell(innerHTML string) string {
	return `<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>TingTing</title>
</head>
<body style="margin:0;padding:0;background:#f5f7fb;color:#172033;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f5f7fb;margin:0;padding:28px 12px;">
    <tr>
      <td align="center">
        <table role="presentation" width="640" cellpadding="0" cellspacing="0" style="width:100%;max-width:640px;border-collapse:separate;border-spacing:0;">
          <tr>
            <td style="padding:0 0 14px;">` + publicEmailBannerHTML + `</td>
          </tr>
          <tr>
            <td style="background:#ffffff;border:1px solid #e6eaf2;border-radius:16px;padding:28px 32px;box-shadow:0 10px 30px rgba(15,23,42,0.06);">
              <div style="color:#334155;font-size:15px;line-height:24px;">` + innerHTML + `</div>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`
}

func cloneMessageWithPublicBanner(msg *domain.EmailMessage) *domain.EmailMessage {
	if msg == nil {
		return nil
	}

	clone := *msg
	clone.To = append([]domain.EmailAddress{}, msg.To...)
	clone.CC = append([]domain.EmailAddress{}, msg.CC...)
	clone.BCC = append([]domain.EmailAddress{}, msg.BCC...)
	clone.Attachments = append([]domain.EmailAttachment{}, msg.Attachments...)
	if msg.Metadata != nil {
		clone.Metadata = make(map[string]string, len(msg.Metadata))
		for key, value := range msg.Metadata {
			clone.Metadata[key] = value
		}
	}
	clone.HTMLBody = withPublicEmailBanner(msg.HTMLBody, msg.TextBody)
	return &clone
}
