package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPayrateTemporalServiceUsesPaidCutoffAndRecalculatesOnlyMutableTimesheets(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	repository := &payrateTemporalRepositoryStub{}
	service := NewPayrateTemporalService(db, repository)

	const projectID uint = 41
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)

	paidDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local)
	fromDate := paidDate.AddDate(0, 0, 1)
	timesheets := []*domain.Timesheet{
		newPayrateTemporalTimesheet(projectID, paidDate, domain.TimesheetStatusApproved, domain.PaymentStatusPaid),
		newPayrateTemporalTimesheet(projectID, fromDate, domain.TimesheetStatusApproved, domain.PaymentStatusPending),
		newPayrateTemporalTimesheet(projectID, fromDate.AddDate(0, 0, 1), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending),
		newPayrateTemporalTimesheet(projectID, fromDate.AddDate(0, 0, 3), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending),
		newPayrateTemporalTimesheet(projectID, fromDate.AddDate(0, 0, 2), domain.TimesheetStatusRejected, domain.PaymentStatusFailed),
		newPayrateTemporalTimesheet(projectID, paidDate.AddDate(0, 0, -1), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending),
	}
	for _, timesheet := range timesheets {
		seedPayrateTemporalTimesheet(t, db, timesheet)
	}

	latestPaidDate, err := service.GetLatestPaidTimesheetDateForProject(ctx, projectID)
	require.NoError(t, err)
	require.NotNil(t, latestPaidDate)
	require.True(t, latestPaidDate.Equal(paidDate), "latest paid %v != seeded %v", latestPaidDate, paidDate)

	require.Error(t, db.Transaction(func(tx *gorm.DB) error {
		return service.validateEffectiveDateTx(ctx, tx, projectID, paidDate)
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return service.validateEffectiveDateTx(ctx, tx, projectID, fromDate)
	}))

	payrate := &domain.Payrate{
		ID:        99,
		ProjectID: projectID,
		FromDate:  fromDate,
		Payrate: domain.PayrateConfiguration(
			`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`,
		),
	}
	require.NoError(t, service.CreateEffectiveDatedPayrate(ctx, payrate))

	assertTimesheetRateUnchanged(t, db, timesheets[0].ID)
	assertTimesheetRateUnchanged(t, db, timesheets[1].ID)
	assertTimesheetRateUpdated(t, db, timesheets[2].ID, payrate.ID)
	assertTimesheetRateUpdated(t, db, timesheets[3].ID, payrate.ID)
	assertTimesheetRateUpdated(t, db, timesheets[4].ID, payrate.ID)
	assertTimesheetRateUnchanged(t, db, timesheets[5].ID)
}

func TestPayrateTemporalServiceUsesLatestPaidTimesheetWorkDateForEffectiveDate(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 48
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)

	// Work on Aug 14 was paid on Aug 17. Its work date—not the later transfer
	// timestamp—sets the new payrate boundary, so Aug 15 is allowed.
	paid := newPayrateTemporalTimesheet(
		projectID,
		time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local),
		domain.TimesheetStatusApproved,
		domain.PaymentStatusPaid,
	)
	paidAt := time.Date(2026, 8, 17, 10, 30, 0, 0, time.Local)
	paid.PaidAt = &paidAt
	seedPayrateTemporalTimesheet(t, db, paid)

	err := db.Transaction(func(tx *gorm.DB) error {
		return service.validateEffectiveDateTx(ctx, tx, projectID, paid.Date)
	})
	require.EqualError(t, err, "Không thể cập nhật mức lương: ngày hiệu lực phải sau ngày chấm công đã trả lương gần nhất")

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return service.validateEffectiveDateTx(ctx, tx, projectID, paid.Date.AddDate(0, 0, 1))
	}))
}

