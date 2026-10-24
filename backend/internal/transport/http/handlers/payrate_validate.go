package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// FieldStatus is the per-field validation state the frontend renders directly.
// "ok"      → green / normal
// "error"   → red border, blocks save
// "warning" → amber border, allows save with confirmation
type FieldStatus string

const (
	FieldOK      FieldStatus = "ok"
	FieldError   FieldStatus = "error"
	FieldWarning FieldStatus = "warning"
)

// FieldResult mirrors back one submitted field with full annotation.
// Frontend maps this 1-to-1 onto the corresponding input.
type FieldResult struct {
	// Submitted is the value the client sent (echoed back for display).
	Submitted string `json:"submitted"`

	// Status drives the input's visual state.
	Status FieldStatus `json:"status"`

	// Locked tells the frontend to make this input read-only.
	Locked bool `json:"locked"`

	// Message is the short inline error/warning shown under the input.
	// Empty when status == "ok".
	Message string `json:"message,omitempty"`

	// Hint is the actionable instruction shown below the message.
	// Tells the user exactly what to type or do.
	Hint string `json:"hint,omitempty"`

	// MinValue is the minimum acceptable value for this field (date or number).
	// Frontend can set input min= attribute directly from this.
	MinValue string `json:"min_value,omitempty"`

	// SuggestedValue is the exact value the user should enter to fix the issue.
	// Frontend can render a "Use this value →" one-click button.
	SuggestedValue string `json:"suggested_value,omitempty"`
}

// RatesFieldResult extends FieldResult with rates-specific context.
type RatesFieldResult struct {
	FieldResult

	// TimesheetCount is how many timesheets are linked to this payrate.
	// Shown to the user so they understand why rates are locked.
	TimesheetCount int64 `json:"timesheet_count,omitempty"`

	// MissingPositions lists the latest config's top-level positions this
	// save drops (warning only — the save may proceed with confirmation).
	// From-date-only model: a dropped position can no longer price new
	// timesheets, so the user must see what they are about to remove.
	MissingPositions []string `json:"missing_positions,omitempty"`

	// KeyDriftWarnings names submitted positions that are case/spelling
	// variants of existing ones (same Vietnamese fold, different spelling).
	// Pricing survives via the lookup fold, but display and dropdowns drift —
	// the user should keep one spelling.
	KeyDriftWarnings []string `json:"key_drift_warnings,omitempty"`
}

// ValidatePayrateFields is the per-field breakdown mirroring the submitted request.
type ValidatePayrateFields struct {
	EffectiveFrom FieldResult      `json:"effective_from"`
	EffectiveTo   FieldResult      `json:"effective_to"`
	Rates         RatesFieldResult `json:"rates"`
}

// ValidatePayrateResponse is the full dry-run response.
// The frontend drives all UI state from this single object.
type ValidatePayrateResponse struct {
	// Valid is true only when every field has status "ok" or "warning".
	Valid bool `json:"valid"`

	// CanSaveWithWarnings is true when there are warnings but no errors.
	CanSaveWithWarnings bool `json:"can_save_with_warnings"`

	// Summary is a one-line headline for the overall result.
	Summary string `json:"summary"`

	// Fields mirrors back every submitted field with its validation state.
	Fields ValidatePayrateFields `json:"fields"`

	// MutableTimesheetsAffected counts the unpaid and unapproved timesheets
	// this save would re-price (work date on or after the effective date).
	// Shown before confirming so the blast radius is explicit.
	MutableTimesheetsAffected int64 `json:"mutable_timesheets_affected"`
}

