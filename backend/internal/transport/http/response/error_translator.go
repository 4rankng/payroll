package response

import (
	"regexp"
	"strings"

	"api-server/internal/constants"
)

// ErrorTranslator handles the translation of technical errors to user-friendly messages
type ErrorTranslator struct {
	// Map of technical error patterns to user-friendly Vietnamese messages
	technicalErrorMappings map[string]string
	// Map of status replacements for Vietnamese
	statusReplacements map[string]string
}

// NewErrorTranslator creates a new error translator instance
func NewErrorTranslator() *ErrorTranslator {
	return &ErrorTranslator{
		technicalErrorMappings: map[string]string{
			"Field 'paytype' doesn't have a default value":                                        "Lỗi máy chủ nội bộ: Thiếu cấu hình loại lương",
			"Field 'pay_type' doesn't have a default value":                                       "Lỗi máy chủ nội bộ: Thiếu cấu hình loại lương",
			"Field 'payrate' doesn't have a default value":                                        "Lỗi máy chủ nội bộ: Thiếu cấu hình mức lương",
			"Field 'pay_rate' doesn't have a default value":                                       "Lỗi máy chủ nội bộ: Thiếu cấu hình mức lương",
			"Employee is not assigned to this project":                                            "Nhân viên chưa được phân công vào dự án này",
			"No active payrate configuration found":                                               "Không tìm thấy cấu hình mức lương hiện hành cho dự án này",
			"Saturday requires day_type":                                                          "Thứ 7 yêu cầu chỉ định loại ngày: 'ngày thường' hoặc 'ngày nghỉ'",
			"Timesheet entry already exists":                                                      "Đã có bảng chấm công cho nhân viên, dự án, ngày và loại lương này",
			"failed to set payrate for path":                                                      "Cấu hình mức lương không hợp lệ cho loại công việc đã chỉ định",
			"Total hours worked per day cannot exceed 24 hours":                                   "Tổng số giờ làm việc trong ngày không thể vượt quá 24 giờ",
			"current status is 'pending_approval', which cannot transition to 'pending_approval'": constants.MsgTimesheetAlreadyExistsPendingVN,
			"current status is 'approved', which cannot transition to 'pending_approval'":         constants.MsgTimesheetAlreadyExistsApprovedVN,
		},
		statusReplacements: map[string]string{
			"'pending_approval'": "'chờ phê duyệt'",
			"'approved'":         "'đã phê duyệt'",
			"'rejected'":         "'đã từ chối'",
			"'draft'":            "'nháp'",
			"pending_approval":   "chờ phê duyệt",
			"approved":           "đã phê duyệt",
			"rejected":           "đã từ chối",
			"draft":              "nháp",
		},
	}
}

// TranslateError converts technical errors to user-friendly Vietnamese messages
func (t *ErrorTranslator) TranslateError(err error) string {
	errMsg := err.Error()

	// First, clean any array indexing from the error message
	cleanedMsg := t.removeArrayIndexing(errMsg)

	// Check for specific technical error patterns
	for pattern, userFriendlyMsg := range t.technicalErrorMappings {
		if strings.Contains(cleanedMsg, pattern) {
			return userFriendlyMsg
		}
	}

	// Apply general message cleaning
	return t.CleanMessage(cleanedMsg)
}

// removeArrayIndexing removes array indexing patterns from error messages
func (t *ErrorTranslator) removeArrayIndexing(errorMsg string) string {
	// Remove array indexing patterns at the beginning like "timesheet[0]:"
	re1 := regexp.MustCompile(`^timesheet\[\d+\]:\s*`)
	errorMsg = re1.ReplaceAllString(errorMsg, "")

	// Remove any array indexing patterns anywhere in the message like "timesheet[N]:"
	re2 := regexp.MustCompile(`timesheet\[\d+\]:\s*`)
	errorMsg = re2.ReplaceAllString(errorMsg, "")

	// Remove any field indexing patterns like "field[0]:" or similar
	re3 := regexp.MustCompile(`\w+\[\d+\]:\s*`)
	errorMsg = re3.ReplaceAllString(errorMsg, "")

	return strings.TrimSpace(errorMsg)
}

// CleanMessage removes technical array indexing patterns and internal values from error messages
func (t *ErrorTranslator) CleanMessage(message string) string {
	// Remove array indexing patterns like "timesheet[0]:", "field[N]:", etc.
	re := regexp.MustCompile(`\w+\[\d+\]:\s*`)
	message = re.ReplaceAllString(message, "")

	// Also handle the specific case where it appears after a colon like ": timesheet[0]:"
	re2 := regexp.MustCompile(`:\s*\w+\[\d+\]:\s*`)
	message = re2.ReplaceAllString(message, ": ")

	// Replace internal status values with user-friendly Vietnamese
	for internal, userFriendly := range t.statusReplacements {
		message = strings.ReplaceAll(message, internal, userFriendly)
	}

	// Replace technical terms with user-friendly Vietnamese
	message = strings.ReplaceAll(message, "cannot transition to", "không thể chuyển sang")
	message = strings.ReplaceAll(message, "current status is", "trạng thái hiện tại là")
	message = strings.ReplaceAll(message, "which cannot transition to", "không thể chuyển sang")

	return strings.TrimSpace(message)
}

// AddTechnicalErrorMapping allows adding new error mappings at runtime
func (t *ErrorTranslator) AddTechnicalErrorMapping(technicalPattern string, userFriendlyMessage string) {
	t.technicalErrorMappings[technicalPattern] = userFriendlyMessage
}

// GetDefaultTranslator returns a singleton instance of the error translator
var defaultTranslator *ErrorTranslator

func GetErrorTranslator() *ErrorTranslator {
	if defaultTranslator == nil {
		defaultTranslator = NewErrorTranslator()
	}
	return defaultTranslator
}
