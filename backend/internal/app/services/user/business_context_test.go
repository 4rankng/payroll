package user

import (
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestNewBusinessContextMapper(t *testing.T) {
	mapper := NewBusinessContextMapper()

	assert.NotNil(t, mapper)
	assert.NotNil(t, mapper.contextMap)
	assert.NotNil(t, mapper.impactMap)
	assert.NotEmpty(t, mapper.defaultTpl)
}

func TestBusinessContextMapper_GetBusinessContext_UserLogin_Success(t *testing.T) {
	mapper := NewBusinessContextMapper()

	log := &domain.AuditLog{
		UserID:    1,
		Message:   "Nguyễn Văn A đã đăng nhập vào hệ thống",
		CreatedAt: time.Now(),
	}

	result := mapper.GetBusinessContext(log)
	assert.Equal(t, "Nguyễn Văn A đã đăng nhập vào hệ thống", result)
}

func TestBusinessContextMapper_GetBusinessContext_MessageBased(t *testing.T) {
	mapper := NewBusinessContextMapper()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{"Timesheet Create", "Nguyễn Văn A đã tạo bản ghi chấm công", "Nguyễn Văn A đã tạo bản ghi chấm công"},
		{"Payroll Update", "Trần Thị B đã cập nhật tính toán lương", "Trần Thị B đã cập nhật tính toán lương"},
		{"Project Delete", "Lê Văn C đã xóa dự án", "Lê Văn C đã xóa dự án"},
		{"User Login", "Admin đã đăng nhập vào hệ thống", "Admin đã đăng nhập vào hệ thống"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &domain.AuditLog{
				UserID:    1,
				Message:   tt.message,
				CreatedAt: time.Now(),
			}

			result := mapper.GetBusinessContext(log)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessContextMapper_GetImpactLevel_MessageBased(t *testing.T) {
	mapper := NewBusinessContextMapper()

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{"High Impact - Delete", "Nguyễn Văn A đã xóa bản ghi chấm công", "HIGH"},
		{"High Impact - Approve", "Trần Thị B đã duyệt chấm công để xử lý lương", "HIGH"},
		{"High Impact - Create Payroll", "Lê Văn C đã tạo bản ghi tính lương", "HIGH"},
		{"Medium Impact - Update", "Phạm Thị D đã cập nhật thông tin nhân viên", "MEDIUM"},
		{"Low Impact - View", "Admin đã xem chi tiết chấm công", "LOW"},
		{"Low Impact - Login", "Người dùng đã đăng nhập vào hệ thống", "LOW"},
		{"Default Medium", "Some generic action", "MEDIUM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &domain.AuditLog{
				UserID:    1,
				Message:   tt.message,
				CreatedAt: time.Now(),
			}

			result := mapper.GetImpactLevel(log)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessContextMapper_GetSupportedEntityTypes(t *testing.T) {
	mapper := NewBusinessContextMapper()

	types := mapper.GetSupportedEntityTypes()

	assert.NotEmpty(t, types)
	assert.Contains(t, types, "timesheet")
	assert.Contains(t, types, "payroll")
	assert.Contains(t, types, "project")
	assert.Contains(t, types, "employee")
	assert.Contains(t, types, "user")
}

func TestBusinessContextMapper_GetSupportedActionsForEntity(t *testing.T) {
	mapper := NewBusinessContextMapper()

	actions := mapper.GetSupportedActionsForEntity("timesheet")

	assert.NotEmpty(t, actions)
	assert.Contains(t, actions, "CREATE")
	assert.Contains(t, actions, "UPDATE")
	assert.Contains(t, actions, "DELETE")
	assert.NotContains(t, actions, "DEFAULT")
}

func TestBusinessContextMapper_GetSupportedActionsForEntity_UnknownEntity(t *testing.T) {
	mapper := NewBusinessContextMapper()

	actions := mapper.GetSupportedActionsForEntity("unknown_entity")

	assert.Empty(t, actions)
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", ""},
		{"Single char", "a", "A"},
		{"Multiple chars", "hello", "Hello"},
		{"Already capitalized", "Hello", "Hello"},
		{"Unicode", "được", "Được"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalizeFirst(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildContextMap(t *testing.T) {
	contextMap := buildContextMap()

	assert.NotNil(t, contextMap)
	assert.NotEmpty(t, contextMap)

	// Verify structure
	assert.Contains(t, contextMap, "timesheet")
	assert.Contains(t, contextMap, "payroll")
	assert.Contains(t, contextMap, "user")
}

func TestBuildImpactMap(t *testing.T) {
	impactMap := buildImpactMap()

	assert.NotNil(t, impactMap)
	assert.NotEmpty(t, impactMap)

	// Verify structure
	assert.Contains(t, impactMap, "ACTION:DELETE")
	assert.Contains(t, impactMap, "ENTITY:payroll")
	assert.Contains(t, impactMap, "payroll:CREATE")
}