// ValidatePayrate performs a dry-run validation without persisting.
//
//	POST /api/v1/payrates/validate          — create dry-run
//	POST /api/v1/payrates/:id/validate      — update dry-run
func (h *PayrateHandler) ValidatePayrate(c *gin.Context) {
	idParam := c.Param("id")
	isUpdate := idParam != "" && idParam != "/"

	var existingPayrate *domain.Payrate
	if isUpdate {
		id, err := strconv.ParseUint(idParam, 10, 64)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidPayrateIDVN)
			return
		}
		existingPayrate, err = h.payrateService.GetPayrate(c.Request.Context(), uint(id))
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}
	}

	var req dto.CreatePayrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, ValidatePayrateResponse{
			Valid:   false,
			Summary: "Dữ liệu gửi lên không hợp lệ",
			Fields: ValidatePayrateFields{
				EffectiveFrom: FieldResult{Status: FieldError, Message: "Không thể đọc dữ liệu: " + err.Error()},
				EffectiveTo:   FieldResult{Status: FieldOK},
				Rates:         RatesFieldResult{FieldResult: FieldResult{Status: FieldError, Message: "Không thể đọc dữ liệu"}},
			},
		})
		return
	}

	// Resolve project ID
	projectID := req.ProjectID
	if isUpdate && existingPayrate != nil {
		projectID = existingPayrate.ProjectID
	}
	if projectID == 0 {
		if pid := c.Query("project_id"); pid != "" {
			if id, err := strconv.ParseUint(pid, 10, 64); err == nil {
				projectID = uint(id)
			}
		}
	}

	today := timeutil.StartOfDay(h.clock.NowUTC())

	// ── Build per-field results ────────────────────────────────────────────

	fromResult := validateFromDate(req.EffectiveFrom, today)
	toResult := validateToDate(req.EffectiveTo, fromResult.parsedDate, today)
	ratesResult := validateRates(req.Rates)

	// ── Project-level checks ───────────────────────────────────────────────
	projectBlocked := false
	if projectID > 0 {
		project, err := h.projectService.GetProject(c.Request.Context(), projectID)
		if err == nil && (project.IsCompleted() || project.IsCancelled()) {
			projectBlocked = true
			// Lock all fields — nothing can be changed
			fromResult.field.Status = FieldError
			fromResult.field.Locked = true
			fromResult.field.Message = fmt.Sprintf("Dự án đang ở trạng thái '%s'", string(project.ProjectStatus))
			fromResult.field.Hint = "Dự án đã hoàn thành hoặc bị hủy — không thể thay đổi cấu hình lương."
			toResult.field.Status = FieldError
			toResult.field.Locked = true
			ratesResult.Status = FieldError
			ratesResult.Locked = true
		}
	}

	// ── Non-latest check (update parity) ──────────────────────────────────
	// From-date-only model: a config a newer config has replaced is history
	// and view-only; mirror that here so the dry-run never reports a save
	// that cannot succeed. The stored to_date is ignored (stale by design).
	if !projectBlocked && isUpdate && existingPayrate != nil {
		isLatest, successor, latestErr := h.payrateService.IsLatestPayrate(c.Request.Context(), existingPayrate)
		if latestErr == nil && !isLatest && successor != nil {
			notLatestMessage := fmt.Sprintf(constants.MsgCannotUpdateNonLatestPayrateVN, successor.FromDate.Format(timeutil.DateFormat))
			fromResult.field.Status = FieldError
			fromResult.field.Locked = true
			fromResult.field.Message = notLatestMessage
			ratesResult.Status = FieldError
			ratesResult.Locked = true
			ratesResult.Message = notLatestMessage
		}
	}

	// ── Temporal / timesheet constraint checks ────────────────────────────
	if !projectBlocked && len(fromResult.errors) == 0 && len(ratesResult.Message) == 0 {
		// The paid floor binds creates and updates alike; the from-date-only
		// ordering rules differ (create appends strictly after the latest
		// config, an update may not cross a sibling). projectID is already
		// resolved from existingPayrate for updates above.
		if projectID > 0 && fromResult.parsedDate != nil {
			h.applyTemporalConstraints(c, projectID, isUpdate, existingPayrate, req.EffectiveFrom, fromResult.parsedDate, &fromResult.field)
		}
	}

	// ── Carry-forward completeness ────────────────────────────────────────
	// Warn when the submitted matrix drops positions present in the latest
	// config: those positions can no longer price new timesheets.
	if !projectBlocked && projectID > 0 && ratesResult.Status == FieldOK {
		if latest, latestErr := h.payrateService.GetLatestPayrateByProject(c.Request.Context(), projectID); latestErr == nil && latest != nil {
			if missing := missingPositions(latest.Payrate, req.Rates); len(missing) > 0 {
				ratesResult.Status = FieldWarning
				ratesResult.MissingPositions = missing
				ratesResult.Message = fmt.Sprintf("Cấu hình đang bỏ %d vị trí có trong cấu hình hiện tại.", len(missing))
				ratesResult.Hint = "Các vị trí bị bỏ sẽ không tạo được bảng công mới: " + strings.Join(missing, ", ")
			} else if drift := keyDriftWarnings(latest.Payrate, req.Rates); len(drift) > 0 {
				ratesResult.Status = FieldWarning
				ratesResult.KeyDriftWarnings = drift
				ratesResult.Message = "Có vị trí viết khác cấu hình hiện tại (khác hoa/thường hoặc dấu)."
				ratesResult.Hint = "Nên dùng cùng một cách viết để danh sách vị trí không bị lệch: " + strings.Join(drift, "; ")
			}
		}
	}

	// ── Assemble response ──────────────────────────────────────────────────
	fields := ValidatePayrateFields{
		EffectiveFrom: fromResult.field,
		EffectiveTo:   toResult.field,
		Rates:         ratesResult,
	}

	hasErrors := fields.EffectiveFrom.Status == FieldError ||
		fields.EffectiveTo.Status == FieldError ||
		fields.Rates.Status == FieldError

	hasWarnings := fields.EffectiveFrom.Status == FieldWarning ||
		fields.EffectiveTo.Status == FieldWarning ||
		fields.Rates.Status == FieldWarning

	valid := !hasErrors
	canSave := !hasErrors && hasWarnings

	resp := ValidatePayrateResponse{
		Valid:               valid,
		CanSaveWithWarnings: canSave,
		Summary:             buildValidateSummary(valid, canSave, hasWarnings),
		Fields:              fields,
	}
	if !projectBlocked && projectID > 0 && fromResult.parsedDate != nil {
		if affected, cntErr := h.payrateService.CountMutableTimesheetsOnOrAfter(c.Request.Context(), projectID, *fromResult.parsedDate); cntErr == nil {
			resp.MutableTimesheetsAffected = affected
		}
	}
	c.JSON(200, resp)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type fromDateResult struct {
	field      FieldResult
	parsedDate *time.Time
	errors     []string
}

