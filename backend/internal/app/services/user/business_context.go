package user

import (
	"strings"
	"unicode"

	"api-server/internal/domain"
)

// BusinessContextMap defines entity-action to description mapping
type BusinessContextMap map[string]map[string]string

// ImpactLevelMap defines entity/action to impact level mapping
type ImpactLevelMap map[string]string

// BusinessContextMapper provides efficient business context mapping using pre-built maps
type BusinessContextMapper struct {
	contextMap BusinessContextMap
	impactMap  ImpactLevelMap
	defaultTpl string
}

// NewBusinessContextMapper creates a new mapper with pre-defined context mappings
func NewBusinessContextMapper() *BusinessContextMapper {
	return &BusinessContextMapper{
		contextMap: buildContextMap(),
		impactMap:  buildImpactMap(),
		defaultTpl: "Thực hiện %s trên %s",
	}
}

// GetBusinessContext returns a meaningful business context description for an audit log entry
// Since audit logs now contain complete messages with user names, this simplifies to just returning the message
func (bcm *BusinessContextMapper) GetBusinessContext(log *domain.AuditLog) string {
	// The new audit pattern stores complete messages with user names included
	// So we can just return the message directly
	return log.Message
}

// GetImpactLevel determines the business impact level of an activity
// Since audit logs now contain complete messages with context, this simplifies to a basic assessment
func (bcm *BusinessContextMapper) GetImpactLevel(log *domain.AuditLog) string {
	// Basic impact assessment based on message content
	message := log.Message

	// High impact activities
	if contains(message, []string{"tạo", "xóa", "phê duyệt", "duyệt", "từ chối", "thanh toán", "đảo ngược"}) {
		return "HIGH"
	}

	// Medium impact activities
	if contains(message, []string{"cập nhật", "sửa", "thay đổi"}) {
		return "MEDIUM"
	}

	// Low impact activities
	if contains(message, []string{"xem", "xuất", "nhập", "đăng nhập", "đăng xuất"}) {
		return "LOW"
	}

	// Default to medium impact
	return "MEDIUM"
}

// buildContextMap creates the pre-compiled context mapping for O(1) lookups
func buildContextMap() BusinessContextMap {
	return BusinessContextMap{
		"timesheet": {
			"CREATE":  "Tạo mới bản ghi chấm công",
			"UPDATE":  "Sửa đổi chi tiết chấm công",
			"APPROVE": "Duyệt chấm công để xử lý lương",
			"REJECT":  "Từ chối chấm công với phản hồi",
			"VIEW":    "Xem chi tiết chấm công",
			"DELETE":  "Xóa bản ghi chấm công",
			"DEFAULT": "Thực hiện %s trên chấm công",
		},
		"payroll": {
			"CREATE":  "Tạo bản ghi tính lương",
			"UPDATE":  "Cập nhật tính toán lương",
			"VIEW":    "Xem lại thông tin lương",
			"DELETE":  "Xóa bản ghi lương",
			"APPROVE": "Duyệt lương để thanh toán",
			"EXPORT":  "Xuất dữ liệu lương",
			"DEFAULT": "Thực hiện %s trên lương",
		},
		"project": {
			"CREATE":  "Tạo dự án mới",
			"UPDATE":  "Cập nhật chi tiết hoặc trạng thái dự án",
			"VIEW":    "Truy cập thông tin dự án",
			"DELETE":  "Xóa dự án",
			"APPROVE": "Duyệt thay đổi dự án",
			"DEFAULT": "Quản lý dự án (%s)",
		},
		"employee": {
			"CREATE":  "Thêm nhân viên mới vào hệ thống",
			"UPDATE":  "Cập nhật thông tin nhân viên",
			"VIEW":    "Xem hồ sơ nhân viên",
			"DELETE":  "Xóa nhân viên khỏi hệ thống",
			"DEFAULT": "Quản lý hồ sơ nhân viên (%s)",
		},
		"project_employee": {
			"CREATE":  "Phân công nhân viên vào dự án",
			"UPDATE":  "Sửa đổi phân công dự án của nhân viên",
			"DELETE":  "Xóa nhân viên khỏi dự án",
			"VIEW":    "Xem lại phân công dự án của nhân viên",
			"DEFAULT": "Quản lý phân công dự án (%s)",
		},
		"payrate": {
			"CREATE":  "Tạo mức lương mới",
			"UPDATE":  "Sửa đổi chi tiết mức lương",
			"APPROVE": "Duyệt thay đổi mức lương",
			"REJECT":  "Từ chối thay đổi mức lương",
			"VIEW":    "Xem lại thông tin mức lương",
			"DELETE":  "Xóa mức lương",
			"DEFAULT": "Quản lý mức lương (%s)",
		},
		"ledger_entry": {
			"CREATE":  "Tạo bút toán tài chính",
			"UPDATE":  "Sửa đổi hồ sơ tài chính",
			"VIEW":    "Xem lại bút toán tài chính",
			"DELETE":  "Xóa bút toán",
			"DEFAULT": "Quản lý hồ sơ tài chính (%s)",
		},
		"import_batch": {
			"CREATE":  "Khởi tạo lô nhập dữ liệu",
			"UPDATE":  "Cập nhật trạng thái lô nhập",
			"VIEW":    "Xem lại lô nhập",
			"DELETE":  "Hủy lô nhập",
			"DEFAULT": "Quản lý nhập dữ liệu (%s)",
		},
		"user": {
			"CREATE":  "Tạo tài khoản người dùng mới",
			"UPDATE":  "Sửa đổi chi tiết tài khoản người dùng",
			"VIEW":    "Truy cập hồ sơ người dùng",
			"DELETE":  "Xóa tài khoản người dùng",
			"LOGIN":   "Đã đăng nhập vào hệ thống",
			"LOGOUT":  "Đã đăng xuất khỏi hệ thống",
			"DEFAULT": "Tài khoản người dùng %s",
		},
	}
}

