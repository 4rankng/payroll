package flex_pay

import "fmt"

// BuildSaoKeEmailBodies generates the HTML and text bodies for the FlexPay sao ke
// reconciliation email. Shared by the manual handler and the automated cron job.
//
// Wording is branded for the "Nhận lương sớm 24/7" early-salary service by
// TING TING SOFT, so partners can distinguish it from the regular monthly
// payroll payment.
func BuildSaoKeEmailBodies(forMonth, dueDate, totalCollect string) (htmlBody, textBody string) {
	htmlBody = fmt.Sprintf(`<p style="text-align:left;"><img src="https://tingting.vip/email-banner.jpg" alt="Ting Ting Soft" width="600" style="display:block;width:100%%;max-width:600px;height:auto;border:0;"></p>
<p>Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,</p>
<p>TING TING SOFT gửi đính kèm <strong>sao kê dịch vụ Nhận lương sớm 24/7</strong> cho tháng %s để Quý Công ty tiện theo dõi và đối chiếu.</p>
<p>Đây là tổng hợp các khoản đã giải ngân trước hạn cho người lao động trong tháng, riêng biệt với kỳ trả lương chính thức.</p>
<p>Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.</p>
<p><strong>Tổng tiền thanh toán: %s</strong></p>
<p><strong>Thông tin chuyển khoản:</strong><br>
Chủ tài khoản: NGUYEN VIET DUNG<br>
Số tài khoản: 1357210887<br>
Ngân hàng: TECHCOMBANK</p>
<p>Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.</p>
<p>Trân trọng,<br>TING TING SOFT</p>`, forMonth, dueDate, totalCollect)

	textBody = fmt.Sprintf(`Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,

TING TING SOFT gửi đính kèm sao kê dịch vụ Nhận lương sớm 24/7 cho tháng %s để Quý Công ty tiện theo dõi và đối chiếu.

Đây là tổng hợp các khoản đã giải ngân trước hạn cho người lao động trong tháng, riêng biệt với kỳ trả lương chính thức.

Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.

Tổng tiền thanh toán: %s

Thông tin chuyển khoản:
Chủ tài khoản: NGUYEN VIET DUNG
Số tài khoản: 1357210887
Ngân hàng: TECHCOMBANK

Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ.

Trân trọng,
TING TING SOFT`, forMonth, dueDate, totalCollect)

	return htmlBody, textBody
}
