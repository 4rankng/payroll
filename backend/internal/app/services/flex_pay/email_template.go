package flex_pay

import (
	"fmt"
	"html"

	"api-server/internal/app/services/config"
)

// BuildSaoKeEmailBodies generates the HTML and text bodies for the FlexPay sao ke
// reconciliation email. Shared by the manual handler and the automated cron job.
//
// Wording is branded for the "Nhận lương sớm 24/7" early-salary service by
// TING TING SOFT, so partners can distinguish it from the regular monthly
// payroll payment.
// Beneficiary visibility: khi ẩn, khối "Thông tin chuyển khoản" bị bỏ và thay
// bằng câu thông báo đây là sao kê.
func BuildSaoKeEmailBodies(forMonth, dueDate, totalCollect string, bank config.TransferBankInfo) (htmlBody, textBody string) {
	escapedMonth := html.EscapeString(forMonth)
	escapedDueDate := html.EscapeString(dueDate)
	escapedTotalCollect := html.EscapeString(totalCollect)

	// Khối chuyển khoản được nối chuỗi (không đi qua Sprintf) nên % giữ nguyên.
	var bankBlockHTML, bankText string
	if !bank.Hidden {
		bankBlockHTML = `
                <tr>
                  <td style="padding:24px 32px 0;">
                    <p style="margin:0 0 12px;color:#111827;font-size:16px;line-height:24px;font-weight:700;">Thông tin chuyển khoản</p>
                    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e6eaf2;border-radius:14px;">
                      <tr>
                        <td style="padding:14px 18px;border-bottom:1px solid #edf2f7;color:#64748b;font-size:14px;line-height:20px;">Chủ tài khoản</td>
                        <td align="right" style="padding:14px 18px;border-bottom:1px solid #edf2f7;color:#111827;font-size:14px;line-height:20px;font-weight:700;">` + html.EscapeString(bank.Holder) + `</td>
                      </tr>
                      <tr>
                        <td style="padding:14px 18px;border-bottom:1px solid #edf2f7;color:#64748b;font-size:14px;line-height:20px;">Số tài khoản</td>
                        <td align="right" style="padding:14px 18px;border-bottom:1px solid #edf2f7;color:#111827;font-size:14px;line-height:20px;font-weight:700;">` + html.EscapeString(bank.Number) + `</td>
                      </tr>
                      <tr>
                        <td style="padding:14px 18px;color:#64748b;font-size:14px;line-height:20px;">Ngân hàng</td>
                        <td align="right" style="padding:14px 18px;color:#111827;font-size:14px;line-height:20px;font-weight:700;">` + html.EscapeString(bank.Name) + `</td>
                      </tr>
                    </table>
                  </td>
                </tr>`
		bankText = `

Thông tin chuyển khoản:
Chủ tài khoản: ` + bank.Holder + `
Số tài khoản: ` + bank.Number + `
Ngân hàng: ` + bank.Name
	} else {
		bankBlockHTML = `
                <tr>
                  <td style="padding:24px 32px 0;">
                    <p style="margin:0;color:#334155;font-size:15px;line-height:24px;">Đây là sao kê dịch vụ để Quý Công ty tiện theo dõi và đối chiếu.</p>
                  </td>
                </tr>`
		bankText = ""
	}

	htmlBody = fmt.Sprintf(`<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sao kê dịch vụ Nhận lương sớm 24/7</title>
</head>
<body style="margin:0;padding:0;background:#f5f7fb;color:#172033;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Arial,sans-serif;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f5f7fb;margin:0;padding:28px 12px;">
    <tr>
      <td align="center">
        <table role="presentation" width="640" cellpadding="0" cellspacing="0" style="width:100%%;max-width:640px;border-collapse:separate;border-spacing:0;">
          <tr>
            <td style="padding:0 0 14px;">
              <img src="https://tingting.vip/email-banner.jpg?v=20260709" alt="TingTing Soft" width="640" style="display:block;width:100%%;max-width:640px;height:auto;border:0;border-radius:16px;">
            </td>
          </tr>
          <tr>
            <td style="background:#ffffff;border:1px solid #e6eaf2;border-radius:16px;padding:0;box-shadow:0 10px 30px rgba(15,23,42,0.06);">
              <table role="presentation" width="100%%" cellpadding="0" cellspacing="0">
                <tr>
                  <td style="padding:28px 32px 8px;">
                    <p style="margin:0 0 8px;color:#64748b;font-size:13px;line-height:20px;font-weight:600;letter-spacing:.02em;text-transform:uppercase;">Sao kê dịch vụ</p>
                    <h1 style="margin:0;color:#111827;font-size:24px;line-height:32px;font-weight:700;">Nhận lương sớm 24/7 - %s</h1>
                  </td>
                </tr>
                <tr>
                  <td style="padding:8px 32px 0;">
                    <p style="margin:0;color:#334155;font-size:15px;line-height:24px;">Kính gửi <strong style="color:#111827;">CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP</strong>,</p>
                    <p style="margin:16px 0 0;color:#334155;font-size:15px;line-height:24px;">TING TING SOFT gửi đính kèm sao kê dịch vụ Nhận lương sớm 24/7 để Quý Công ty tiện theo dõi và đối chiếu các khoản đã giải ngân trước hạn trong tháng.</p>
                  </td>
                </tr>
                <tr>
                  <td style="padding:24px 32px 0;">
                    <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:14px;">
                      <tr>
                        <td style="padding:18px 20px;">
                          <p style="margin:0;color:#64748b;font-size:13px;line-height:18px;">Tổng tiền thanh toán</p>
                          <p style="margin:4px 0 0;color:#0f172a;font-size:28px;line-height:36px;font-weight:800;">%s</p>
                        </td>
                        <td align="right" style="padding:18px 20px;">
                          <p style="margin:0;color:#64748b;font-size:13px;line-height:18px;">Hạn thanh toán</p>
                          <p style="margin:4px 0 0;color:#0f172a;font-size:16px;line-height:24px;font-weight:700;">%s</p>
                        </td>
                      </tr>
                    </table>
                  </td>
                </tr>
%s
                <tr>
                  <td style="padding:24px 32px 30px;">
                    <p style="margin:0;color:#334155;font-size:15px;line-height:24px;">Kính mong Quý Công ty kiểm tra và thanh toán trước ngày <strong style="color:#111827;">%s</strong>. Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.</p>
                    <p style="margin:20px 0 0;color:#111827;font-size:15px;line-height:24px;font-weight:700;">TING TING SOFT</p>
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:16px 8px 0;text-align:center;color:#94a3b8;font-size:12px;line-height:18px;">Email tự động từ TingTing. Vui lòng phản hồi nếu Quý Công ty cần hỗ trợ đối chiếu.</td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, escapedMonth, escapedTotalCollect, escapedDueDate, bankBlockHTML, escapedDueDate)

	textBody = fmt.Sprintf(`Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,

TING TING SOFT gửi đính kèm sao kê dịch vụ Nhận lương sớm 24/7 cho tháng %s để Quý Công ty tiện theo dõi và đối chiếu.

Đây là tổng hợp các khoản đã giải ngân trước hạn cho người lao động trong tháng, riêng biệt với kỳ trả lương chính thức.

Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.

Tổng tiền thanh toán: %s%s

Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.

Trân trọng,
TING TING SOFT`, forMonth, dueDate, totalCollect, bankText)

	return htmlBody, textBody
}
