package attendance

import "strings"

// ClassifyAttemptError maps a Vietnamese validation error message from the
// CheckIn/CheckOut service to a stable reason category for aggregation.
// The classifier matches on substrings of the canonical messages defined in
// attendance_service.go. Unknown messages fall through to "other".
//
// Category list — keep in sync with any message changes in attendance_service.go:
//
//   - geofence_not_configured: "Chưa cấu hình vị trí vào làm cho dự án"
//   - geofence_outside:        "Bạn đang ở ngoài khu vực chấm công"
//   - check_in_not_enabled:   "Bạn chưa được cấp quyền chấm công"
//   - already_checked_in:      "Bạn đã vào làm trong ngày hôm nay rồi"
//   - shift_not_configured:   "Chưa cấu hình ca làm việc cho vị trí này"
//   - check_in_window:         "Giờ vào làm không hợp lệ" (check-in outside ±1h)
//   - check_out_window:        contains "Chỉ có thể tan ca từ" or "Đã quá giờ tan ca"
//   - already_checked_out:     "Bạn đã tan ca rồi"
//   - already_auto_rejected:   "Ca làm việc đã bị tự động từ chối"
//   - orphaned:                "Ca làm việc đã quá hạn tan ca"
//   - not_flexible_project:    "không hỗ trợ chấm công linh hoạt"
//   - no_flexible_project:     "Bạn không thuộc dự án linh hoạt nào"
//   - multiple_flexible:       "Bạn thuộc nhiều dự án linh hoạt"
//   - no_attendance:          "Không tìm thấy thông tin vào làm hợp lệ"
//   - check_in_disabled:       "Chấm công đã bị vô hiệu hóa"
//   - other:                   any unrecognized message
func ClassifyAttemptError(msg string) string {
	classifiers := []struct {
		substr   string
		category string
	}{
		{"Chưa cấu hình vị trí vào làm", "geofence_not_configured"},
		{"ngoài khu vực chấm công", "geofence_outside"},
		{"Bạn chưa được cấp quyền chấm công", "check_in_not_enabled"},
		{"Bạn đã vào làm trong ngày", "already_checked_in"},
		{"Chưa cấu hình ca làm việc", "shift_not_configured"},
		{"Giờ vào làm không hợp lệ", "check_in_window"},
		{"Chỉ có thể tan ca từ", "check_out_window"},
		{"Đã quá giờ tan ca", "check_out_window"},
		{"Bạn đã tan ca rồi", "already_checked_out"},
		{"tự động từ chối", "already_auto_rejected"},
		{"quá hạn tan ca", "orphaned"},
		{"không hỗ trợ chấm công linh hoạt", "not_flexible_project"},
		{"Bạn không thuộc dự án linh hoạt", "no_flexible_project"},
		{"nhiều dự án linh hoạt", "multiple_flexible"},
		{"Không tìm thấy thông tin vào làm", "no_attendance"},
		{"Chấm công đã bị vô hiệu hóa", "check_in_disabled"},
	}

	for _, c := range classifiers {
		if strings.Contains(msg, c.substr) {
			return c.category
		}
	}
	return "other"
}