// buildImpactMap creates the pre-compiled impact level mapping for O(1) lookups
func buildImpactMap() ImpactLevelMap {
	return ImpactLevelMap{
		// Action-based impact (highest priority)
		"ACTION:DELETE":  "high",
		"ACTION:APPROVE": "high",
		"ACTION:REJECT":  "high",
		"ACTION:CREATE":  "medium",
		"ACTION:UPDATE":  "medium",
		"ACTION:VIEW":    "low",
		"ACTION:LOGIN":   "low",
		"ACTION:LOGOUT":  "low",
		"ACTION:EXPORT":  "medium",

		// Entity-based impact (secondary priority)
		"ENTITY:payroll":      "high",
		"ENTITY:ledger_entry": "high",
		"ENTITY:user":         "medium",
		"ENTITY:employee":     "medium",
		"ENTITY:project":      "medium",
		"ENTITY:payrate":      "medium",
		"ENTITY:timesheet":    "low",

		// Specific combinations for fine-grained control
		"payroll:CREATE":      "high",
		"payroll:UPDATE":      "high",
		"ledger_entry:CREATE": "high",
		"ledger_entry:UPDATE": "high",
		"user:DELETE":         "high",
		"employee:DELETE":     "high",
		"timesheet:APPROVE":   "medium",
		"timesheet:REJECT":    "medium",
	}
}

// GetSupportedEntityTypes returns all entity types with context mappings
func (bcm *BusinessContextMapper) GetSupportedEntityTypes() []string {
	types := make([]string, 0, len(bcm.contextMap))
	for entityType := range bcm.contextMap {
		types = append(types, entityType)
	}
	return types
}

// GetSupportedActionsForEntity returns all supported actions for a given entity type
func (bcm *BusinessContextMapper) GetSupportedActionsForEntity(entityType string) []string {
	if entityActions, exists := bcm.contextMap[entityType]; exists {
		actions := make([]string, 0, len(entityActions))
		for action := range entityActions {
			if action != "DEFAULT" {
				actions = append(actions, action)
			}
		}
		return actions
	}
	return []string{}
}

// contains checks if a string contains any of the given substrings
func contains(s string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substring)) {
			return true
		}
	}
	return false
}

// capitalizeFirst capitalizes the first letter of a string (replacement for deprecated strings.Title)
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
