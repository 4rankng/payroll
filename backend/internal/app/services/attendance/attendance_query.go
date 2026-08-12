package attendance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// GetTodayAttendance returns the attendance record for the employee for the
// current day. For night shifts checked in before midnight, it falls back to an
// open previous-day attendance so the mobile card can still show "Tan ca" after
// midnight. It returns nil if no current-day or open previous-day record exists.
func (s *AttendanceService) GetTodayAttendance(ctx context.Context, employeeID uint) (*domain.Attendance, error) {
	now := s.clock.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	att, err := s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, today)
	if err != nil {
		return nil, err
	}
	if att != nil {
		s.normalizeLegacySalaryRejectReason(ctx, att)
		if isAutoRejectedNoCheckout(att) {
			return nil, nil
		}
		return att, nil
	}

	yesterday := today.AddDate(0, 0, -1)
	att, err = s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, yesterday)
	if err != nil {
		return nil, err
	}
	if att == nil || att.CheckOutTime != nil || att.SalaryRejectReason != nil {
		return nil, nil
	}
	if att.IsApproved() {
		// Older approvals were stored as a logical completion only. Repair that
		// physical open row before returning the mobile read model so the employee
		// can immediately begin today's shift and the database no longer retains
		// a stale open check-in.
		if err := s.transactionManager.WithTransaction(ctx, func(txCtx context.Context) error {
			return s.completeLegacyApprovedOpenAttendance(txCtx, employeeID, today)
		}); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return att, nil
}

// GetByID returns an attendance record by its ID.
func (s *AttendanceService) GetByID(ctx context.Context, id uint) (*domain.Attendance, error) {
	att, err := s.attendanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.normalizeLegacySalaryRejectReason(ctx, att)
	return att, nil
}

// List returns a list of attendance records based on filters.
func (s *AttendanceService) List(ctx context.Context, filters domain.AttendanceFilters) ([]*domain.Attendance, int64, error) {
	attendances, err := s.attendanceRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, err
	}
	for _, att := range attendances {
		s.normalizeLegacySalaryRejectReason(ctx, att)
	}

	count, err := s.attendanceRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, err
	}

	return attendances, count, nil
}

func (s *AttendanceService) normalizeLegacySalaryRejectReason(ctx context.Context, att *domain.Attendance) {
	if att == nil || att.SalaryRejectReason == nil || att.CheckOutTime == nil {
		return
	}
	if !strings.Contains(*att.SalaryRejectReason, "không khớp trọn ca") {
		return
	}

	assignment, err := s.projectEmployeeRepo.GetActiveAssignmentByProjectAndEmployee(ctx, att.ProjectID, att.EmployeeID)
	if err != nil {
		observability.GetLogger().Warn("Failed to load assignment for salary reject reason normalization", "attendance_id", att.ID, "error", err)
		return
	}

	payrate, err := s.payrateRepo.GetActiveByProjectAndDate(ctx, att.ProjectID, att.Date)
	if err != nil {
		observability.GetLogger().Warn("Failed to load payrate for salary reject reason normalization", "attendance_id", att.ID, "error", err)
		return
	}

	amount, reason, err := s.calculateEarningAmount(payrate, assignment.Position, att.CheckInTime, *att.CheckOutTime)
	if err != nil {
		observability.GetLogger().Warn("Failed to normalize salary reject reason", "attendance_id", att.ID, "error", err)
		return
	}
	if amount == 0 && reason != "" {
		att.SalaryRejectReason = &reason
	}
}

func (s *AttendanceService) calculateEarningAmount(payrate *domain.Payrate, position string, ci, co time.Time) (int64, string, error) {
	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		return 0, "Cấu hình mức lương chưa hợp lệ, chưa thể ghi lương ca này.", err
	}

	effectivePosition, positionFound, shifts := resolveShifts(flattened, position, ci)

	// A shift earns its flat amount when the attendance fits BOTH the check-in
	// and checkout windows — the same predicates the gates use, so a checkout the
	// gate accepts is guaranteed to earn. Among matching shifts pick the one
	// nearest the check-in start (tie-broken by earlier start) so the payout is
	// deterministic regardless of map iteration order.
	var match *parsedShift
	var matchDistance time.Duration
	for i := range shifts {
		sh := &shifts[i]
		if !checkInFitsShift(sh, ci) || !checkOutFitsShift(sh, co) {
			continue
		}
		distance := ci.Sub(sh.start).Abs()
		if match == nil || distance < matchDistance || (distance == matchDistance && sh.start.Before(match.start)) {
			match, matchDistance = sh, distance
		}
	}
	if match != nil {
		if match.amount <= 0 {
			return 0, "Mức lương ca được cấu hình là 0đ. Vui lòng liên hệ quản lý.", nil
		}
		return int64(match.amount), "", nil
	}

	if !positionFound {
		return 0, fmt.Sprintf("Chưa có mức lương cho vị trí \"%s\".", position), nil
	}
	closest := closestShift(shifts, ci)
	if closest == nil {
		return 0, fmt.Sprintf("Chưa có ca làm hợp lệ trong cấu hình mức lương cho vị trí \"%s\".", effectivePosition), nil
	}
	return 0, fmt.Sprintf(
		"Thời gian vào %s và tan %s không hợp lệ. Bạn phải vào làm từ %s đến %s và tan ca từ %s đến %s.",
		ci.Format("15:04"),
		co.Format("15:04"),
		closest.start.Add(-checkInShiftWindow).Format("15:04"),
		closest.start.Add(checkInShiftWindow).Format("15:04"),
		closest.end.Add(-checkOutLowerGrace).Format("15:04"),
		closest.end.Add(checkOutUpperGrace).Format("15:04"),
	), nil
}
