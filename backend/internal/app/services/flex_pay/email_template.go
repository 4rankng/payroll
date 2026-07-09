package flex_pay

import "fmt"

// BuildSaoKeEmailBodies generates the HTML and text bodies for the FlexPay sao ke
// reconciliation email. Shared by the manual handler and the automated cron job.
//
// Wording is branded "Nhận lương sớm 24/7" (the early-salary / advance product)
// so partners can distinguish it from the regular monthly payroll payment.
func BuildSaoKeEmailBodies(forMonth, dueDate, totalCollect string) (htmlBody, textBody string) {
	htmlBody = fmt.Sprintf(`<p>Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,</p>
<p>TingTing gửi đính kèm <strong>sao kê dịch vụ Nhận lương sớm 24/7</strong> cho tháng %s để Quý Công ty tiện theo dõi và đối chiếu.</p>
<p>Đây là tổng hợp các khoản đã giải ngân trước hạn cho người lao động trong tháng, riêng biệt với kỳ trả lương chính thức.</p>
<p>Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.</p>
<p><strong>Tổng tiền thanh toán: %s</strong></p>
<p><strong>Thông tin chuyển khoản:</strong><br>
Chủ tài khoản: NGUYEN VIET DUNG<br>
Số tài khoản: 1357210887<br>
Ngân hàng: TECHCOMBANK</p>
<p>Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ của TingTing.</p>
<p>Trân trọng,<br>Dịch vụ Nhận lương sớm 24/7 - TingTing</p>`, forMonth, dueDate, totalCollect)

	textBody = fmt.Sprintf(`Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,

TingTing gửi đính kèm sao kê dịch vụ Nhận lương sớm 24/7 cho tháng %s để Quý Công ty tiện theo dõi và đối chiếu.

Đây là tổng hợp các khoản đã giải ngân trước hạn cho người lao động trong tháng, riêng biệt với kỳ trả lương chính thức.

Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.

Tổng tiền thanh toán: %s

Thông tin chuyển khoản:
Chủ tài khoản: NGUYEN VIET DUNG
Số tài khoản: 1357210887
Ngân hàng: TECHCOMBANK

Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ của TingTing.

Trân trọng,
Dịch vụ Nhận lương sớm 24/7 - TingTing`, forMonth, dueDate, totalCollect)

	return htmlBody, textBody
}
