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

func TestPayrateTemporalServiceRejectsCreateOnOrBeforeLatestConfigDate(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()

	const projectID uint = 49
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	storedDate := time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local)
	requestDate := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC) // same calendar date, UTC clock
	repository := &payrateTemporalRepositoryStub{}
	service := NewPayrateTemporalService(db, repository)

	seedPayrateTemporalPayrate(t, db, &domain.Payrate{
		ID: 3, ProjectID: projectID,
		FromDate: storedDate,
		Payrate:  domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	})

	err := service.CreateEffectiveDatedPayrate(ctx, &domain.Payrate{
		ProjectID: projectID,
		FromDate:  requestDate,
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	})

	require.Error(t, err)
	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domain.ErrCodePayrateNotLatest, domainErr.Code)
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

func TestPayrateTemporalServiceMovesStartDateEarlierInPlace(t *testing.T) {
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

	// Moving the latest config earlier is an in-place correction: no boundary
	// stitching, the predecessor row stays untouched, and the derived end date
	// simply follows the new successor from_date.
	target.FromDate = time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)
	target.Payrate = domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`)
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persistedPrevious domain.Payrate
	require.NoError(t, db.First(&persistedPrevious, previous.ID).Error)
	require.NotNil(t, persistedPrevious.ToDate)
	require.Equal(t, "2026-08-21", persistedPrevious.ToDate.Format("2006-01-02"))

	var persistedTarget domain.Payrate
	require.NoError(t, db.First(&persistedTarget, target.ID).Error)
	require.True(t, persistedTarget.FromDate.Equal(target.FromDate))
	require.Nil(t, persistedTarget.ToDate)
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

func TestPayrateTemporalServiceLaterRateChangeRequiresNewConfig(t *testing.T) {
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

	// Moving the start later past linked rows is rejected: a rate change from
	// a later date is a new config, not a date move.
	target.FromDate = time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local)
	target.Payrate = domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`)
	require.Error(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, target.ID).Error)
	require.True(t, persisted.FromDate.Equal(time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)))
	require.Nil(t, persisted.ToDate)

	// The later change goes through the create flow: a pure append leaves the
	// predecessor untouched, keeps paid rows on the config that priced them,
	// and re-links mutable rows to the new config.
	next := &domain.Payrate{
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 22, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	}
	require.NoError(t, service.CreateEffectiveDatedPayrate(ctx, next))
	require.Equal(t, uint(99), next.ID)

	var persistedAfterCreate domain.Payrate
	require.NoError(t, db.First(&persistedAfterCreate, target.ID).Error)
	require.Nil(t, persistedAfterCreate.ToDate, "predecessor must stay untouched: no to_date stitching")
	require.True(t, persistedAfterCreate.FromDate.Equal(time.Date(2026, 8, 15, 0, 0, 0, 0, time.Local)))

	assertTimesheetRateOnConfig(t, db, paid.ID, 12, 100000)
	assertTimesheetRateOnConfig(t, db, paidLater.ID, 12, 100000)
	assertTimesheetRateOnConfig(t, db, pending.ID, 99, 250000)
}

func TestPayrateTemporalServiceMovesStartDateLaterWithNoLinkedTimesheets(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 52
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	target := &domain.Payrate{
		ID:        16,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	}
	seedPayrateTemporalPayrate(t, db, target)

	// No linked timesheets: a start-date correction updates in place and
	// keeps the config open-ended.
	target.FromDate = time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, 16).Error)
	require.True(t, persisted.FromDate.Equal(time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)))
	require.Nil(t, persisted.ToDate)
	require.Equal(t, uint(16), target.ID, "no new row: the config is corrected in place")
}

func TestPayrateTemporalServiceRejectsUpdateOfReplacedConfig(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 51
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	endDate := time.Date(2026, 8, 5, 0, 0, 0, 0, time.Local)
	replaced := &domain.Payrate{
		ID:        15,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		ToDate:    &endDate,
	}
	successor := &domain.Payrate{
		ID:        17,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 6, 0, 0, 0, 0, time.Local),
	}
	seedPayrateTemporalPayrate(t, db, replaced)
	seedPayrateTemporalPayrate(t, db, successor)

	// A config a newer config has replaced is history — view-only regardless
	// of the stale stored to_date.
	target := *replaced
	target.FromDate = time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local)
	err := service.UpdateEffectiveDatedPayrate(ctx, &target)
	require.Error(t, err)
	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domain.ErrCodePayrateNotLatest, domainErr.Code)

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, replaced.ID).Error)
	require.True(t, persisted.FromDate.Equal(replaced.FromDate))
}

