package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoanRepaymentScheduleRepository struct {
	*BaseRepository
}

func NewLoanRepaymentScheduleRepository(db *Database) domain.LoanRepaymentScheduleRepository {
	return &LoanRepaymentScheduleRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// db returns the appropriate DB instance for the context — the caller's
// open transaction when one exists (row locks, inserts, and updates must
// share it; a second pooled connection would self-deadlock on locks this
// transaction holds and break atomicity).
func (r *LoanRepaymentScheduleRepository) db(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

func (r *LoanRepaymentScheduleRepository) Create(ctx context.Context, schedule *domain.LoanRepaymentSchedule) error {
	return r.db(ctx).Create(schedule).Error
}

func (r *LoanRepaymentScheduleRepository) GetByID(ctx context.Context, id uint) (*domain.LoanRepaymentSchedule, error) {
	var schedule domain.LoanRepaymentSchedule
	err := r.db(ctx).First(&schedule, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("loan repayment schedule not found")
		}
		return nil, err
	}

	return &schedule, nil
}

func (r *LoanRepaymentScheduleRepository) GetByIDForUpdate(ctx context.Context, id uint) (*domain.LoanRepaymentSchedule, error) {
	var schedule domain.LoanRepaymentSchedule
	err := r.dbForContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&schedule, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("loan repayment schedule not found")
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *LoanRepaymentScheduleRepository) GetByLoanID(ctx context.Context, loanID uint) ([]*domain.LoanRepaymentSchedule, error) {
	var schedules []*domain.LoanRepaymentSchedule
	err := r.db(ctx).
		Where("loan_id = ?", loanID).
		Order("period ASC").
		Find(&schedules).Error

	return schedules, err
}

// GetByLoanIDs batch-fetches schedules for multiple loans in one query, grouped
// by loan id. Mirrors GetByLoanID's ordering (period ASC) within each loan.
// Loans with no schedules are absent from the returned map. Eliminates the
// loan-list N+1 flagged by the ck:debug 2026-07-04 audit.
func (r *LoanRepaymentScheduleRepository) GetByLoanIDs(ctx context.Context, loanIDs []uint) (map[uint][]*domain.LoanRepaymentSchedule, error) {
	if len(loanIDs) == 0 {
		return make(map[uint][]*domain.LoanRepaymentSchedule), nil
	}
	var schedules []*domain.LoanRepaymentSchedule
	if err := r.db(ctx).
		Where("loan_id IN ?", loanIDs).
		Order("loan_id, period ASC").
		Find(&schedules).Error; err != nil {
		return nil, err
	}
	out := make(map[uint][]*domain.LoanRepaymentSchedule, len(loanIDs))
	for _, s := range schedules {
		out[s.LoanID] = append(out[s.LoanID], s)
	}
	return out, nil
}

func (r *LoanRepaymentScheduleRepository) GetPendingSchedulesByLoan(ctx context.Context, loanID uint) ([]*domain.LoanRepaymentSchedule, error) {
	var schedules []*domain.LoanRepaymentSchedule
	err := r.db(ctx).
		Where("loan_id = ? AND status = ?", loanID, domain.ScheduleStatusPending).
		Order("due_date ASC").
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) GetDueSchedules(ctx context.Context, dueDate time.Time) ([]*domain.LoanRepaymentSchedule, error) {
	var schedules []*domain.LoanRepaymentSchedule
	err := r.db(ctx).
		Where("status = ? AND transaction_id IS NULL AND due_date <= ?", domain.ScheduleStatusPending, dueDate).
		Order("due_date ASC, id ASC").
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) ListPendingForReminder(ctx context.Context, start, end time.Time) ([]*domain.LoanRepaymentReminder, error) {
	var reminders []*domain.LoanRepaymentReminder
	err := r.db(ctx).
		Table("loan_repayment_schedules AS schedules").
		Select(`
			schedules.id AS schedule_id,
			loans.loan_code AS loan_code,
			lenders.name AS lender_name,
			schedules.period AS period,
			schedules.due_date AS due_date,
			schedules.amount AS amount`).
		Joins("JOIN loans ON loans.id = schedules.loan_id AND loans.deleted_at IS NULL").
		Joins("JOIN lenders ON lenders.id = loans.lender_id AND lenders.deleted_at IS NULL").
		Where("schedules.status = ?", domain.ScheduleStatusPending).
		Where("schedules.due_date >= ? AND schedules.due_date < ?", start, end).
		Order("loans.loan_code ASC, schedules.period ASC, schedules.id ASC").
		Scan(&reminders).Error

	return reminders, err
}

func (r *LoanRepaymentScheduleRepository) GetByTransactionID(ctx context.Context, transactionID uint) (*domain.LoanRepaymentSchedule, error) {
	var schedule domain.LoanRepaymentSchedule
	err := r.db(ctx).
		Where("transaction_id = ?", transactionID).
		First(&schedule).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("loan repayment schedule not found")
		}
		return nil, err
	}

	return &schedule, nil
}

func (r *LoanRepaymentScheduleRepository) ListByTransactionIDs(ctx context.Context, transactionIDs []uint) ([]*domain.LoanRepaymentSchedule, error) {
	if len(transactionIDs) == 0 {
		return []*domain.LoanRepaymentSchedule{}, nil
	}

	var schedules []*domain.LoanRepaymentSchedule
	err := r.db(ctx).
		Where("transaction_id IN ?", transactionIDs).
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) Update(ctx context.Context, schedule *domain.LoanRepaymentSchedule) error {
	return r.db(ctx).Save(schedule).Error
}

func (r *LoanRepaymentScheduleRepository) Delete(ctx context.Context, id uint) error {
	return r.db(ctx).Delete(&domain.LoanRepaymentSchedule{}, id).Error
}
