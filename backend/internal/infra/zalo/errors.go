package zalo

// Zalo ZNS business error codes, as documented in the ZBS spec (Part VIII) and
// mirrored in the PHP port's vfic_zns_error_message map. These are returned in
// the JSON body's top-level "error" field (NOT the HTTP status, which is
// typically 200 even on business errors).
const (
	ErrOK                  = 0      // Success
	ErrUnknown             = -100   // Lỗi không xác định
	ErrInvalidPhone        = -108   // Số điện thoại không hợp lệ
	ErrInsufficientBal     = -115   // Tài khoản ZBS không đủ số dư
	ErrNoZaloAccount       = -118   // Số điện thoại chưa liên kết tài khoản Zalo
	ErrOANoPermission      = -120   // OA chưa được cấp quyền tính năng này
	ErrBadAccessToken      = -124   // Access token không hợp lệ (triggers one retry)
	ErrDevModeLimit        = -126   // Hết hạn mức (development mode)
	ErrTemplateTestOnly    = -127   // Template test chỉ gửi được cho quản trị viên OA
	ErrOANoPhoneSend       = -135   // OA chưa có quyền gửi tin SĐT
	ErrNeedZBSConnect      = -136   // Cần kết nối ZBS Account
	ErrZBSChargeFailed     = -137   // Thanh toán ZBS thất bại
	ErrUserOptedOut        = -139   // Người dùng từ chối nhận loại tin này
	ErrUserNotEligible     = -140   // Người dùng không đủ điều kiện nhận tin
	ErrUserBlockedOA       = -141   // Người dùng từ chối nhận tin từ OA
	ErrDailyQuotaPhone     = -144   // Vượt giới hạn gửi tin SĐT trong ngày
	ErrDailyQuotaTpl       = -147   // Template vượt giới hạn gửi trong ngày
	ErrBadTemplateID       = -109   // Template ID không hợp lệ
	ErrTemplateNotAppr     = -131   // Template chưa được phê duyệt
	ErrMissingParam        = -1122  // Template thiếu tham số
	ErrParamOverCap        = -1121  // Tham số vượt giới hạn ký tự
	ErrInvalidRefreshToken = -14014 // Refresh token không hợp lệ (đã dùng/hết hạn) — cần dán lại
)

// ErrorMessage returns a Vietnamese human-readable description for a Zalo error
// code. Unknown codes produce a generic "Lỗi ZNS #<code>".
func ErrorMessage(code int) string {
	switch code {
	case ErrOK:
		return "Thành công"
	case ErrUnknown:
		return "Lỗi không xác định"
	case ErrInvalidPhone:
		return "Số điện thoại không hợp lệ"
	case ErrInsufficientBal:
		return "Tài khoản ZBS không đủ số dư"
	case ErrNoZaloAccount:
		return "Số điện thoại chưa liên kết tài khoản Zalo"
	case ErrOANoPermission:
		return "OA chưa được cấp quyền tính năng này"
	case ErrBadAccessToken:
		return "Access token không hợp lệ"
	case ErrInvalidRefreshToken:
		return "Refresh token không hợp lệ hoặc đã được dùng — dán lại cặp token mới từ Zalo OA Console"
	case ErrDevModeLimit:
		return "Hết hạn mức (development mode)"
	case ErrTemplateTestOnly:
		return "Template test chỉ gửi được cho quản trị viên OA"
	case ErrOANoPhoneSend:
		return "OA chưa có quyền gửi tin SĐT"
	case ErrNeedZBSConnect:
		return "Cần kết nối ZBS Account"
	case ErrZBSChargeFailed:
		return "Thanh toán ZBS thất bại (không đủ số dư)"
	case ErrUserOptedOut:
		return "Người dùng từ chối nhận loại tin này"
	case ErrUserNotEligible:
		return "Người dùng không đủ điều kiện nhận tin"
	case ErrUserBlockedOA:
		return "Người dùng từ chối nhận tin từ OA"
	case ErrDailyQuotaPhone:
		return "Vượt giới hạn gửi tin SĐT trong ngày"
	case ErrDailyQuotaTpl:
		return "Template vượt giới hạn gửi trong ngày"
	case ErrBadTemplateID:
		return "Template ID không hợp lệ"
	case ErrTemplateNotAppr:
		return "Template chưa được phê duyệt"
	case ErrMissingParam:
		return "Template thiếu tham số"
	case ErrParamOverCap:
		return "Tham số vượt giới hạn ký tự"
	default:
		return "Lỗi ZNS #" + itoa(code)
	}
}

// itoa avoids importing strconv just for one error path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