func validateFromDate(submitted string, today time.Time) fromDateResult {
	r := fromDateResult{}
	r.field.Submitted = submitted

	if submitted == "" {
		r.field.Status = FieldError
		r.field.Message = "Ngày bắt đầu là bắt buộc."
		r.field.Hint = "Chọn ngày bắt đầu hiệu lực cho cấu hình lương."
		r.errors = append(r.errors, "missing")
		return r
	}

	parsed, err := time.Parse(timeutil.DateFormat, submitted)
	if err != nil {
		r.field.Status = FieldError
		r.field.Message = fmt.Sprintf("Định dạng ngày không hợp lệ (cần: %s).", timeutil.DateFormat)
		r.field.Hint = fmt.Sprintf("Ví dụ: %s", today.Format(timeutil.DateFormat))
		r.errors = append(r.errors, "format")
		return r
	}

	r.parsedDate = &parsed
	r.field.Status = FieldOK
	return r
}

type toDateResult struct {
	field      FieldResult
	parsedDate *time.Time
}

func validateToDate(submitted *string, fromDate *time.Time, today time.Time) toDateResult {
	r := toDateResult{}
	if submitted == nil || *submitted == "" {
		r.field.Submitted = ""
		r.field.Status = FieldOK
		return r
	}

	r.field.Submitted = *submitted
	parsed, err := time.Parse(timeutil.DateFormat, *submitted)
	if err != nil {
		r.field.Status = FieldError
		r.field.Message = fmt.Sprintf("Định dạng ngày kết thúc không hợp lệ (cần: %s).", timeutil.DateFormat)
		r.field.Hint = fmt.Sprintf("Ví dụ: %s", today.AddDate(0, 6, 0).Format(timeutil.DateFormat))
		r.field.SuggestedValue = today.AddDate(0, 6, 0).Format(timeutil.DateFormat)
		return r
	}

	r.parsedDate = &parsed

	if fromDate != nil && parsed.Before(*fromDate) {
		r.field.Status = FieldError
		r.field.Message = "Ngày kết thúc phải sau ngày bắt đầu."
		suggested := fromDate.AddDate(0, 1, 0).Format(timeutil.DateFormat)
		r.field.Hint = fmt.Sprintf("Chọn ngày sau %s.", fromDate.Format(timeutil.DateFormat))
		r.field.MinValue = fromDate.AddDate(0, 0, 1).Format(timeutil.DateFormat)
		r.field.SuggestedValue = suggested
		return r
	}

	r.field.Status = FieldOK
	return r
}

