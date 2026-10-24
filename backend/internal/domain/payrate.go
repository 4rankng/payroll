package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/pkg/utils"
	"github.com/nqd/flat"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
)

// SkillLevel represents client-defined skill level names
// No fixed values - clients can use any names like:
// "experienced", "tất cả", "có kinh nghiệm", etc.
type SkillLevel string

// PayrateConfiguration represents flexible JSON structure for pay rates
// Internally flattened to path -> value mapping for easy access
// Original JSON: {"skill": {"category": {"sub": 30000}}}
// Flattened: {"skill.category.sub": 30000}
type PayrateConfiguration json.RawMessage

// MarshalJSON implements json.Marshaler interface to return the raw JSON
func (p PayrateConfiguration) MarshalJSON() ([]byte, error) {
	if len(p) == 0 {
		return []byte("{}"), nil
	}
	return []byte(p), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (p *PayrateConfiguration) UnmarshalJSON(data []byte) error {
	*p = PayrateConfiguration(data)
	return nil
}

// Validate performs basic JSON validation for CRUD operations
func (p PayrateConfiguration) Validate() error {
	if len(p) == 0 {
		return NewValidationError("cấu hình mức lương là bắt buộc")
	}

	// Basic JSON structure validation only
	var temp map[string]interface{}
	if err := json.Unmarshal(p, &temp); err != nil {
		return NewValidationError("định dạng JSON không hợp lệ: " + err.Error())
	}

	if len(temp) == 0 {
		return NewValidationError("cấu hình mức lương không thể trống")
	}

	return nil
}

// ValidateDetailed performs comprehensive validation including rate values
// Use this for business logic validation, not CRUD operations
func (p PayrateConfiguration) ValidateDetailed() error {
	if err := p.Validate(); err != nil {
		return err
	}

	flattened, err := p.Flatten()
	if err != nil {
		return NewValidationError("không thể làm phẳng cấu hình: " + err.Error())
	}

	// Validate all values are non-negative integers
	for path, value := range flattened {
		if value < 0 {
			return NewValidationError(fmt.Sprintf("rate value cannot be negative at path: %s", path))
		}
	}

	// Validate overlap for flexible shifts
	if err := p.ValidateFlexibleOverlap(); err != nil {
		return err
	}

	return nil
}

// Flatten converts nested JSON to flat path -> value map using nqd/flat library
// "experienced.weekday.08:00-17:00" -> 32500
func (p PayrateConfiguration) Flatten() (map[string]int, error) {
	if len(p) == 0 {
		return make(map[string]int), nil
	}

	var temp map[string]interface{}
	if err := json.Unmarshal(p, &temp); err != nil {
		return nil, err
	}

	// Use nqd/flat library to flatten the nested structure
	flattened, err := flat.Flatten(temp, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to flatten structure: %w", err)
	}

	// Convert flattened map to map[string]int and validate values
	result := make(map[string]int)
	for path, value := range flattened {
		switch v := value.(type) {
		case float64:
			if v < 0 {
				return nil, NewValidationError(fmt.Sprintf("giá trị mức lương không thể âm tại đường dẫn: %s", path))
			}
			if v != float64(int(v)) {
				return nil, NewValidationError(fmt.Sprintf("giá trị mức lương phải là số nguyên tại đường dẫn: %s", path))
			}
			result[path] = int(v)
		case int:
			if v < 0 {
				return nil, NewValidationError(fmt.Sprintf("giá trị mức lương không thể âm tại đường dẫn: %s", path))
			}
			result[path] = v
		default:
			return nil, NewValidationError(fmt.Sprintf("loại giá trị mức lương không hợp lệ tại đường dẫn: %s, phải là số nguyên", path))
		}
	}

	return result, nil
}

// payrateShift represents a parsed shift for overlap validation.
type payrateShift struct {
	Start    time.Time
	End      time.Time
	Original string // original "HH:MM-HH:MM" for error messages
}

// ValidateFlexibleOverlap validates that shift times in flexible payrate configurations do not overlap.
// Expected format for keys: "{position}.{dayType}.{HH:MM}-{HH:MM}"
func (p PayrateConfiguration) ValidateFlexibleOverlap() error {
	flattened, err := p.Flatten()
	if err != nil {
		return err
	}

	positionShifts := make(map[string][]payrateShift)

	for key := range flattened {
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}

		timeRange := parts[len(parts)-1]
		timeParts := strings.Split(timeRange, "-")
		if len(timeParts) != 2 {
			continue // Not a time range, probably standard payrate
		}

		layout := "15:04"
		start, err := time.Parse(layout, timeParts[0])
		if err != nil {
			return NewValidationError(fmt.Sprintf("thời gian bắt đầu không hợp lệ: %s trong %s", timeParts[0], key))
		}
		end, err := time.Parse(layout, timeParts[1])
		if err != nil {
			return NewValidationError(fmt.Sprintf("thời gian kết thúc không hợp lệ: %s trong %s", timeParts[1], key))
		}

		if end.Before(start) {
			end = end.Add(24 * time.Hour) // Overnight shift
		}

		positionAndDay := strings.Join(parts[:len(parts)-1], ".")
		positionShifts[positionAndDay] = append(positionShifts[positionAndDay], payrateShift{Start: start, End: end, Original: timeRange})
	}

	for pos, shifts := range positionShifts {
		for i := 0; i < len(shifts); i++ {
			for j := i + 1; j < len(shifts); j++ {
				s1, s2 := shifts[i], shifts[j]
				if shiftsOverlap(s1, s2) {
					return NewValidationError(fmt.Sprintf("ca làm việc bị chồng lấn cho %s: %s và %s",
						pos, s1.Original, s2.Original))
				}
			}
		}
	}

	return nil
}

