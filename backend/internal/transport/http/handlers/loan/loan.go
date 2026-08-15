package loan

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/loan"
	"api-server/internal/pkg/clock"
)

type Handler struct {
	loanService *loan.LoanService
	clock       clock.Clock
}

func NewHandler(loanService *loan.LoanService, clk clock.Clock) *Handler {
	if clk == nil {
		clk = clock.New()
	}
	return &Handler{
		loanService: loanService,
		clock:       clk,
	}
}

// buildLoanResponse converts a single loan to a response DTO. It fetches the
// loan's schedules itself (used by create/update detail paths). The list path
// uses buildLoanResponses which batches the schedule fetch.
func (h *Handler) buildLoanResponse(ctx context.Context, loan *domain.Loan) dto.LoanResponse {
	var schedules []*domain.LoanRepaymentSchedule
	if _, s, err := h.loanService.GetLoanWithSchedules(ctx, loan.ID); err == nil {
		schedules = s
	}
	return h.populateLoanResponse(loan, schedules)
}

// buildLoanResponses converts a list of loans to response DTOs with a SINGLE
// batched schedule fetch (was: one GetLoanWithSchedules — i.e. two queries —
// per loan, an N+1 flagged by the ck:debug 2026-07-04 audit). Schedules are
// fetched once via GetSchedulesByLoanIDs and looked up per loan.
func (h *Handler) buildLoanResponses(ctx context.Context, loans []*domain.Loan) []dto.LoanResponse {
	loanIDs := make([]uint, len(loans))
	for i, l := range loans {
		loanIDs[i] = l.ID
	}
	scheduleMap, err := h.loanService.GetSchedulesByLoanIDs(ctx, loanIDs)
	if err != nil {
		// Degrade gracefully: build responses without schedule-derived fields
		// rather than failing the whole list. The error is logged upstream.
		scheduleMap = map[uint][]*domain.LoanRepaymentSchedule{}
	}
	responses := make([]dto.LoanResponse, len(loans))
	for i, loan := range loans {
		responses[i] = h.populateLoanResponse(loan, scheduleMap[loan.ID])
	}
	return responses
}

// populateLoanResponse maps a loan + its (already-fetched) schedules to a
// LoanResponse DTO. Shared by the single-loan and batch paths so they cannot
// drift. The schedules slice may be nil/empty (no schedules, or fetch failed).
func (h *Handler) populateLoanResponse(loan *domain.Loan, schedules []*domain.LoanRepaymentSchedule) dto.LoanResponse {
	lenderBrief := dto.LenderBriefResponse{
		ID:   loan.Lender.ID,
		Name: loan.Lender.Name,
	}
	if loan.Lender.Email != nil {
		lenderBrief.Email = loan.Lender.Email
	}

	var disbursementDatePtr *string
	if loan.DisbursedAt != nil {
		dateStr := loan.DisbursedAt.Format("2006-01-02")
		disbursementDatePtr = &dateStr
	}

	// Next payment date/amount: first pending schedule with a future due date.
	var nextPaymentDate *string
	var nextPaymentAmount *int64
	if loan.Status == domain.LoanStatusActive && loan.DisbursedAt != nil {
		now := h.clock.Now()
		for _, schedule := range schedules {
			if schedule.Status == domain.ScheduleStatusPending && schedule.DueDate.After(now) {
				dateStr := schedule.DueDate.Format("2006-01-02")
				nextPaymentDate = &dateStr
				nextPaymentAmount = &schedule.Amount
				break
			}
		}
	}

	return dto.LoanResponse{
		ID:                   loan.ID,
		LoanCode:             loan.LoanCode,
		Lender:               lenderBrief,
		PrincipalAmount:      loan.PrincipalAmount,
		OutstandingPrincipal: loan.OutstandingPrincipal,
		InterestRateBps:      loan.InterestRateBps,
		TotalInterestPaid:    loan.TotalInterestPaid,
		TermMonths:           loan.TermMonths,
		DisbursementDate:     disbursementDatePtr,
		NextPaymentDate:      nextPaymentDate,
		NextPaymentAmount:    nextPaymentAmount,
		PaymentDayOfMonth:    loan.PaymentDayOfMonth,
		Status:               string(loan.Status),
		Description:          loan.Description,
		CreatedAt:            loan.CreatedAt,
		UpdatedAt:            loan.UpdatedAt,
	}
}

