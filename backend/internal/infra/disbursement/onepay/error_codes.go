package onepay

import "fmt"

// ErrorMessagesVI maps OnePay response_code → user-friendly Vietnamese
// message. Codes pulled from OnePay Payout API documentation section III.1.
//
// "00" is the documented success code — included so MessageVI is total over
// the values the provider returns.
//
// Note: code 23 is not documented by OnePay. It falls through to the
// generic "Lỗi không xác định" fallback.
var ErrorMessagesVI = map[string]string{
	"00":  "Thành công",
	"01":  "Giao dịch timeout",
	"02":  "Trạng thái thẻ không hợp lệ",
	"03":  "Thẻ chưa đăng ký dịch vụ",
	"04":  "Không xác thực được thông tin thẻ",
	"05":  "Không đủ số dư",
	"06":  "Trạng thái tài khoản không hợp lệ",
	"07":  "Giao dịch bị trùng",
	"08":  "Mã giao dịch không hợp lệ",
	"09":  "Số tiền không hợp lệ",
	"10":  "Thông tin không hợp lệ",
	"12":  "Mã ngân hàng không hợp lệ",
	"13":  "Mã ngân hàng không hợp lệ",
	"14":  "Thông tin thẻ không hợp lệ",
	"15":  "Thông tin tài khoản không hợp lệ",
	"16":  "Mã tiền tệ không hợp lệ",
	"17":  "Nội dung chuyển tiền quá dài",
	"18":  "Không tìm thấy operator",
	"19":  "Không tìm thấy ngân hàng xử lý",
	"20":  "Không tìm thấy đơn vị",
	"21":  "Thông số không hợp lệ",
	"22":  "Giao dịch trùng mã ref",
	"23":  "ID người dùng RSA không hợp lệ",
	"30":  "Trạng thái giao dịch không hợp lệ",
	"31":  "Không thể cập nhật giao dịch",
	"32":  "Không thể tạo giao dịch",
	"34":  "Ngân hàng xử lý đã thay đổi, vui lòng tạo lại giao dịch",
	"38":  "Không tìm thấy giao dịch",
	"39":  "Không tìm thấy giao dịch",
	"40":  "Quá hạn mức tối thiểu trên lần giao dịch",
	"41":  "Quá hạn mức tối đa trên lần giao dịch",
	"42":  "Quá hạn mức tối đa trên ngày",
	"45":  "Không hợp lệ số dư",
	"50":  "Quá hạn số lượng giao dịch trong lô",
	"70":  "Thông tin tài khoản đơn vị không hợp lệ",
	"71":  "Trạng thái đơn vị không hợp lệ",
	"72":  "Thông tin xác thực đơn vị không hợp lệ",
	"73":  "Tài khoản đã bị chặn",
	"80":  "Lỗi cơ sở dữ liệu",
	"81":  "Không có ngân hàng hỗ trợ",
	"82":  "Không có ngân hàng nhận",
	"83":  "Ngân hàng xử lý không hợp lệ",
	"86":  "Chức năng tạm thời đóng",
	"87":  "Service xác thực không hợp lệ",
	"88":  "Region xác thực không hợp lệ",
	"89":  "OWS Request xác thực không hợp lệ",
	"90":  "Thuật toán ký không hợp lệ",
	"91":  "Hệ thống ngân hàng gián đoạn",
	"92":  "Chữ ký xác thực không hợp lệ",
	"93":  "Hệ thống ngân hàng gián đoạn",
	"94":  "Hệ thống ngân hàng gián đoạn",
	"95":  "Hệ thống ngân hàng gián đoạn",
	"96":  "Hệ thống ngân hàng gián đoạn",
	"97":  "Access key id không hợp lệ",
	"98":  "Authorization hết hạn",
	"99":  "Hệ thống ngân hàng gián đoạn",
	"100": "Hệ thống tạm thời gián đoạn",
	"101": "Mật khẩu cũ không đúng",
	"102": "Tham số truyền vào không hợp lệ",
	"ZZ":  "Giao dịch thất bại",
}