// shiftsOverlap checks whether two shifts overlap, handling overnight shifts correctly.
// An overnight shift (e.g., 22:00-06:00 normalized to 22:00-30:00) may overlap with
// a morning shift (05:00-07:00) that is on the base timeline. We check the non-overnight
// shift against both the original and the +24h shifted version of the overnight shift.
func shiftsOverlap(a, b payrateShift) bool {
	// Standard overlap check on same timeline
	if a.Start.Before(b.End) && b.Start.Before(a.End) {
		return true
	}
	// Check if shifting B by +24h reveals an overlap (B starts "tomorrow" relative to A)
	bShifted := payrateShift{Start: b.Start.Add(24 * time.Hour), End: b.End.Add(24 * time.Hour)}
	if a.Start.Before(bShifted.End) && bShifted.Start.Before(a.End) {
		return true
	}
	// Check if shifting A by +24h reveals an overlap (A starts "tomorrow" relative to B)
	aShifted := payrateShift{Start: a.Start.Add(24 * time.Hour), End: a.End.Add(24 * time.Hour)}
	if aShifted.Start.Before(b.End) && b.Start.Before(aShifted.End) {
		return true
	}
	return false
}

// ResolveFlexibleOverlaps detects overlapping shifts in flexible payrate configurations
// and auto-resolves them: the wider shift wins, the contained (narrower) shift is removed.
// This is the "new config wins" policy — the user's latest save is authoritative.
func (p PayrateConfiguration) ResolveFlexibleOverlaps() (PayrateConfiguration, error) {
	if len(p) == 0 {
		return p, nil
	}

	flattened, err := p.Flatten()
	if err != nil {
		return p, err
	}

	type shiftEntry struct {
		flatKey string
		start   time.Time
		end     time.Time
	}

	positionShifts := make(map[string][]shiftEntry)

	for key := range flattened {
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}
		timeRange := parts[len(parts)-1]
		timeParts := strings.Split(timeRange, "-")
		if len(timeParts) != 2 {
			continue
		}

		layout := "15:04"
		start, err := time.Parse(layout, timeParts[0])
		if err != nil {
			continue
		}
		end, err := time.Parse(layout, timeParts[1])
		if err != nil {
			continue
		}
		if !end.After(start) {
			end = end.Add(24 * time.Hour)
		}

		positionAndDay := strings.Join(parts[:len(parts)-1], ".")
		positionShifts[positionAndDay] = append(positionShifts[positionAndDay], shiftEntry{
			flatKey: key,
			start:   start,
			end:     end,
		})
	}

	// Find narrower (contained) shifts to remove
	removeKeys := make(map[string]bool)
	for _, shifts := range positionShifts {
		for i := 0; i < len(shifts); i++ {
			for j := i + 1; j < len(shifts); j++ {
				a, b := shifts[i], shifts[j]
				if shiftsOverlap(payrateShift{Start: a.start, End: a.end}, payrateShift{Start: b.start, End: b.end}) {
					aDur := a.end.Sub(a.start)
					bDur := b.end.Sub(b.start)
					if aDur >= bDur {
						removeKeys[b.flatKey] = true
					} else {
						removeKeys[a.flatKey] = true
					}
				}
			}
		}
	}

	if len(removeKeys) == 0 {
		return p, nil
	}

	// Remove keys from the nested JSON structure
	var nested map[string]interface{}
	if err := json.Unmarshal(p, &nested); err != nil {
		return p, err
	}

	for key := range removeKeys {
		removeNestedKey(nested, strings.Split(key, "."))
	}

	result, err := json.Marshal(nested)
	if err != nil {
		return p, err
	}
	return PayrateConfiguration(result), nil
}