// buildLoanDetailResponse converts loan to detailed response DTO with schedules
func (h *Handler) buildLoanDetailResponse(ctx context.Context, loan *domain.Loan) dto.LoanDetailResponse {
	lenderBrief := dto.LenderBriefResponse{
		ID:   loan.Lender.ID,
		Name: loan.Lender.Name,
	}
	if loan.Lender.Email != nil {
		lenderBrief.Email = loan.Lender.Email
	}

	// Set disbursement date pointer
	var disbursementDatePtr *string
	if loan.DisbursedAt != nil {
		dateStr := loan.DisbursedAt.Format("2006-01-02")
		disbursementDatePtr = &dateStr
	}

	// Calculate next payment date and amount for active loans that have been disbursed
	var nextPaymentDate *string
	var nextPaymentAmount *int64
	if loan.Status == domain.LoanStatusActive && loan.DisbursedAt != nil {
		// Get next payment schedule
		if _, schedules, err := h.loanService.GetLoanWithSchedules(ctx, loan.ID); err == nil {
			now := h.clock.Now()
			for _, schedule := range schedules {
				if schedule.Status == domain.ScheduleStatusPending && schedule.DueDate.After(now) {
					dateStr := schedule.DueDate.Format("2006-01-02")
					nextPaymentDate = &dateStr
					nextPaymentAmount = &schedule.Amount
					break
				}
			}
		}
	}

	// Fetch repayment schedules
	var schedules []dto.RepaymentScheduleResponse
	if _, scheduleList, err := h.loanService.GetLoanWithSchedules(ctx, loan.ID); err == nil {
		for _, schedule := range scheduleList {
			var paidAtStr *string
			if schedule.PaidAt != nil {
				paidAtStrVal := schedule.PaidAt.Format(time.RFC3339)
				paidAtStr = &paidAtStrVal
			}

			scheduleResponse := dto.RepaymentScheduleResponse{
				ID:              schedule.ID,
				Period:          schedule.Period,
				DueDate:         schedule.DueDate.Format("2006-01-02"),
				Amount:          schedule.Amount,
				PrincipalAmount: schedule.PrincipalAmount,
				InterestAmount:  schedule.InterestAmount,
				Status:          string(schedule.Status),
				PaidAt:          paidAtStr,
			}
			schedules = append(schedules, scheduleResponse)
		}
	}

	return dto.LoanDetailResponse{
		ID:                   loan.ID,
		LoanCode:             loan.LoanCode,
		Lender:               lenderBrief,
		PrincipalAmount:      loan.PrincipalAmount,
		OutstandingPrincipal: loan.OutstandingPrincipal,
		InterestRateBps:      loan.InterestRateBps,
		TotalInterestPaid:    loan.TotalInterestPaid,
		TermMonths:           loan.TermMonths,
		DisbursementDate:     disbursementDatePtr,
		NextPaymentDate:      nextPaymentDate,
		NextPaymentAmount:    nextPaymentAmount,
		PaymentDayOfMonth:    loan.PaymentDayOfMonth,
		Status:               string(loan.Status),
		Description:          loan.Description,
		Schedules:            schedules,
		CreatedAt:            loan.CreatedAt,
		UpdatedAt:            loan.UpdatedAt,
	}
}

// CreateLoan creates a new loan (supports both auto-interest and custom-schedule)
func (h *Handler) CreateLoan(c *gin.Context) {
	// Bind request body once
	var req dto.CreateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	if len(req.Schedules) == 0 {
		response.BadRequest(c, "Lịch thanh toán là bắt buộc")
		return
	}

	// Parse start date (acts as disbursement date for loan)
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	// Convert repayment schedules
	var repaymentSchedules []domain.LoanRepaymentSchedule
	for _, scheduleReq := range req.Schedules {
		dueDate, parseErr := time.Parse("2006-01-02", scheduleReq.DueDate)
		if parseErr != nil {
			response.BadRequest(c, constants.MsgInvalidDateFormatVN)
			return
		}

		repaymentSchedules = append(repaymentSchedules, domain.LoanRepaymentSchedule{
			DueDate: dueDate,
			Amount:  scheduleReq.Amount,
			Status:  domain.ScheduleStatusPending,
		})
	}

	loan := &domain.Loan{
		LenderID:        req.LenderID,
		PrincipalAmount: req.PrincipalAmount,
		StartDate:       startDate,
		Status:          domain.LoanStatusActive,
	}

	createdLoan, err := h.loanService.CreateCustomScheduleLoan(c.Request.Context(), loan, repaymentSchedules, userID.(uint))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể tạo khoản vay")
		return
	}

	// If disburse_now is true, disburse immediately
	if req.DisburseNow {
		disbursementDate := startDate
		reference := ""
		if req.DisbursementReference != nil {
			reference = *req.DisbursementReference
		}

		createdLoan, _, err = h.loanService.DisburseLoan(
			c.Request.Context(),
			createdLoan.ID,
			disbursementDate,
			reference,
			userID.(uint),
		)
		if err != nil {
			response.InternalServerError(c, "Khoản vay đã tạo nhưng giải ngân thất bại: "+err.Error())
			return
		}
	}

	loanResponse := h.buildLoanResponse(c.Request.Context(), createdLoan)
	response.SuccessCreated(c, loanResponse, constants.MsgCustomLoanCreatedVN)
}