func TestPayrateTemporalServiceLatestConfigWithStaleEndDateStaysEditable(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 54
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	staleEnd := time.Date(2026, 8, 14, 0, 0, 0, 0, time.Local)
	lone := &domain.Payrate{
		ID:        18,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		ToDate:    &staleEnd, // leftover stitching bookkeeping; no successor exists
	}
	seedPayrateTemporalPayrate(t, db, lone)

	// The stale to_date must not brick the project's only (latest) config.
	lone.FromDate = time.Date(2026, 8, 5, 0, 0, 0, 0, time.Local)
	lone.Payrate = domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`)
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, lone))
}

func TestPayrateTemporalServiceDeleteHealsTimelineAndRepricesMutableLinks(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 55
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	predecessor := &domain.Payrate{
		ID:        19,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":100000}}}`),
	}
	target := &domain.Payrate{
		ID:        20,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	}
	seedPayrateTemporalPayrate(t, db, predecessor)
	seedPayrateTemporalPayrate(t, db, target)

	pending := newPayrateTemporalTimesheet(projectID, time.Date(2026, 8, 11, 0, 0, 0, 0, time.Local), domain.TimesheetStatusPendingApproval, domain.PaymentStatusPending)
	pending.PayrateID = target.ID
	seedPayrateTemporalTimesheet(t, db, pending)

	deleted, err := service.DeleteEffectiveDatedPayrate(ctx, target.ID)
	require.NoError(t, err)
	require.Equal(t, target.ID, deleted.ID)

	// Soft-deleted...
	var count int64
	require.NoError(t, db.Model(&domain.Payrate{}).Where("id = ?", target.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)

	// ...and the linked mutable row was re-priced under the covering config.
	assertTimesheetRateOnConfig(t, db, pending.ID, predecessor.ID, 100000)
}

func TestPayrateTemporalServiceDeleteBlockedByApprovedTimesheets(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 56
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	target := &domain.Payrate{
		ID:        21,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	}
	seedPayrateTemporalPayrate(t, db, target)

	paid := newPayrateTemporalTimesheet(projectID, time.Date(2026, 8, 11, 0, 0, 0, 0, time.Local), domain.TimesheetStatusApproved, domain.PaymentStatusPaid)
	paid.PayrateID = target.ID
	seedPayrateTemporalTimesheet(t, db, paid)

	_, err := service.DeleteEffectiveDatedPayrate(ctx, target.ID)
	require.Error(t, err)
	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domain.ErrCodePayrateImmutableLinked, domainErr.Code)

	var count int64
	require.NoError(t, db.Model(&domain.Payrate{}).Where("id = ?", target.ID).Count(&count).Error)
	require.Equal(t, int64(1), count, "immutable links hard-block the delete")
}

func TestPayrateTemporalServiceDropsSuppliedEndDateOnUpdate(t *testing.T) {
	db := newPayrateTemporalTestDB(t)
	ctx := context.Background()
	service := NewPayrateTemporalService(db, nil)

	const projectID uint = 53
	require.NoError(t, db.Exec("INSERT INTO projects (id, is_flexible) VALUES (?, ?)", projectID, false).Error)
	target := &domain.Payrate{
		ID:        16,
		ProjectID: projectID,
		FromDate:  time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local),
		Payrate:   domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-18:00":250000}}}`),
	}
	seedPayrateTemporalPayrate(t, db, target)

	// A caller (or legacy API client) supplying effective_to must not be able
	// to end the config: persisting it would make the row "ended" by the
	// update guard and brick it from further edits — same ruling as create.
	suppliedEnd := time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local)
	target.ToDate = &suppliedEnd
	require.NoError(t, service.UpdateEffectiveDatedPayrate(ctx, target))

	var persisted domain.Payrate
	require.NoError(t, db.First(&persisted, target.ID).Error)
	require.Nil(t, persisted.ToDate, "update must keep the config open-ended despite a supplied effective_to")
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