// removeNestedKey deletes a key at the given dot-separated path from a nested map.
func removeNestedKey(m map[string]interface{}, path []string) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		delete(m, path[0])
		return
	}
	next, ok := m[path[0]].(map[string]interface{})
	if !ok {
		return
	}
	removeNestedKey(next, path[1:])
}

// Parse returns the original nested structure for JSON responses
func (p PayrateConfiguration) Parse() (map[string]interface{}, error) {
	var parsed map[string]interface{}
	err := json.Unmarshal(p, &parsed)
	return parsed, err
}

// IsEmpty returns true if the configuration is empty
func (p PayrateConfiguration) IsEmpty() bool {
	return len(p) == 0
}

// normalizeVietnamese removes Vietnamese diacritics and converts to lowercase for case-insensitive comparison
// Examples: "Ngày lễ" -> "ngay le", "TC 150%" -> "tc 150%", "Ca đêm" -> "ca dem"
//
// Delegates to the canonical utils.NormalizeVietnamese (unidecode-based; folds
// Đ/đ automatically) shared across the codebase, so payrate label matching and
// every other Vietnamese fold can never diverge.
func normalizeVietnamese(s string) string {
	return utils.NormalizeVietnamese(s)
}

// GetRate returns rate value for a flattened path with case-insensitive Vietnamese-normalized matching
func (p PayrateConfiguration) GetRate(path string) (int, error) {
	flattened, err := p.Flatten()
	if err != nil {
		return 0, NewValidationError("không thể phân tích cấu hình mức lương: " + err.Error())
	}

	// Normalize the search path for comparison
	normalizedPath := normalizeVietnamese(path)

	// Try exact match first (for performance)
	if value, exists := flattened[path]; exists {
		return value, nil
	}

	// Fall back to normalized case-insensitive search
	for key, value := range flattened {
		if normalizeVietnamese(key) == normalizedPath {
			return value, nil
		}
	}

	return 0, NewValidationError(fmt.Sprintf("không tìm thấy mức lương cho đường dẫn: %s", path))
}

// NormalizedKeys returns a copy of the configuration whose object keys are
// trimmed and Unicode NFC-normalized. Diacritics are preserved: these keys
// double as display names in the UI (position dropdowns are built from them).
// Sibling keys that collide under the same Vietnamese fold GetRate uses for
// lookups are rejected, so divergent spellings of one position ("Lái xe nâng"
// vs "lái xe nâng") cannot both be persisted and silently price at zero.
func (p PayrateConfiguration) NormalizedKeys() (PayrateConfiguration, error) {
	if len(p) == 0 {
		return p, nil
	}

	var level map[string]interface{}
	if err := json.Unmarshal(p, &level); err != nil {
		return nil, NewValidationError("định dạng JSON không hợp lệ: " + err.Error())
	}

	normalized, err := normalizeConfigKeys(level)
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(normalized)
	if err != nil {
		return nil, NewValidationError("không thể chuẩn hóa cấu hình mức lương: " + err.Error())
	}
	return PayrateConfiguration(out), nil
}