func validateRates(rates []byte) RatesFieldResult {
	r := RatesFieldResult{}
	r.Submitted = string(rates)

	if len(rates) == 0 {
		r.Status = FieldError
		r.Message = "Cấu hình mức lương không được để trống."
		r.Hint = "Thêm ít nhất một vị trí và mức lương."
		return r
	}

	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(rates),
	}
	if err := payrate.Payrate.ValidateDetailed(); err != nil {
		r.Status = FieldError
		r.Message = err.Error()
		r.Hint = "Tất cả mức lương phải là số nguyên không âm."
		return r
	}

	// Normalize keys (trim + NFC) and surface Vietnamese fold-collisions as a
	// field error before save — mirrors the write-path normalization.
	if _, normErr := payrate.Payrate.NormalizedKeys(); normErr != nil {
		r.Status = FieldError
		r.Message = normErr.Error()
		r.Hint = "Mỗi vị trí và khung giờ cần một tên duy nhất."
		return r
	}

	r.Status = FieldOK
	return r
}

// applyTemporalConstraints enforces the timeline rules mirrored from the save
// transaction: the paid floor binds creates and updates alike; ordering
// differs — a create must start strictly after the latest config, an update
// may not move onto or past a sibling config's start.
func (h *PayrateHandler) applyTemporalConstraints(
	c *gin.Context,
	projectID uint,
	isUpdate bool,
	existing *domain.Payrate,
	submittedFrom string,
	fromDate *time.Time,
	fromField *FieldResult,
) {
	// Paid floor: the start date must fall after the project's most recent
	// paid timesheet work date. Mutable (unpaid and unapproved) rows are
	// recalculated on save.
	latestPaidDate, err := h.payrateService.GetLatestPaidTimesheetDateForProject(c.Request.Context(), projectID)
	if err == nil && latestPaidDate != nil {
		// Wall-clock comparison: converting the stored date to UTC first would
		// shift it back a day under a loc=Local MySQL DSN (GMT+7/+8).
		earliestDate := timeutil.StartOfDay(*latestPaidDate).AddDate(0, 0, 1)
		if timeutil.StartOfDay(*fromDate).Before(earliestDate) {
			earliestDateStr := earliestDate.Format(timeutil.DateFormat)
			fromField.Status = FieldError
			fromField.Message = fmt.Sprintf("Ngày %s có trước ngày trả lương gần nhất của dự án.", submittedFrom)
			fromField.Hint = fmt.Sprintf("Chọn ngày từ %s trở đi để chỉ cập nhật các bảng công chưa thanh toán và chưa duyệt.", earliestDateStr)
			fromField.MinValue = earliestDateStr
			fromField.SuggestedValue = earliestDateStr
			return
		}
	}

	if isUpdate {
		if existing == nil {
			return
		}
		hasSibling, siblingErr := h.payrateService.HasSiblingPayrateOnOrAfter(c.Request.Context(), projectID, existing.ID, *fromDate)
		if siblingErr == nil && hasSibling {
			fromField.Status = FieldError
			fromField.Message = constants.MsgCannotUpdatePayrateConflictingConfigVN
			fromField.Hint = "Ngày hiệu lực không được trùng hoặc vượt quá ngày bắt đầu của cấu hình khác."
		}
		return
	}

	// Create: the timeline is append-only — the new config starts strictly
	// after the latest existing one; correcting the present is an update.
	latest, latestErr := h.payrateService.GetLatestPayrateByProject(c.Request.Context(), projectID)
	if latestErr != nil || latest == nil {
		return
	}
	latestDay := timeutil.StartOfDay(latest.FromDate)
	if !timeutil.StartOfDay(*fromDate).After(latestDay) {
		latestStr := latestDay.Format(timeutil.DateFormat)
		nextStr := latestDay.AddDate(0, 0, 1).Format(timeutil.DateFormat)
		fromField.Status = FieldError
		fromField.Message = fmt.Sprintf("Ngày %s không được trước hoặc trùng ngày bắt đầu của cấu hình mới nhất (%s).", submittedFrom, latestStr)
		fromField.Hint = "Để sửa mức lương hiện tại, hãy chỉnh sửa cấu hình đang mở. Cấu hình mới chỉ dùng cho các ngày sau đó."
		fromField.MinValue = nextStr
		fromField.SuggestedValue = nextStr
	}
}

