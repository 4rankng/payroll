package disbursement

import (
	"github.com/gin-gonic/gin"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"
)

// FeeScheduleHandler exposes admin CRUD on the disbursement-provider fee
// schedule list. Routes are admin-gated upstream by the Casbin middleware.
type FeeScheduleHandler struct {
	service *disbursement.FeeScheduleService
	clock   clock.Clock
}

// NewFeeScheduleHandler wires the handler with the schedule service.
func NewFeeScheduleHandler(service *disbursement.FeeScheduleService, clk clock.Clock) *FeeScheduleHandler {
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
	today := now.Format(domain.DisbursementFeeScheduleDateLayout)

	// Pre-compute the active entry per provider — O(n) instead of O(n²).
	activeByProvider := make(map[string]string) // provider → active entry ID
	{
		seen := map[string]bool{}
		for i := range entries {
			p := entries[i].Provider
			if seen[p] {
				continue
			}
			seen[p] = true
			a := domain.ActiveDisbursementFeeScheduleAt(entries, p, now)
			if a != nil {
				activeByProvider[p] = a.ID
			}
		}
	}

	out := make([]dto.DisbursementFeeScheduleEntryResponse, 0, len(entries))
	for i := range entries {
		e := entries[i]
		isActive := activeByProvider[e.Provider] == e.ID
		isPending := e.EffectiveDate > today
		out = append(out, toFeeScheduleResponse(e, isActive, isPending))
	}

	response.Success(c, dto.DisbursementFeeScheduleListResponse{Entries: out}, "Lấy danh sách phí giao dịch chi hộ thành công")
}

// Create appends a new schedule entry. Validation (effective_date not past,
// non-negative fee) is enforced at the service layer.
func (h *FeeScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateDisbursementFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	actorID := actorUserID(c)
	created, err := h.service.Create(c.Request.Context(), disbursement.CreateInput{
		Provider:      req.Provider,
		EffectiveDate: req.EffectiveDate,
		FeeVND:        req.FeeVND,
		Notes:         req.Notes,
	}, actorID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	today := now.Format(domain.DisbursementFeeScheduleDateLayout)
	isPending := created.EffectiveDate > today
	isActive := !isPending // newly-created schedules with today's effective_date are active immediately

	response.SuccessCreated(c, toFeeScheduleResponse(*created, isActive, isPending), "Tạo cấu hình phí giao dịch chi hộ thành công")
}

// Update mutates a future-dated entry. Past or currently-active entries are
// immutable for audit integrity.
func (h *FeeScheduleHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.BadRequest(c, "id is required")
		return
	}

	var req dto.UpdateDisbursementFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), id, disbursement.UpdateInput{
		EffectiveDate: req.EffectiveDate,
		FeeVND:        req.FeeVND,
		Notes:         req.Notes,
	})
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	now := h.clock.Now()
	today := now.Format(domain.DisbursementFeeScheduleDateLayout)
	isPending := updated.EffectiveDate > today
	isActive := !isPending // not pending ⇒ must be active for its provider (dedup prevents same-provider same-date)
	response.Success(c, toFeeScheduleResponse(*updated, isActive, isPending), "Cập nhật cấu hình phí giao dịch chi hộ thành công")
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

	response.Success(c, gin.H{"id": id}, "Xóa cấu hình phí giao dịch chi hộ thành công")
}

func toFeeScheduleResponse(e domain.DisbursementFeeScheduleEntry, isActive, isPending bool) dto.DisbursementFeeScheduleEntryResponse {
	return dto.DisbursementFeeScheduleEntryResponse{
		ID:                e.ID,
		Provider:          e.Provider,
		EffectiveDate:     e.EffectiveDate,
		FeeVND:            e.FeeVND,
		Notes:             e.Notes,
		CreatedAt:         e.CreatedAt,
		CreatedByUserID:   e.CreatedByUserID,
		IsCurrentlyActive: isActive,
		IsPending:         isPending,
		Summary:           disbursement.FormatFeeAmount(e.FeeVND) + " VNĐ",
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