// normalizeConfigKeys rewrites one object level: keys are trimmed and NFC-
// composed, and two sibling keys folding to the same canonical form are
// rejected with both original spellings named. Nested objects recurse.
func normalizeConfigKeys(level map[string]interface{}) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(level))
	folds := make(map[string]string, len(level))

	for key, value := range level {
		cleanKey := strings.TrimSpace(norm.NFC.String(key))
		if cleanKey == "" {
			return nil, NewValidationError("tên vị trí/khung giờ không được để trống sau khi xử lý khoảng trắng")
		}

		fold := normalizeVietnamese(cleanKey)
		if first, dupe := folds[fold]; dupe {
			return nil, NewValidationError(fmt.Sprintf(constants.MsgPayrateFoldKeyCollisionVN, first, cleanKey))
		}
		folds[fold] = cleanKey

		if nested, ok := value.(map[string]interface{}); ok {
			nestedOut, err := normalizeConfigKeys(nested)
			if err != nil {
				return nil, err
			}
			value = nestedOut
		}
		out[cleanKey] = value
	}

	return out, nil
}

// Payrate represents pay rate configuration for a project
type Payrate struct {
	ID          uint                 `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ProjectID   uint                 `json:"project_id" gorm:"not null;type:bigint unsigned"`
	PayrateJSON string               `json:"-" gorm:"column:payrate_json;type:json;not null;comment:'custom payrate format per project; values are hourly for standard projects and per-shift totals for flexible projects'"`
	Payrate     PayrateConfiguration `json:"payrate" gorm:"-"`
	FromDate    time.Time            `json:"fromDate" gorm:"type:date;not null"`
	ToDate      *time.Time           `json:"toDate" gorm:"type:date"`
	DeletedAt   gorm.DeletedAt       `json:"-" gorm:"index"`
	CreatedBy   uint                 `json:"created_by" gorm:"not null;type:bigint unsigned;comment:'User who created the payrate'"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`

	// Relationships
	Project     Project `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
	CreatedUser User    `json:"created_user" gorm:"foreignKey:CreatedBy;references:ID"`
}

// PayrateRepository defines the interface for payrate persistence operations.
// From-date-only model: the config for a work date is the project's latest
// from_date on or before it; to_date is derived and never a query input.
type PayrateRepository interface {
	// Existing methods (non-transactional)
	Create(ctx context.Context, payrate *Payrate) error
	GetByID(ctx context.Context, id uint) (*Payrate, error)
	Update(ctx context.Context, payrate *Payrate) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters PayrateFilters) ([]*Payrate, error)
	Count(ctx context.Context, filters PayrateFilters) (int64, error)
	GetByProject(ctx context.Context, projectID uint) ([]*Payrate, error)
	GetActiveByProjectAndDate(ctx context.Context, projectID uint, date time.Time) (*Payrate, error)
	GetCurrentOrUpcomingByProject(ctx context.Context, projectID uint) (*Payrate, error)
	HasTimesheetsFromDate(ctx context.Context, id uint, fromDate time.Time) (bool, error)
	HasProjectTimesheetsFromDate(ctx context.Context, projectID uint, fromDate time.Time) (bool, error)
	GetLatestTimesheetDateForPayrate(ctx context.Context, payrateID uint) (*time.Time, error)
	GetActiveByProject(ctx context.Context, projectID uint, date time.Time) ([]*Payrate, error)

	// Transaction-aware methods for temporal operations
	CreateWithTx(ctx context.Context, tx interface{}, payrate *Payrate) error
}