// ErrorCodeNames maps OnePay response_code → the official code name from docs.
var ErrorCodeNames = map[string]string{
	"00":  "SUCCESS",
	"01":  "TXN_PENDING",
	"02":  "INVALID_CARD_STATE",
	"03":  "CARD_NOT_REGISTER",
	"04":  "DO_NOT_VERIFY_CARD",
	"05":  "INSUFFICENT_FUND",
	"06":  "INVALID_ACCOUNT_STATE",
	"07":  "DUPLICATE_TXN",
	"08":  "INVALID_TRANSACTION_ID",
	"09":  "INVALID_AMOUNT",
	"10":  "INVALID_FUND_TRANSFER_INFO",
	"12":  "INVALID_BANK_CODE",
	"13":  "INVALID_BANK_CODE",
	"14":  "INVALID_CARD_INFO",
	"15":  "INVALID_ACCOUNT_INFO",
	"16":  "INVALID_CURRENCY_CODE",
	"17":  "REMARK_MAX_LENGTH",
	"18":  "NO_OPERATOR_FOUND",
	"19":  "NO_BANK_PROCESS_FOUND",
	"20":  "PARTNER_NOT_FOUND",
	"21":  "INVALID_PARAMETERS",
	"22":  "DUPLICATE_TXN_REF",
	"23":  "INVALID_RSA_USER_ID",
	"30":  "INVALID_TXN_STATE",
	"31":  "CAN_NOT_UPDATE_TXN",
	"32":  "CAN_NOT_INSERT_TXN",
	"34":  "CHANGED_BANK_PROCESS",
	"38":  "TXN_NOT_FOUND",
	"39":  "NO_DATA_FOUND",
	"40":  "LIMIT_MIN_AMOUNT_PER_TXN",
	"41":  "LIMIT_MAX_AMOUNT_PER_TXN",
	"42":  "LIMIT_MAX_AMOUNT_PER_DAY",
	"45":  "INVALID_BALANCE",
	"50":  "LIMIT_MAX_TXN_PER_BATCH",
	"70":  "INVALID_MERCHANT_INFO",
	"71":  "INVALID_MERCHANT_STATE",
	"72":  "INVALID_MERCHANT_AUTHORIZATION",
	"73":  "INACTIVE_OPERATOR",
	"80":  "DB_SQL_ERROR",
	"81":  "NO_BANK_PROCESS_FOUND",
	"82":  "NO_BANK_MAP_FOUND",
	"83":  "INVALID_BANK_PROCESS",
	"86":  "FUNCITON_TEMPORARILY_NOT_AVAILABLE",
	"87":  "INVALID_AUTHORIZATION_SERVICE",
	"88":  "INVALID_AUTHORIZATION_REGION",
	"89":  "INVALID_AUTHORIZATION_OWS_REQUEST",
	"90":  "INVALID_AUTHORIZATION_ALGORITHM",
	"91":  "BANK_SYSTEM_ERROR",
	"92":  "INVALID_AUTHORIZATION_SIGNATURE",
	"93":  "BANK_SYSTEM_ERROR",
	"94":  "BANK_SYSTEM_ERROR",
	"95":  "BANK_SYSTEM_ERROR",
	"96":  "BANK_SYSTEM_ERROR",
	"97":  "INVALID_AUTHORIZATION_ACCESS_KEY_ID",
	"98":  "EXPIRED_AUTHORIZATION",
	"99":  "BANK_SYSTEM_ERROR",
	"100": "INTERNAL_SERVER_ERROR",
	"101": "OPERATOR_PASS_INVALID",
	"102": "INVALID_PARAMS",
	"ZZ":  "TXN_FAILED",
}

// MessageVI returns the Vietnamese translation for a OnePay error code,
// or a generic fallback for unknown codes. Empty input yields "".
func MessageVI(code string) string {
	if code == "" {
		return ""
	}
	if msg, ok := ErrorMessagesVI[code]; ok {
		return msg
	}
	return fmt.Sprintf("Lỗi không xác định (mã: %s)", code)
}

// CodeName returns the official OnePay code name for a response code.
func CodeName(code string) string {
	if name, ok := ErrorCodeNames[code]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN_%s", code)
}
