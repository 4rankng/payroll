package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
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
		id, err := strconv.ParseUint(idParam, 10, 32)
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
			if id, err := strconv.ParseUint(pid, 10, 32); err == nil {
				projectID = uint(id)
			}
		}
	}

	today := timeutil.StartOfDay(h.clock.NowUTC())
	todayStr := today.Format(timeutil.DateFormat)

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

	// ── Temporal / timesheet conflict checks ──────────────────────────────
	if !projectBlocked && len(fromResult.errors) == 0 && len(ratesResult.Message) == 0 {
		if isUpdate && existingPayrate != nil {
			h.applyUpdateConstraints(c, existingPayrate, req, &fromResult.field, &toResult.field, &ratesResult, today, todayStr)
		} else if projectID > 0 && fromResult.parsedDate != nil {
			h.applyCreateConstraints(c, projectID, req.EffectiveFrom, fromResult.parsedDate, &fromResult.field, today, todayStr)
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

	c.JSON(200, ValidatePayrateResponse{
		Valid:               valid,
		CanSaveWithWarnings: canSave,
		Summary:             buildValidateSummary(valid, canSave, hasWarnings),
		Fields:              fields,
	})
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

	r.Status = FieldOK
	return r
}

func (h *PayrateHandler) applyUpdateConstraints(
	c *gin.Context,
	existing *domain.Payrate,
	req dto.CreatePayrateRequest,
	fromField *FieldResult,
	toField *FieldResult,
	ratesField *RatesFieldResult,
	today time.Time,
	todayStr string,
) {
	isUsed, err := h.payrateService.IsPayrateUsedByTimesheets(c.Request.Context(), existing.ID)
	if err != nil || !isUsed {
		return
	}

	existingFromStr := existing.FromDate.Format(timeutil.DateFormat)
	existingFrom := timeutil.StartOfDay(existing.FromDate.UTC())

	// Find earliest valid start date for a new config = latest timesheet date + 1 day
	earliestNewStart := today
	latestDate, latestErr := h.payrateService.GetLatestTimesheetDateForPayrate(c.Request.Context(), existing.ID)
	if latestErr == nil && latestDate != nil {
		earliestNewStart = timeutil.StartOfDay(latestDate.UTC()).AddDate(0, 0, 1)
	}
	earliestNewStartStr := earliestNewStart.Format(timeutil.DateFormat)

	ratesChanged := strings.TrimSpace(string(existing.Payrate)) != strings.TrimSpace(string(req.Rates))

	submittedFrom, _ := time.Parse(timeutil.DateFormat, req.EffectiveFrom)
	submittedFromOnly := timeutil.StartOfDay(submittedFrom.UTC())
	fromDateChanged := !submittedFromOnly.Equal(existingFrom)

	// Case 1: rates changed AND fromDate is on or after earliestNewStart
	// → valid "create new config" scenario, let it through
	if ratesChanged && fromDateChanged && !submittedFromOnly.Before(earliestNewStart) {
		return
	}

	// Case 2: rates changed but fromDate is still the original (or too early)
	if ratesChanged {
		ratesField.Status = FieldError
		ratesField.Locked = true
		ratesField.Message = "Mức lương bị khóa — cấu hình này đã có bảng công liên kết."
		ratesField.Hint = fmt.Sprintf(
			"Để thay đổi mức lương, hãy đổi ngày bắt đầu sang %s (ngày sau bảng công gần nhất) để tạo cấu hình mới.",
			earliestNewStartStr,
		)
		ratesField.SuggestedValue = earliestNewStartStr

		// If fromDate is still the original, guide user to change it
		if !fromDateChanged {
			fromField.Status = FieldError
			fromField.Message = fmt.Sprintf("Đổi ngày bắt đầu sang %s để tạo cấu hình mới với mức lương mới.", earliestNewStartStr)
			fromField.Hint = fmt.Sprintf("Ngày bắt đầu sớm nhất hợp lệ: %s (ngày sau bảng công gần nhất).", earliestNewStartStr)
			fromField.MinValue = earliestNewStartStr
			fromField.SuggestedValue = earliestNewStartStr
		} else if submittedFromOnly.Before(earliestNewStart) {
			// fromDate changed but still too early
			fromField.Status = FieldError
			fromField.Message = fmt.Sprintf("Ngày %s vẫn còn bảng công liên kết. Ngày sớm nhất hợp lệ: %s.", req.EffectiveFrom, earliestNewStartStr)
			fromField.Hint = fmt.Sprintf("Chọn ngày từ %s trở đi.", earliestNewStartStr)
			fromField.MinValue = earliestNewStartStr
			fromField.SuggestedValue = earliestNewStartStr
		}
		return
	}

	// Case 3: rates unchanged — only fromDate or toDate changes allowed
	if fromDateChanged {
		fromField.Status = FieldError
		fromField.Locked = true
		fromField.Message = fmt.Sprintf("Ngày bắt đầu bị khóa — đã có bảng công liên kết (ngày gốc: %s).", existingFromStr)
		fromField.Hint = fmt.Sprintf("Khôi phục lại ngày bắt đầu gốc: %s.", existingFromStr)
		fromField.SuggestedValue = existingFromStr
	}

	// To date in the past?
	if req.EffectiveTo != nil && *req.EffectiveTo != "" {
		toDate, err := time.Parse(timeutil.DateFormat, *req.EffectiveTo)
		if err == nil && timeutil.StartOfDay(toDate.UTC()).Before(today) {
			toField.Status = FieldError
			toField.Message = "Ngày kết thúc không thể là ngày trong quá khứ khi đã có bảng công liên kết."
			toField.Hint = fmt.Sprintf("Chọn ngày từ hôm nay (%s) trở đi, hoặc để trống.", todayStr)
			toField.MinValue = todayStr
			toField.SuggestedValue = todayStr
		}
	}
}

func (h *PayrateHandler) applyCreateConstraints(
	c *gin.Context,
	projectID uint,
	submittedFrom string,
	fromDate *time.Time,
	fromField *FieldResult,
	today time.Time,
	todayStr string,
) {
	fromDateOnly := timeutil.StartOfDay(fromDate.UTC())
	if !fromDateOnly.Before(today) {
		return
	}

	hasTimesheets, err := h.payrateService.HasProjectTimesheetsFromDate(c.Request.Context(), projectID, fromDateOnly)
	if err != nil || !hasTimesheets {
		return
	}

	fromField.Status = FieldError
	fromField.Message = fmt.Sprintf("Dự án đã có bảng công từ ngày %s — không thể dùng ngày này.", submittedFrom)
	fromField.Hint = fmt.Sprintf("Chọn ngày bắt đầu từ hôm nay (%s) trở đi.", todayStr)
	fromField.MinValue = todayStr
	fromField.SuggestedValue = todayStr
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