// missingPositions lists the latest config's top-level position keys that the
// submitted configuration drops, compared under the Vietnamese lookup fold so
// case variants count as present.
func missingPositions(latest domain.PayrateConfiguration, submitted []byte) []string {
	var latestMap, submittedMap map[string]json.RawMessage
	if err := json.Unmarshal(latest, &latestMap); err != nil {
		return nil
	}
	if err := json.Unmarshal(submitted, &submittedMap); err != nil {
		return nil
	}

	foldKeys := func(m map[string]json.RawMessage) map[string]string {
		out := make(map[string]string, len(m))
		for k := range m {
			out[utils.NormalizeVietnamese(strings.TrimSpace(k))] = k
		}
		return out
	}
	submittedFolds := foldKeys(submittedMap)

	var missing []string
	for k := range latestMap {
		if _, ok := submittedFolds[utils.NormalizeVietnamese(strings.TrimSpace(k))]; !ok {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

// keyDriftWarnings names submitted positions whose spelling differs from the
// latest config's spelling of the same folded position (e.g. "xuất hàng" vs
// "Xuất hàng"). Legal — lookups fold — but the display drift compounds.
func keyDriftWarnings(latest domain.PayrateConfiguration, submitted []byte) []string {
	var latestMap, submittedMap map[string]json.RawMessage
	if err := json.Unmarshal(latest, &latestMap); err != nil {
		return nil
	}
	if err := json.Unmarshal(submitted, &submittedMap); err != nil {
		return nil
	}

	var warnings []string
	for k := range submittedMap {
		fold := utils.NormalizeVietnamese(strings.TrimSpace(k))
		if latestSpelling, ok := latestFolding(latestMap)[fold]; ok && latestSpelling != k {
			warnings = append(warnings, fmt.Sprintf("%q (hiện tại: %q)", k, latestSpelling))
		}
	}
	sort.Strings(warnings)
	return warnings
}

func latestFolding(m map[string]json.RawMessage) map[string]string {
	out := make(map[string]string, len(m))
	for k := range m {
		out[utils.NormalizeVietnamese(strings.TrimSpace(k))] = k
	}
	return out
}

func buildValidateSummary(valid, canSave, hasWarnings bool) string {
	if valid && !hasWarnings {
		return "Tất cả hợp lệ — sẵn sàng lưu"
	}
	if canSave {
		return "Có cảnh báo — bạn vẫn có thể lưu"
	}
	return "Có lỗi cần sửa trước khi lưu"
}
