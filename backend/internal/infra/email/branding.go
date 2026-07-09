package email

import (
	"html"
	"strings"

	"api-server/internal/domain"
)

const publicEmailBannerURL = "https://tingting.vip/email-banner.jpg"

const publicEmailBannerHTML = `<p style="text-align:left;"><img src="https://tingting.vip/email-banner.jpg" alt="Ting Ting Soft" width="600" style="display:block;width:100%;max-width:600px;height:auto;border:0;"></p>`

func withPublicEmailBanner(htmlBody, textBody string) string {
	body := strings.TrimSpace(htmlBody)
	if body == "" {
		body = textBodyToHTML(textBody)
	}
	if body == "" || strings.Contains(body, publicEmailBannerURL) {
		return body
	}

	lowerBody := strings.ToLower(body)
	if bodyIdx := strings.Index(lowerBody, "<body"); bodyIdx >= 0 {
		if closeIdx := strings.Index(lowerBody[bodyIdx:], ">"); closeIdx >= 0 {
			insertPos := bodyIdx + closeIdx + 1
			return body[:insertPos] + "\n" + publicEmailBannerHTML + "\n" + body[insertPos:]
		}
	}

	return publicEmailBannerHTML + "\n" + body
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
	return `<p style="margin:0 0 12px;line-height:1.5;">` + escaped + `</p>`
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
