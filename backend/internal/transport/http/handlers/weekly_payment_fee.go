package handlers

import (
	"github.com/gin-gonic/gin"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/payroll"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"
)

// WeeklyPaymentFeeHandler exposes admin CRUD on the weekly-payment fee
// schedule list (phí trả lương tuần). Routes are admin-gated upstream by the
// Casbin middleware: admin role has wildcard /api/* access; partners are
// excluded by absence of an explicit allow rule.
type WeeklyPaymentFeeHandler struct {
	service *payroll.WeeklyPaymentFeeScheduleService
	clock   clock.Clock
}

// NewWeeklyPaymentFeeHandler wires the handler with the schedule service.
func NewWeeklyPaymentFeeHandler(service *payroll.WeeklyPaymentFeeScheduleService, clk clock.Clock) *WeeklyPaymentFeeHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &WeeklyPaymentFeeHandler{service: service, clock: clk}
}

// List returns every schedule entry, newest effective_date first, with flags
// indicating which one is currently active and which are pending.
func (h *WeeklyPaymentFeeHandler) List(c *gin.Context) {
	entries, err := h.service.List(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	active := domain.ActiveWeeklyPaymentFeeScheduleAt(entries, now)
	today := now.Format(domain.WeeklyPaymentFeeScheduleDateLayout)

	out := make([]dto.WeeklyPaymentFeeScheduleEntryResponse, 0, len(entries))
	for i := range entries {
		e := entries[i]
		isActive := active != nil && active.ID == e.ID
		isPending := e.EffectiveDate > today
		out = append(out, toWeeklyPaymentFeeResponse(e, isActive, isPending))
	}

	response.Success(c, dto.WeeklyPaymentFeeScheduleListResponse{Entries: out}, "Lấy danh sách cấu hình phí trả lương tuần thành công")
}

// Create appends a new schedule entry. Validation (effective_date not past,
// percentage bounds) is enforced at the service layer.
func (h *WeeklyPaymentFeeHandler) Create(c *gin.Context) {
	var req dto.CreateWeeklyPaymentFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	created, err := h.service.Create(c.Request.Context(), payroll.WeeklyFeeScheduleCreateInput{
		EffectiveDate: req.EffectiveDate,
		Percentage:    req.Percentage,
		Notes:         req.Notes,
	}, weeklyFeeActorUserID(c))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	today := now.Format(domain.WeeklyPaymentFeeScheduleDateLayout)
	isPending := created.EffectiveDate > today
	isActive := !isPending // entries with today's effective_date are active immediately

	response.SuccessCreated(c, toWeeklyPaymentFeeResponse(*created, isActive, isPending), "Tạo cấu hình phí trả lương tuần thành công")
}

// Update mutates a future-dated entry. Past or currently-active entries are
// immutable for audit integrity.
func (h *WeeklyPaymentFeeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "id is required")
		return
	}

	var req dto.UpdateWeeklyPaymentFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), id, payroll.WeeklyFeeScheduleUpdateInput{
		EffectiveDate: req.EffectiveDate,
		Percentage:    req.Percentage,
		Notes:         req.Notes,
	})
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	today := now.Format(domain.WeeklyPaymentFeeScheduleDateLayout)
	isPending := updated.EffectiveDate > today
	response.Success(c, toWeeklyPaymentFeeResponse(*updated, false, isPending), "Cập nhật cấu hình phí trả lương tuần thành công")
}

// Delete removes a future-dated entry. The active and historical entries
// remain immutable; the system always retains at least one schedule.
func (h *WeeklyPaymentFeeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "id is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, gin.H{"id": id}, "Xóa cấu hình phí trả lương tuần thành công")
}

func toWeeklyPaymentFeeResponse(e domain.WeeklyPaymentFeeScheduleEntry, isActive, isPending bool) dto.WeeklyPaymentFeeScheduleEntryResponse {
	return dto.WeeklyPaymentFeeScheduleEntryResponse{
		ID:                e.ID,
		EffectiveDate:     e.EffectiveDate,
		Percentage:        e.Percentage,
		Notes:             e.Notes,
		CreatedAt:         e.CreatedAt,
		CreatedByUserID:   e.CreatedByUserID,
		IsCurrentlyActive: isActive,
		IsPending:         isPending,
		Summary:           domain.FormatWeeklyPaymentFeeSummary(e),
	}
}

func weeklyFeeActorUserID(c *gin.Context) uint {
	v, ok := c.Get(constants.CtxUserID)
	if !ok {
		return 0
	}
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}
