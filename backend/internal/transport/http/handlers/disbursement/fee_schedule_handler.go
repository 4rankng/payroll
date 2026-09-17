package disbursement

import (
	"context"
	"strings"

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
	// enabledProviders reports which disbursement providers are registered
	// (enabled in config). List filters out disabled providers so admins only
	// see fee history relevant to this environment; Create rejects them.
	enabledProviders func(ctx context.Context) []string
}

// NewFeeScheduleHandler wires the handler with the schedule service and the
// enabled-provider lookup. enabledProviders may be nil — the handler then
// shows all providers (used by tests and legacy wiring).
func NewFeeScheduleHandler(service *disbursement.FeeScheduleService, clk clock.Clock, enabledProviders func(ctx context.Context) []string) *FeeScheduleHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &FeeScheduleHandler{service: service, clock: clk, enabledProviders: enabledProviders}
}

// filterEnabled keeps only entries whose provider is enabled. A nil or
// empty lookup keeps everything so no wiring change can silently hide data.
func (h *FeeScheduleHandler) filterEnabled(ctx context.Context, entries []domain.DisbursementFeeScheduleEntry) []domain.DisbursementFeeScheduleEntry {
	if h.enabledProviders == nil {
		return entries
	}
	enabled := h.enabledProviders(ctx)
	if len(enabled) == 0 {
		return entries
	}
	allowed := make(map[string]bool, len(enabled))
	for _, p := range enabled {
		allowed[strings.ToLower(p)] = true
	}
	filtered := make([]domain.DisbursementFeeScheduleEntry, 0, len(entries))
	for _, e := range entries {
		if allowed[strings.ToLower(e.Provider)] {
			filtered = append(filtered, e)
		}
	}
	return filtered
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

	entries = h.filterEnabled(c.Request.Context(), entries)

	// Enabled providers ride along so the admin form only offers them.
	enabledProviders := []string(nil)
	if h.enabledProviders != nil {
		enabledProviders = h.enabledProviders(c.Request.Context())
	}

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

	response.Success(c, dto.DisbursementFeeScheduleListResponse{Entries: out, Providers: enabledProviders}, "Lấy danh sách phí giao dịch chi hộ thành công")
}

// Create appends a new schedule entry. Validation (effective_date not past,
// non-negative fee) is enforced at the service layer.
func (h *FeeScheduleHandler) Create(c *gin.Context) {
	var req dto.CreateDisbursementFeeScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	if !h.providerEnabled(c.Request.Context(), req.Provider) {
		response.BadRequest(c, "Nhà cung cấp "+req.Provider+" chưa được kích hoạt trên hệ thống")
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

// providerEnabled reports whether the given provider is enabled. When no
// lookup is wired every provider passes so legacy wiring keeps working.
func (h *FeeScheduleHandler) providerEnabled(ctx context.Context, provider string) bool {
	if h.enabledProviders == nil {
		return true
	}
	enabled := h.enabledProviders(ctx)
	if len(enabled) == 0 {
		return true
	}
	for _, p := range enabled {
		if strings.EqualFold(p, provider) {
			return true
		}
	}
	return false
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