// PayrateFilters represents filtering options for payrate queries
type PayrateFilters struct {
	ProjectID *uint
	CreatedBy *uint
	FromDate  *time.Time
	ToDate    *time.Time
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// BeforeCreate is a GORM hook called before creating a record
func (p *Payrate) BeforeCreate(tx *gorm.DB) error {
	return p.marshalPayrateConfig()
}

// BeforeUpdate is a GORM hook called before updating a record
func (p *Payrate) BeforeUpdate(tx *gorm.DB) error {
	return p.marshalPayrateConfig()
}

// AfterFind is a GORM hook called after finding a record
func (p *Payrate) AfterFind() error {
	return p.unmarshalPayrateConfig()
}

// marshalPayrateConfig converts PayrateConfiguration to JSON string
func (p *Payrate) marshalPayrateConfig() error {
	if p.Payrate.IsEmpty() {
		p.PayrateJSON = "{}"
		return nil
	}

	// Ensure the JSON string is not empty
	jsonStr := string(p.Payrate)
	if jsonStr == "" || jsonStr == "null" {
		p.PayrateJSON = "{}"
		return nil
	}

	p.PayrateJSON = jsonStr
	return nil
}

// unmarshalPayrateConfig converts JSON string to PayrateConfiguration
func (p *Payrate) unmarshalPayrateConfig() error {
	if p.PayrateJSON == "" {
		p.Payrate = PayrateConfiguration("{}")
		return nil
	}

	p.Payrate = PayrateConfiguration(p.PayrateJSON)
	return nil
}

// ValidateProjectID validates the project ID
func (p *Payrate) ValidateProjectID() error {
	if p.ProjectID == 0 {
		return NewValidationError("ID dự án là bắt buộc")
	}
	return nil
}

// ValidatePayrateConfig validates the payrate configuration
func (p *Payrate) ValidatePayrateConfig() error {
	return p.Payrate.Validate()
}

// ValidateDates validates from and to dates
func (p *Payrate) ValidateDates() error {
	if p.FromDate.IsZero() {
		return NewValidationError("ngày bắt đầu là bắt buộc")
	}

	if p.ToDate != nil {
		if p.ToDate.Before(p.FromDate) {
			return NewValidationError("ngày kết thúc phải sau ngày bắt đầu")
		}
	}

	return nil
}

// IsValid validates the entire payrate entity
func (p *Payrate) IsValid() error {
	if err := p.ValidateProjectID(); err != nil {
		return err
	}
	if err := p.ValidatePayrateConfig(); err != nil {
		return err
	}
	if err := p.ValidateDates(); err != nil {
		return err
	}
	return nil
}

// IsActiveOnDate reports whether this config has taken effect on the given
// date. From-date-only model: the config in force for a date is the project's
// latest from_date on or before it — this check is only the lower bound, and
// the resolver guarantees the row it returns is the latest qualifying one.
// to_date is derived bookkeeping and never gates activity.
func (p *Payrate) IsActiveOnDate(date time.Time) bool {
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	fromDateOnly := time.Date(p.FromDate.Year(), p.FromDate.Month(), p.FromDate.Day(), 0, 0, 0, 0, p.FromDate.Location())
	return !dateOnly.Before(fromDateOnly)
}

// GetRateValue returns the VND rate value for a flattened path
// Examples:
// - Vietnamese: GetRateValue("tất cả.ngày thường.bình thường")
func (p *Payrate) GetRateValue(path string) (int, error) {
	if p.Payrate.IsEmpty() {
		return 0, NewValidationError("cấu hình mức lương chưa được tải")
	}

	return p.Payrate.GetRate(path)
}

// SetPayrateConfiguration replaces the entire payrate configuration
// For individual rate updates, use CRUD operations to replace the whole config
func (p *Payrate) SetPayrateConfiguration(config PayrateConfiguration) error {
	if err := config.Validate(); err != nil {
		return err
	}
	p.Payrate = config
	return nil
}

// GetAllRates returns the raw JSON configuration
func (p *Payrate) GetAllRates() PayrateConfiguration {
	if p.Payrate.IsEmpty() {
		return PayrateConfiguration("{}")
	}
	return p.Payrate
}

// Note: No skill level validation functions needed
// Clients define their own skill level names freely
