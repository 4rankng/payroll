package workers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"api-server/internal/app/services/config"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type paidAmountTimesheetRepo struct {
	domain.TimesheetRepository
	rows    []*domain.Timesheet
	updates []domain.PaymentStatusUpdate
}

func (r *paidAmountTimesheetRepo) GetByIDs(context.Context, []uint) ([]*domain.Timesheet, error) {
	return r.rows, nil
}
func (r *paidAmountTimesheetRepo) BulkUpdatePaymentStatus(_ context.Context, updates []domain.PaymentStatusUpdate) error {
	r.updates = append(r.updates, updates...)
	return nil
}
func (r *paidAmountTimesheetRepo) BulkUpdateRevenueReceivable(context.Context, map[uint]int64) error {
	return nil
}

type paidAmountCodeRepo struct {
	domain.TransactionCodeRepository
	code *domain.TransactionCode
}

func (r paidAmountCodeRepo) GetByCode(context.Context, string) (*domain.TransactionCode, error) {
	return r.code, nil
}

type paidAmountAssignments struct {
	domain.ProjectEmployeeRepository
}

func (paidAmountAssignments) GetActiveAssignmentsByProjectsAndEmployees(context.Context, []uint, []uint) ([]*domain.ProjectEmployee, error) {
	return []*domain.ProjectEmployee{{ProjectID: 1, EmployeeID: 1, PaymentSchedule: "monthly"}}, nil
}

type changedPercentageSettings struct{ config.SettingReader }

func (changedPercentageSettings) GetSettingByKey(_ context.Context, key string) (*domain.Settings, error) {
	value := "0.5"
	return &domain.Settings{Key: key, Value: &value, ValueType: domain.ValueTypeNumber}, nil
}

func TestUpdateForTransferKeepsSavedAmountAcrossConfigurationChanges(t *testing.T) {
	for _, test := range []struct {
		name         string
		stored       int64
		snapshot     *int64
		alreadyPaid  bool
		paidMismatch bool
		want         int64
	}{
		{name: "new export at 70 percent then setting changes to 50", stored: 168000, snapshot: int64PointerForPaidTest(168000), want: 84000},
		{name: "unversioned bank gross amount retains legacy fallback", stored: 240000, want: 60000},
		{name: "legacy amount absent retains existing fallback", stored: 0, want: 60000},
		{name: "partial retry preserves matching paid share", stored: 168000, snapshot: int64PointerForPaidTest(168000), alreadyPaid: true, want: 84000},
		{name: "different already-paid amount requires reconciliation", stored: 168000, snapshot: int64PointerForPaidTest(168000), alreadyPaid: true, paidMismatch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := miniredis.RunT(t)
			cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
			t.Cleanup(func() { _ = cache.Close() })
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			rows := []*domain.Timesheet{{ID: 1, ProjectID: 1, EmployeeID: 1, Amount: 120000}, {ID: 2, ProjectID: 1, EmployeeID: 1, Amount: 120000}}
			if test.alreadyPaid {
				rows[0].PaymentStatus = domain.PaymentStatusPaid
				rows[0].PaidAmount = 84000
				if test.paidMismatch {
					rows[0].PaidAmount = 70000
				}
			}
			repo := &paidAmountTimesheetRepo{rows: rows}
			data, err := json.Marshal(domain.TransactionCodeData{MonthlyPay: &domain.CyclePayData{TimesheetIDs: []uint{1, 2}, Amount: test.stored, TransferAmountSnapshot: test.snapshot}})
			require.NoError(t, err)
			worker := NewBulkTransferPaymentWorker(nil, repo, paidAmountAssignments{}, paidAmountCodeRepo{code: &domain.TransactionCode{Data: data}}, config.NewSettingsConfigService(changedPercentageSettings{}), infrastructure.NewIdempotencyService(cache, logger), nil, nil, nil)
			if test.paidMismatch {
				err := worker.UpdateForTransfer(context.Background(), "VFICtest", true, "bank-reference")
				require.ErrorContains(t, err, "reconciliation required")
				require.Empty(t, repo.updates, "do not write a partial allocation")
				return
			}
			require.NoError(t, worker.UpdateForTransfer(context.Background(), "VFICtest", true, "bank-reference"))
			wantCount := 2
			if test.alreadyPaid {
				wantCount = 1
			}
			require.Len(t, repo.updates, wantCount)
			for _, update := range repo.updates {
				require.NotNil(t, update.PaidAmount)
				require.Equal(t, test.want, *update.PaidAmount)
				require.Equal(t, domain.PaymentStatusPaid, update.PaymentStatus)
				if test.alreadyPaid {
					require.Equal(t, uint(2), update.TimesheetID)
				}
			}
			require.NoError(t, worker.UpdateForTransfer(context.Background(), "VFICtest", true, "bank-reference"))
			require.Len(t, repo.updates, wantCount, "duplicate callback must not write a second payment")
		})
	}
}

func int64PointerForPaidTest(value int64) *int64 { return &value }