func TestPayrateTemporalServiceRejectsDuplicateCalendarDateAcrossTimezones(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()

	const projectID uint = 49
	storedDate := time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local)
	requestDate := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	repository := &payrateTemporalRepositoryStub{
		activePayrates: []*domain.Payrate{{ProjectID: projectID, FromDate: storedDate}},
	}
	service := NewPayrateTemporalService(db, repository)

	err := db.Transaction(func(tx *gorm.DB) error {
		return service.manageActivePayrates(ctx, tx, projectID, requestDate)
	})

	require.EqualError(t, err, "Đã tồn tại mức lương với ngày hiệu lực 2026-08-22")
}

func TestPayrateTemporalServiceUsesConfiguredTotalForFlexibleProjects(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 42
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, true).Error)
	timesheet := newPayrateTemporalTimesheet(
		projectID,
		time.Date(2026, 8, 12, 0, 0, 0, 0, time.Local),
		domain.TimesheetStatusPendingApproval,
		domain.PaymentStatusPending,
	)
	seedPayrateTemporalTimesheet(t, db, timesheet)

	payrate := &domain.Payrate{
		ID:        100,
		ProjectID: projectID,
		FromDate:  timesheet.Date,
		Payrate: domain.PayrateConfiguration(
			`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`,
		),
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return service.recalculateMutableTimesheets(ctx, tx, payrate)
	}))

	var persisted domain.Timesheet
	require.NoError(t, db.First(&persisted, timesheet.ID).Error)
	require.Equal(t, int64(250000), persisted.Amount)
}

func TestPayrateTemporalServiceRollsBackMutableTimesheetsWhenNewConfigDoesNotCoverPayType(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	repository := &payrateTemporalRepositoryStub{}
	service := NewPayrateTemporalService(db, repository)

	const projectID uint = 43
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	fromDate := time.Date(2026, 8, 12, 0, 0, 0, 0, time.Local)
	valid := newPayrateTemporalTimesheet(projectID, fromDate, domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending)
	unsupported := newPayrateTemporalTimesheet(projectID, fromDate.AddDate(0, 0, 1), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending)
	unsupported.PayType = "Công nhân.ngày thường.Ca đêm"
	seedPayrateTemporalTimesheet(t, db, valid)
	seedPayrateTemporalTimesheet(t, db, unsupported)

	payrate := &domain.Payrate{
		ProjectID: projectID,
		FromDate:  fromDate,
		Payrate: domain.PayrateConfiguration(
			`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`,
		),
	}
	require.Error(t, service.CreateEffectiveDatedPayrate(ctx, payrate))

	assertTimesheetRateUnchanged(t, db, valid.ID)
	assertTimesheetRateUnchanged(t, db, unsupported.ID)
}

