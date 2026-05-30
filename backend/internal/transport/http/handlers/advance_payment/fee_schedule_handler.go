package advance_payment

import (
	"github.com/gin-gonic/gin"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"
)

// FeeScheduleHandler exposes admin CRUD on the advance-payment fee schedule
// list. Routes are admin-gated upstream by the Casbin middleware (admin role
// has wildcard /api/* access; partners are excluded by absence of an explicit
// allow rule).
type FeeScheduleHandler struct {
	service *advance_payment.FeeScheduleService
	clock   clock.Clock
}

// NewFeeScheduleHandler wires the handler with the schedule service.
func NewFeeScheduleHandler(service *advance_payment.FeeScheduleService, clk clock.Clock) *FeeScheduleHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &FeeScheduleHandler{service: service, clock: clk}
}

// List returns every schedule entry, newest effective_date first, with flags
// indicating which one is currently active and which are pending.
func (h *FeeScheduleHandler) List(c *gin.Context) {
	entries, err := h.service.List(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	active := domain.ActiveFeeScheduleAt(entries, now)
	today := now.Format(domain.AdvancePaymentFeeScheduleDateLayout)

	out := make([]dto.FeeScheduleEntryResponse, 0, len(entries))
	for i := range entries {
		e := entries[i]
		isActive := active != nil && active.ID == e.ID
		isPending := e.EffectiveDate > today
		out = append(out, toFeeScheduleResponse(e, isActive, isPending))
	}

	response.Success(c, dto.FeeScheduleListResponse{Entries: out}, "Lấy danh sách cấu hình phí thành công")
}

// Create appends a new schedule entry. Validation (effective_date not past,
// tier shape, percentage/min-fee bounds) is enforced at the service layer.
func (h *FeeScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	actorID := actorUserID(c)
	created, err := h.service.Create(c.Request.Context(), advance_payment.CreateInput{
		EffectiveDate: req.EffectiveDate,
		Tiers:         tiersFromDTO(req.Tiers),
		MinFeeVND:     req.MinFeeVND,
		Notes:         req.Notes,
	}, actorID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	today := now.Format(domain.AdvancePaymentFeeScheduleDateLayout)
	isPending := created.EffectiveDate > today
	isActive := !isPending // newly-created schedules with today's effective_date are active immediately

	response.SuccessCreated(c, toFeeScheduleResponse(*created, isActive, isPending), "Tạo cấu hình phí thành công")
}

// Update mutates a future-dated entry. Past or currently-active entries are
// immutable for audit integrity.
func (h *FeeScheduleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "id is required")
		return
	}

	var req dto.UpdateFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), id, advance_payment.UpdateInput{
		EffectiveDate: req.EffectiveDate,
		Tiers:         tiersFromDTO(req.Tiers),
		MinFeeVND:     req.MinFeeVND,
		Notes:         req.Notes,
	})
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	today := now.Format(domain.AdvancePaymentFeeScheduleDateLayout)
	isPending := updated.EffectiveDate > today
	response.Success(c, toFeeScheduleResponse(*updated, false, isPending), "Cập nhật cấu hình phí thành công")
}

// Delete removes a future-dated entry. The active and historical entries
// remain immutable; the system always retains at least one schedule.
func (h *FeeScheduleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "id is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, gin.H{"id": id}, "Xóa cấu hình phí thành công")
}

func tiersFromDTO(in []dto.FeeScheduleTierDTO) []domain.FeeScheduleTier {
	out := make([]domain.FeeScheduleTier, len(in))
	for i, t := range in {
		out[i] = domain.FeeScheduleTier{MinAmount: t.MinAmount, Percentage: t.Percentage}
	}
	return out
}

func tiersToDTO(in []domain.FeeScheduleTier) []dto.FeeScheduleTierDTO {
	out := make([]dto.FeeScheduleTierDTO, len(in))
	for i, t := range in {
		out[i] = dto.FeeScheduleTierDTO{MinAmount: t.MinAmount, Percentage: t.Percentage}
	}
	return out
}

func toFeeScheduleResponse(e domain.FeeScheduleEntry, isActive, isPending bool) dto.FeeScheduleEntryResponse {
	return dto.FeeScheduleEntryResponse{
		ID:                e.ID,
		EffectiveDate:     e.EffectiveDate,
		Tiers:             tiersToDTO(e.Tiers),
		MinFeeVND:         e.MinFeeVND,
		Notes:             e.Notes,
		CreatedAt:         e.CreatedAt,
		CreatedByUserID:   e.CreatedByUserID,
		IsCurrentlyActive: isActive,
		IsPending:         isPending,
		Summary:           advance_payment.FormatScheduleSummary(e),
	}
}

func actorUserID(c *gin.Context) uint {
	v, ok := c.Get(constants.CtxUserID)
	if !ok {
		return 0
	}
	if id, ok := v.(uint); ok {
		return id
	}
	return 0
}