// GetLoan retrieves a loan by ID
func (h *Handler) GetLoan(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	loan, err := h.loanService.GetLoan(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		response.InternalServerError(c, "Không thể lấy thông tin khoản vay")
		return
	}

	loanResponse := h.buildLoanDetailResponse(c.Request.Context(), loan)
	response.Success(c, loanResponse, constants.MsgLoanFetchedVN)
}

// ListLoans retrieves a paginated list of loans
func (h *Handler) ListLoans(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))
	sortBy := c.DefaultQuery("sortBy", "created_at")
	sortOrder := c.DefaultQuery("sortOrder", "DESC")

	// Validate pagination parameters
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	if page < 1 {
		page = 1
	}

	filters := domain.LoanFilters{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	// Filter by lender_id
	if lenderIDStr := c.Query("lender_id"); lenderIDStr != "" {
		lenderID, err := strconv.ParseUint(lenderIDStr, 10, 32)
		if err == nil {
			lenderIDUint := uint(lenderID)
			filters.LenderID = &lenderIDUint
		}
	}

	// Filter by status
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.LoanStatus(statusStr)
		filters.Status = &status
	}

	loans, total, err := h.loanService.ListLoans(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể lấy danh sách khoản vay")
		return
	}

	// Convert to response DTOs (batched schedule fetch — one query for all loans,
	// not N+1. See buildLoanResponses / GetSchedulesByLoanIDs.)
	loanResponses := h.buildLoanResponses(c.Request.Context(), loans)

	// Build pagination and return according to API spec
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	response.SuccessWithPagination(c, loanResponses, constants.MsgLoanListFetchedVN, pagination)
}

// DisburseLoan disburses a loan
func (h *Handler) DisburseLoan(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	var req dto.DisburseLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse disbursement date
	disbursementDate, err := time.Parse("2006-01-02", req.DisbursementDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	reference := ""
	if req.DisbursementReference != nil {
		reference = *req.DisbursementReference
	}

	loan, ledgerEntryIDs, err := h.loanService.DisburseLoan(
		c.Request.Context(),
		uint(id),
		disbursementDate,
		reference,
		userID.(uint),
	)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể giải ngân khoản vay")
		return
	}

	disburseResponse := dto.DisburseLoanResponse{
		LoanID:          loan.ID,
		DisbursedAmount: loan.PrincipalAmount,
		LedgerEntryIDs:  ledgerEntryIDs,
	}

	response.Success(c, disburseResponse, constants.MsgLoanDisbursedVN)
}

// RepayPrincipal makes a principal repayment
func (h *Handler) RepayPrincipal(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	var req dto.RepayLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse payment date
	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	reference := ""
	if req.PaymentReference != nil {
		reference = *req.PaymentReference
	}

	loan, ledgerEntryIDs, err := h.loanService.RepayPrincipal(
		c.Request.Context(),
		uint(id),
		req.Amount,
		paymentDate,
		reference,
		req.Notes,
		userID.(uint),
	)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể trả nợ gốc")
		return
	}

	repayResponse := dto.RepayLoanResponse{
		LoanID:               loan.ID,
		RepaidAmount:         req.Amount,
		OutstandingPrincipal: loan.OutstandingPrincipal,
		Status:               string(loan.Status),
		LedgerEntryIDs:       ledgerEntryIDs,
	}

	response.Success(c, repayResponse, constants.MsgLoanRepaidVN)
}

// GetSchedule retrieves the payment schedule for a loan
func (h *Handler) GetSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	schedule, err := h.loanService.GetLoanSchedule(c.Request.Context(), uint(id))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		response.InternalServerError(c, "Không thể lấy lịch thanh toán")
		return
	}

	// Convert to response DTOs
	scheduleResponses := make([]dto.ScheduleItemResponse, len(schedule))
	for i, item := range schedule {
		scheduleResponses[i] = dto.ScheduleItemResponse{
			Period:  item.Period,
			DueDate: item.DueDate.Format("2006-01-02"),
			Type:    item.Type,
			Amount:  item.Amount,
			Status:  item.Status,
		}
	}

	response.Success(c, scheduleResponses, constants.MsgLoanScheduleFetchedVN)
}

