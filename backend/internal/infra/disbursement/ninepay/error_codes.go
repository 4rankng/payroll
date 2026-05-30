package ninepay

import "fmt"

// ErrorMessagesVI maps 9pay's documented error codes to user-facing
// Vietnamese messages. The codes themselves come straight from the
// integration spec; the translations are written for non-technical
// employees who see them in notifications and error_message columns.
//
// "000" is the documented success code — included so MessageVI is
// total over the values the provider returns.
var ErrorMessagesVI = map[string]string{
	"000":  "Thành công",
	"318":  "Dữ liệu không hợp lệ",
	"431":  "Số dư tài khoản chi hộ không đủ",
	"666":  "Loại tài khoản không hợp lệ",
	"702":  "Giao dịch đã kết thúc (request_id trùng lặp)",
	"1001": "Không tìm thấy thông tin tài khoản",
	"1002": "Thông tin xác thực không hợp lệ",
	"1004": "Thông tin tài khoản ngân hàng không hợp lệ",
	"1005": "Dịch vụ chi hộ chưa được kích hoạt",
	"1006": "Không tìm thấy thông tin ngân hàng",
	"1007": "Số tiền vượt quá hạn mức cho phép",
	"1008": "Tên tài khoản không hợp lệ",
}

// MessageVI returns the Vietnamese translation for a 9pay error code,
// or a generic "unknown error (code: X)" fallback for codes not in the
// map. Empty input yields "" so callers can short-circuit on success.
func MessageVI(code string) string {
	if code == "" {
		return ""
	}
	if msg, ok := ErrorMessagesVI[code]; ok {
		return msg
	}
	return fmt.Sprintf("Lỗi không xác định (mã: %s)", code)
}

// TranslateError satisfies infrastructure.ErrorTranslator on *Provider
// and *queuedProvider — service-layer code resolves the Vietnamese
// message via the registry instead of importing this package directly.
func (p *Provider) TranslateError(code string) string { return MessageVI(code) }

// TranslateError on the queued wrapper delegates to the inner Provider.
func (q *queuedProvider) TranslateError(code string) string { return q.inner.TranslateError(code) }