func TestPayrateTemporalServiceMovesStartDateEarlierAndRecalculatesMutableTimesheets(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 44
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)

	// Sibling config still current, its immutable rows end Aug 7.
	sibling := &domain.Payrate{
		ID: 7, ProjectID: projectID,
		FromDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		ToDate:   func() *time.Time { t := time.Date(2026, 8, 7, 0, 0, 0, 0, time.Local); return &t }(),
	}
	// Target config created too late; admin needs to move it back to Aug 8.
	target := &domain.Payrate{
		ID: 8, ProjectID: projectID,
		FromDate: time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local),
	}
	seedPayrateTemporalPayrate(t, db, sibling)
	seedPayrateTemporalPayrate(t, db, target)

	paidDate := time.Date(2026, 8, 7, 0, 0, 0, 0, time.Local)
	newStart := paidDate.AddDate(0, 0, 1)
	earlierPending := newPayrateTemporalTimesheet(projectID, newStart, domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending)
	approvedPending := newPayrateTemporalTimesheet(projectID, newStart.AddDate(0, 0, 1), domain.TimesheetStatusApproved, domain.PaymentStatusPending)
	seedPayrateTemporalTimesheet(t, db, earlierPending)
	seedPayrateTemporalTimesheet(t, db, approvedPending)
	seedPayrateTemporalTimesheet(t, db, newPayrateTemporalTimesheet(projectID, paidDate, domain.TimesheetStatusApproved, domain.PaymentStatusPaid))

	// Moving the start date back to the day after the last paid row must pass.
	target.FromDate = newStart
	target.Payrate = domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`)
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	assertTimesheetRateUpdated(t, db, earlierPending.ID, target.ID)
	// Approved-but-unpaid rows stay immutable.
	assertTimesheetRateUnchanged(t, db, approvedPending.ID)

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, target.ID).Error)
	require.True(t, persisted.FromDate.Equal(newStart))
}

func TestPayrateTemporalServiceMovingLatestRateSynchronizesPredecessorEndDate(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 50
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)

	previousEnd := time.Date(2026, 8, 21, 0, 0, 0, 0, time.Local)
	previous := &domain.Payrate{
		ID: 13, ProjectID: projectID,
		FromDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		ToDate:   &previousEnd,
	}
	target := &domain.Payrate{
		ID: 14, ProjectID: projectID,
		FromDate: time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local),
	}
	seedPayrateTemporalPayrate(t, db, previous)
	seedPayrateTemporalPayrate(t, db, target)

	target.FromDate = time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persistedPrevious domain.Payrate
	require.NoError(t, db.First(&persistedPrevious, previous.ID).Error)
	require.NotNil(t, persistedPrevious.ToDate)
	require.Equal(t, "2026-08-14", persistedPrevious.ToDate.Format("2006-01-02"))
}

func TestPayrateTemporalServiceRejectsStartDateOnOrBeforeLatestPaidTimesheet(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 45
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	target := &domain.Payrate{
		ID: 9, ProjectID: projectID,
		FromDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local),
	}
	seedPayrateTemporalPayrate(t, db, target)

	paidDate := time.Date(2026, 8, 7, 0, 0, 0, 0, time.Local)
	seedPayrateTemporalTimesheet(t, db, newPayrateTemporalTimesheet(projectID, paidDate, domain.TimesheetStatusApproved, domain.PaymentStatusPaid))
	seedPayrateTemporalTimesheet(t, db, newPayrateTemporalTimesheet(projectID, paidDate.AddDate(0, 0, 1), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending))

	target.FromDate = paidDate
	require.Error(t, service.UpdateEffectiveDatedPayrate(ctx, target))
}

func TestPayrateTemporalServiceRejectsUpdateWhenSiblingStartsLater(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 46
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	sibling := &domain.Payrate{ID: 10, ProjectID: projectID, FromDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local)}
	target := &domain.Payrate{ID: 11, ProjectID: projectID, FromDate: time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local)}
	seedPayrateTemporalPayrate(t, db, sibling)
	seedPayrateTemporalPayrate(t, db, target)

	// Moving the target before its sibling's start would demote it from the
	// project's most recent config, so the sibling would silently reclaim dates.
	target.FromDate = time.Date(2026, 8, 8, 0, 0, 0, 0, time.Local)
	require.Error(t, service.UpdateEffectiveDatedPayrate(ctx, target))
}

func TestPayrateTemporalServiceSplitsConfigWhenStartDateMovesLater(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	repository := &payrateTemporalRepositoryStub{}
	service := NewPayrateTemporalService(db, repository)

	const projectID uint = 47
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	target := &domain.Payrate{
		ID:        12,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":100000}}}`),
	}
	seedPayrateTemporalPayrate(t, db, target)

	// Aug 15–21 were worked, approved, and paid under this config.
	paid := newPayrateTemporalTimesheet(projectID, time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local), domain.TimesheetStatusApproved, domain.PaymentStatusPaid)
	paid.PayrateID = target.ID
	seedPayrateTemporalTimesheet(t, db, paid)
	paidLater := newPayrateTemporalTimesheet(projectID, time.Date(2026, 8, 21, 0, 0, 0, 0, time.Local), domain.TimesheetStatusApproved, domain.PaymentStatusPaid)
	paidLater.PayrateID = target.ID
	seedPayrateTemporalTimesheet(t, db, paidLater)

	// Aug 22 onward is uploaded but still mutable, linked to the same config.
	pending := newPayrateTemporalTimesheet(projectID, time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending)
	pending.PayrateID = target.ID
	seedPayrateTemporalTimesheet(t, db, pending)

	// Moving the start to Aug 22 (the day after the last paid row) must treat
	// the update as a create: the old config closes at Aug 21, a new open
	// config takes over from Aug 22, paid rows stay on the old config, and
	// mutable rows are re-linked and re-priced under the new one.
	target.FromDate = time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local)
	target.Payrate = domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`)
	oldID := target.ID
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, oldID).Error)
	require.NotNil(t, persisted.ToDate)
	require.Equal(t, "2026-08-21", persisted.ToDate.Format("2006-01-02"))

	// The handler carries the new config's identity after the split.
	require.Equal(t, uint(99), target.ID)

	assertTimesheetRateOnConfig(t, db, paid.ID, 12, 100000)
	assertTimesheetRateOnConfig(t, db, paidLater.ID, 12, 100000)
	assertTimesheetRateOnConfig(t, db, pending.ID, 99, 250000)
}

func TestPayrateTemporalServiceSplitsConfigWithNoLinkedTimesheets(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	repository := &payrateTemporalRepositoryStub{}
	service := NewPayrateTemporalService(db, repository)

	const projectID uint = 52
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	target := &domain.Payrate{
		ID:        16,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	}
	seedPayrateTemporalPayrate(t, db, target)

	// No timesheets at all: the split still applies, recalculation is a no-op.
	target.FromDate = time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, 16).Error)
	require.NotNil(t, persisted.ToDate)
	require.Equal(t, "2026-08-03", persisted.ToDate.Format("2006-01-02"))
	require.Equal(t, uint(99), target.ID, "caller receives the new config identity")
}

func TestPayrateTemporalServiceRejectsSplitOfClosedConfig(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	repository := &payrateTemporalRepositoryStub{}
	service := NewPayrateTemporalService(db, repository)

	const projectID uint = 51
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	endDate := time.Date(2026, 8, 5, 0, 0, 0, 0, time.Local)
	closed := &domain.Payrate{
		ID:        15,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		ToDate:    &endDate,
	}
	seedPayrateTemporalPayrate(t, db, closed)

	// Ended configs are history: any start-date move — later, unchanged, or
	// earlier — is rejected with the dedicated message, and nothing changes.
	for name, from := range map[string]time.Time{
		"later":   time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local),
		"earlier": time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local),
	} {
		t.Run(name, func(t *testing.T) {
			target := *closed
			target.FromDate = from
			err := service.UpdateEffectiveDatedPayrate(ctx, &target)
			require.EqualError(t, err, "Không thể cập nhật cấu hình lương đã kết thúc — hãy chỉnh sửa cấu hình lương hiện hành")

			var persisted domain.Payrate
			require.NoError(t, db.First(&persisted, closed.ID).Error)
			require.True(t, persisted.FromDate.Equal(closed.FromDate))
			require.NotNil(t, persisted.ToDate)
			require.True(t, persisted.ToDate.Equal(*closed.ToDate))
		})
	}
}

func assertTimesheetRateOnConfig(t *testing.T, db *gorm.DB, timesheetID, payrateID uint, rate int64) {
	t.Helper()
	var timesheet domain.Timesheet
	require.NoError(t, db.First(&timesheet, timesheetID).Error)
	require.Equal(t, payrateID, timesheet.PayrateID)
	require.Equal(t, rate, timesheet.PayRate)
}

func seedPayrateTemporalPayrate(t *testing.T, db *gorm.DB, payrate *domain.Payrate) {
	t.Helper()
	require.NoError(t, db.Exec(`
		INSERT INTO payrates (id, project_id, from_date, to_date, payrate_json, created_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, payrate.ID, payrate.ProjectID, payrate.FromDate, payrate.ToDate, string(payrate.Payrate), 1).Error)
}

