package advance_payment

import (
	"context"

	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/notification"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// Config holds the dependencies for the advance payment service.
//
// FeeResolver is the new authoritative source for advance-payment fees,
// reading per-date schedules from the settings JSON via FeeScheduleService.
// GetAdvanceCashFeePercentage / GetAdvancePaymentFeeMin remain for the
// employee-facing "headline rate" UI label that has no transaction amount in
// hand — they now read the active schedule's first-tier percentage instead
// of the retired legacy settings keys.
type Config struct {
	AdvancePaymentRepo          domain.AdvancePaymentRepository
	AdvancePaymentRequestRepo   domain.AdvancePaymentRequestRepository
	TransactionCodeRepo         domain.TransactionCodeRepository
	TransactionRepo             domain.TransactionRepository
	EmployeeRepo                domain.EmployeeRepository
	ProjectEmployeeRepo         domain.ProjectEmployeeRepository
	ProjectRepo                 domain.ProjectRepository
	BankRepo                    domain.BankRepository
	BulkTransferFileRepo        domain.BulkTransferFileRepository
	AssetRepo                   domain.AssetRepository
	CacheService                domain.CacheServiceUseCase
	EmployeeUserService         *employee.EmployeeUserService
	EmployeeService             *employee.EmployeeService
	LedgerRepo                  domain.LedgerEntryRepository
	EmployeeNotifier            notification.EmployeeNotifier
	FeeResolver                 FeeResolver
	Clock                       clock.Clock
	GetAdvancePaymentPercentage func(ctx context.Context) float64
	GetAdvancePaymentFeeMin     func(ctx context.Context) uint64
	GetAdvanceCashFeePercentage func(ctx context.Context) float64
	GetPartnerCompany           func(ctx context.Context) string
	// GetSelfCheckInAdvancePercentage returns the authoritative self-check-in
	// advance percentage (e.g. 70) used to derive the advanceable cap from
	// credited salary. Exposed to the employee app so per-shift rows can show
	// the advanceable amount alongside the full wage.
	GetSelfCheckInAdvancePercentage func(ctx context.Context) uint64
	// GetTransferLimits returns the active disbursement provider's per-transfer
	// amount bounds. Returns zero-value limits when no provider is configured.
	// Used by CreateRequest to fail-fast when the net amount (after fee) would
	// be rejected by the provider.
	GetTransferLimits func(ctx context.Context) ProviderTransferLimits
}

// ProviderTransferLimits describes the disbursement provider's per-transfer bounds.
type ProviderTransferLimits struct {
	MinAmount int64
	MaxAmount int64
}
