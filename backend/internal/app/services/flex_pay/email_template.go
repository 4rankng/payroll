package flex_pay

import "fmt"

// BuildSaoKeEmailBodies generates the HTML and text bodies for the FlexPay sao ke
// reconciliation email. Shared by the manual handler and the automated cron job.
func BuildSaoKeEmailBodies(forMonth, dueDate, totalCollect string) (htmlBody, textBody string) {
	htmlBody = fmt.Sprintf(`<p>Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,</p>
<p>Chi tiết sao kê thanh toán ứng lương cho tháng %s được đính kèm trong email này để Quý Công ty tiện theo dõi và đối chiếu.</p>
<p>Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.</p>
<p><strong>Tổng tiền thanh toán: %s</strong></p>
<p><strong>Thông tin chuyển khoản:</strong><br>
Chủ tài khoản: NGUYEN VIET DUNG<br>
Số tài khoản: 1357210887<br>
Ngân hàng: TECHCOMBANK</p>
<p>Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ của TingTing.</p>
<p>Trân trọng,<br>Dịch vụ thanh toán TingTing</p>`, forMonth, dueDate, totalCollect)

	textBody = fmt.Sprintf(`Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,

Chi tiết sao kê thanh toán ứng lương cho tháng %s được đính kèm trong email này để Quý Công ty tiện theo dõi và đối chiếu.

Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.

Tổng tiền thanh toán: %s

Thông tin chuyển khoản:
Chủ tài khoản: NGUYEN VIET DUNG
Số tài khoản: 1357210887
Ngân hàng: TECHCOMBANK

Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ của TingTing.

Trân trọng,
Dịch vụ thanh toán TingTing`, forMonth, dueDate, totalCollect)

	return htmlBody, textBody
}