type payrateTemporalRepositoryStub struct {
	domain.PayrateRepository
	activePayrates []*domain.Payrate
}

func (s *payrateTemporalRepositoryStub) FindActiveByProject(context.Context, any, uint) ([]*domain.Payrate, error) {
	return s.activePayrates, nil
}

func (s *payrateTemporalRepositoryStub) CloseActiveByProject(context.Context, any, uint, time.Time) error {
	return nil
}

func (s *payrateTemporalRepositoryStub) CreateWithTx(_ context.Context, _ any, payrate *domain.Payrate) error {
	payrate.ID = 99
	return nil
}

func newPayrateTemporalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", databaseName)), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY,
			is_flexible BOOLEAN NOT NULL DEFAULT FALSE,
			deleted_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE payrates (
			id INTEGER PRIMARY KEY,
			project_id INTEGER NOT NULL,
			from_date DATETIME NOT NULL,
			to_date DATETIME,
			payrate_json TEXT NOT NULL,
			created_by INTEGER NOT NULL,
			deleted_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE timesheets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			employee_id INTEGER NOT NULL,
			payrate_id INTEGER NOT NULL,
			date DATETIME NOT NULL,
			hours_worked REAL NOT NULL,
			paytype TEXT NOT NULL,
			payrate INTEGER NOT NULL,
			amount INTEGER NOT NULL,
			timesheet_status TEXT NOT NULL,
			payment_status TEXT NOT NULL,
			paid_at DATETIME,
			created_by INTEGER NOT NULL,
			deleted_at DATETIME,
			updated_at DATETIME
		)
	`).Error)
	// cache=shared keeps the in-memory DB alive while any pooled connection
	// remains open; without this, repeat runs in one process (go test -count>1)
	// hit "table projects already exists".
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newPayrateTemporalTimesheet(projectID uint, date time.Time, status domain.TimesheetStatus, paymentStatus domain.PaymentStatus) *domain.Timesheet {
	return &domain.Timesheet{
		ProjectID:     projectID,
		EmployeeID:    1,
		PayrateID:     1,
		Date:          date,
		HoursWorked:   8,
		PayType:       "Công nhân.ngày thường.08:00-18:00",
		PayRate:       100000,
		Amount:        800000,
		Status:        status,
		PaymentStatus: paymentStatus,
		CreatedBy:     1,
	}
}

func seedPayrateTemporalTimesheet(t *testing.T, db *gorm.DB, timesheet *domain.Timesheet) {
	t.Helper()
	result := db.Exec(`
		INSERT INTO timesheets (
			project_id, employee_id, payrate_id, date, hours_worked, paytype,
			payrate, amount, timesheet_status, payment_status, paid_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		timesheet.ProjectID,
		timesheet.EmployeeID,
		timesheet.PayrateID,
		timesheet.Date,
		timesheet.HoursWorked,
		timesheet.PayType,
		timesheet.PayRate,
		timesheet.Amount,
		timesheet.Status,
		timesheet.PaymentStatus,
		timesheet.PaidAt,
		timesheet.CreatedBy,
	)
	require.NoError(t, result.Error)
	timesheet.ID = uint(result.RowsAffected)

	var persisted domain.Timesheet
	require.NoError(t, db.Where("project_id = ? AND date = ?", timesheet.ProjectID, timesheet.Date).First(&persisted).Error)
	timesheet.ID = persisted.ID
}

func assertTimesheetRateUnchanged(t *testing.T, db *gorm.DB, timesheetID uint) {
	t.Helper()
	var timesheet domain.Timesheet
	require.NoError(t, db.First(&timesheet, timesheetID).Error)
	require.Equal(t, uint(1), timesheet.PayrateID)
	require.Equal(t, int64(100000), timesheet.PayRate)
	require.Equal(t, int64(800000), timesheet.Amount)
}

func assertTimesheetRateUpdated(t *testing.T, db *gorm.DB, timesheetID, payrateID uint) {
	t.Helper()
	var timesheet domain.Timesheet
	require.NoError(t, db.First(&timesheet, timesheetID).Error)
	require.Equal(t, payrateID, timesheet.PayrateID)
	require.Equal(t, int64(250000), timesheet.PayRate)
	require.Equal(t, int64(2000000), timesheet.Amount)
}
