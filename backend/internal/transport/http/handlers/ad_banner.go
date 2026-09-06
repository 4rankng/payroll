package handlers

import (
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/ad_banner"
	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// AdBannerHandler serves the admin campaign CRUD and the employee resolve +
// click endpoints. Admin routes are additionally gated upstream by an explicit
// Authorize() group (admin has the /api/* Casbin wildcard; partners are
// excluded by the absence of an allow rule).
type AdBannerHandler struct {
	service *ad_banner.Service
	logger  *slog.Logger
}

func NewAdBannerHandler(service *ad_banner.Service, logger *slog.Logger) *AdBannerHandler {
	return &AdBannerHandler{service: service, logger: logger}
}

// GetMyAdBanner resolves the single campaign addressing the authenticated
// employee, or null when none does.
func (h *AdBannerHandler) GetMyAdBanner(c *gin.Context) {
	userID := h.requireUserID(c)
	if userID == 0 {
		return
	}

	banner, err := h.service.ResolveForEmployeeByUser(c.Request.Context(), userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	if banner == nil {
		response.Success(c, nil, "Không có quảng cáo")
		return
	}
	response.Success(c, toAdBannerResponse(*banner, nil), "Lấy quảng cáo thành công")
}

// RecordClick records a CTA tap. Failures are logged and swallowed: a
// click-tracking outage must never block a worker from reaching the hotline.
func (h *AdBannerHandler) RecordClick(c *gin.Context) {
	userID := h.requireUserID(c)
	if userID == 0 {
		return
	}

	bannerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "id không hợp lệ")
		return
	}

	var req dto.AdBannerClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	if err := h.service.RecordCTAClick(c.Request.Context(), uint(bannerID), userID, req.CTAIndex); err != nil {
		h.logger.Warn("Failed to record ad banner click",
			"banner_id", bannerID, "user_id", userID, "cta_index", req.CTAIndex, "error", err)
	}
	response.Success(c, nil, "Đã ghi nhận")
}

// List returns every campaign with per-CTA click counts.
func (h *AdBannerHandler) List(c *gin.Context) {
	banners, counts, err := h.service.ListWithStats(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	out := make([]dto.AdBannerResponse, 0, len(banners))
	for _, b := range banners {
		out = append(out, toAdBannerResponse(*b, counts[b.ID]))
	}
	response.Success(c, dto.AdBannerListResponse{Banners: out}, "Lấy danh sách chiến dịch thành công")
}

// Create publishes a new campaign.
func (h *AdBannerHandler) Create(c *gin.Context) {
	var req dto.CreateAdBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	banner := adBannerFromRequest(req.Title, req.Body, req.Bullets, req.CTAs, req.Footer,
		req.TargetProjectIDs, req.Priority, req.StartsAt, req.EndsAt, req.IsActive)
	banner.CreatedBy = actorUserIDFromContext(c)

	created, err := h.service.Create(c.Request.Context(), banner)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.SuccessCreated(c, toAdBannerResponse(*created, nil), "Tạo chiến dịch quảng cáo thành công")
}

// Update replaces a campaign's content and window. The updated_at bump is the
// new campaign version: employees see the sheet once more.
func (h *AdBannerHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "id không hợp lệ")
		return
	}

	var req dto.UpdateAdBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	updated := adBannerFromRequest(req.Title, req.Body, req.Bullets, req.CTAs, req.Footer,
		req.TargetProjectIDs, req.Priority, req.StartsAt, req.EndsAt, req.IsActive)

	result, err := h.service.Update(c.Request.Context(), uint(id), updated)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, toAdBannerResponse(*result, nil), "Cập nhật chiến dịch quảng cáo thành công")
}

// Delete soft-deletes a campaign.
func (h *AdBannerHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "id không hợp lệ")
		return
	}

	if err := h.service.Delete(c.Request.Context(), uint(id)); err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, gin.H{"id": uint(id)}, "Xóa chiến dịch quảng cáo thành công")
}

// requireUserID extracts the authenticated user id or writes a 403.
func (h *AdBannerHandler) requireUserID(c *gin.Context) uint {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return 0
	}
	return *userIDPtr
}

func actorUserIDFromContext(c *gin.Context) uint {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		return 0
	}
	return *userIDPtr
}

// adBannerFromRequest maps a create/update request onto the domain entity.
// A nil isActive means "active" — new campaigns default to running.
func adBannerFromRequest(title, body string, bullets []string, ctas []dto.AdBannerCTADTO, footer string,
	targetProjectIDs []uint, priority int, startsAt, endsAt time.Time, isActive *bool) *domain.AdBanner {
	domainCTAs := make([]domain.AdBannerCTA, len(ctas))
	for i, cta := range ctas {
		domainCTAs[i] = domain.AdBannerCTA{
			Label: cta.Label,
			Type:  domain.AdBannerCTAType(cta.Type),
			Value: cta.Value,
		}
	}

	active := true
	if isActive != nil {
		active = *isActive
	}

	return &domain.AdBanner{
		Title:            title,
		Body:             body,
		Bullets:          bullets,
		CTAs:             domainCTAs,
		Footer:           footer,
		TargetProjectIDs: targetProjectIDs,
		Priority:         priority,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		IsActive:         active,
	}
}

func toAdBannerResponse(b domain.AdBanner, clickCounts map[int]int64) dto.AdBannerResponse {
	ctaDTOs := make([]dto.AdBannerCTADTO, len(b.CTAs))
	for i, cta := range b.CTAs {
		ctaDTOs[i] = dto.AdBannerCTADTO{Label: cta.Label, Type: string(cta.Type), Value: cta.Value}
	}

	var counts map[string]int64
	if len(clickCounts) > 0 {
		counts = make(map[string]int64, len(clickCounts))
		for idx, cnt := range clickCounts {
			counts[strconv.Itoa(idx)] = cnt
		}
	}

	return dto.AdBannerResponse{
		ID:               b.ID,
		Title:            b.Title,
		Body:             b.Body,
		Bullets:          b.Bullets,
		CTAs:             ctaDTOs,
		Footer:           b.Footer,
		TargetProjectIDs: b.TargetProjectIDs,
		Priority:         b.Priority,
		StartsAt:         b.StartsAt,
		EndsAt:           b.EndsAt,
		IsActive:         b.IsActive,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		ClickCounts:      counts,
	}
}