// UpdateLoan updates loan metadata
func (h *Handler) UpdateLoan(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	var req dto.UpdateLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	// Build update data map
	updateData := make(map[string]interface{})
	if req.Description != nil {
		updateData["description"] = *req.Description
	}
	if req.PaymentDayOfMonth != nil {
		updateData["payment_day_of_month"] = *req.PaymentDayOfMonth
	}

	updatedLoan, err := h.loanService.UpdateLoan(c.Request.Context(), uint(id), updateData)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể cập nhật khoản vay")
		return
	}

	loanResponse := h.buildLoanResponse(c.Request.Context(), updatedLoan)
	response.Success(c, loanResponse, constants.MsgLoanUpdatedVN)
}

// DeleteLoan deletes a loan that has not yet been disbursed
func (h *Handler) DeleteLoan(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	if err := h.loanService.DeleteLoan(c.Request.Context(), uint(id)); err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể xóa khoản vay")
		return
	}

	response.Success(c, nil, constants.MsgLoanDeletedVN)
}

// CreateCustomScheduleLoan creates a new loan with custom repayment schedule
func (h *Handler) CreateCustomScheduleLoan(c *gin.Context) {
	var req dto.CreateCustomScheduleLoanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse disbursement date
	disbursementDate, err := time.Parse("2006-01-02", req.DisbursementDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	// Convert repayment schedules
	var repaymentSchedules []domain.LoanRepaymentSchedule
	for _, scheduleReq := range req.Schedules {
		dueDate, err := time.Parse("2006-01-02", scheduleReq.DueDate)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidDateFormatVN)
			return
		}

		repaymentSchedules = append(repaymentSchedules, domain.LoanRepaymentSchedule{
			DueDate: dueDate,
			Amount:  scheduleReq.Amount,
			Status:  domain.ScheduleStatusPending,
		})
	}

	loan := &domain.Loan{
		LenderID:        req.LenderID,
		PrincipalAmount: req.PrincipalAmount,
		StartDate:       disbursementDate, // For custom schedules, start date is disbursement date
		Status:          domain.LoanStatusActive,
	}

	// If disburse_now is true, set outstanding principal to principal amount
	if req.DisburseNow {
		loan.OutstandingPrincipal = loan.PrincipalAmount
		loan.DisbursedAt = &disbursementDate
	}

	createdLoan, err := h.loanService.CreateCustomScheduleLoan(c.Request.Context(), loan, repaymentSchedules, userID.(uint))
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "Không thể tạo khoản vay lịch tùy chỉnh")
		return
	}

	loanResponse := h.buildLoanResponse(c.Request.Context(), createdLoan)
	response.SuccessCreated(c, loanResponse, constants.MsgCustomLoanCreatedVN)
}

// ProcessScheduledPayment processes a scheduled payment
func (h *Handler) ProcessScheduledPayment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidLoanIDVN)
		return
	}

	var req dto.ProcessScheduledPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN+": "+err.Error())
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse payment date
	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidDateFormatVN)
		return
	}

	// Handle payment reference as string
	paymentReference := ""
	if req.PaymentReference != nil {
		paymentReference = *req.PaymentReference
	}

	loan, ledgerEntryIDs, err := h.loanService.ProcessScheduledPayment(
		c.Request.Context(),
		uint(id),
		req.ScheduleID,
		paymentDate,
		paymentReference,
		req.Notes,
		userID.(uint),
	)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, constants.MsgLoanNotFoundVN)
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		// Log the actual error for debugging
		slog.Error("ProcessScheduledPayment error", "loanID", id, "scheduleID", req.ScheduleID, "error", err)
		response.InternalServerError(c, "Không thể xử lý thanh toán theo lịch")
		return
	}

	// Get the schedule to get the correct amount paid
	schedule, err := h.loanService.RepaymentScheduleRepo.GetByID(c.Request.Context(), req.ScheduleID)
	if err != nil {
		response.InternalServerError(c, "Không thể lấy thông tin lịch thanh toán")
		return
	}

	paymentResponse := dto.ProcessScheduledPaymentResponse{
		LoanID:               loan.ID,
		SchedulePeriod:       schedule.Period,
		PaidAmount:           schedule.Amount,
		OutstandingPrincipal: loan.OutstandingPrincipal,
		Status:               string(loan.Status),
		LedgerEntryIDs:       ledgerEntryIDs,
	}

	response.Success(c, paymentResponse, constants.MsgScheduledPaymentProcessedVN)
}
