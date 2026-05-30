package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type LoanRepaymentScheduleRepository struct {
	*BaseRepository
}

func NewLoanRepaymentScheduleRepository(db *Database) domain.LoanRepaymentScheduleRepository {
	return &LoanRepaymentScheduleRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *LoanRepaymentScheduleRepository) Create(ctx context.Context, schedule *domain.LoanRepaymentSchedule) error {
	return r.DB.WithContext(ctx).Create(schedule).Error
}

func (r *LoanRepaymentScheduleRepository) GetByID(ctx context.Context, id uint) (*domain.LoanRepaymentSchedule, error) {
	var schedule domain.LoanRepaymentSchedule
	err := r.DB.WithContext(ctx).First(&schedule, id).Error

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
	err := r.DB.WithContext(ctx).
		Where("loan_id = ?", loanID).
		Order("period ASC").
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) GetPendingSchedulesByLoan(ctx context.Context, loanID uint) ([]*domain.LoanRepaymentSchedule, error) {
	var schedules []*domain.LoanRepaymentSchedule
	err := r.DB.WithContext(ctx).
		Where("loan_id = ? AND status = ?", loanID, domain.ScheduleStatusPending).
		Order("due_date ASC").
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) GetDueSchedules(ctx context.Context, dueDate time.Time) ([]*domain.LoanRepaymentSchedule, error) {
	var schedules []*domain.LoanRepaymentSchedule
	err := r.DB.WithContext(ctx).
		Where("status = ? AND transaction_id IS NULL AND due_date <= ?", domain.ScheduleStatusPending, dueDate).
		Order("due_date ASC, id ASC").
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) GetByTransactionID(ctx context.Context, transactionID uint) (*domain.LoanRepaymentSchedule, error) {
	var schedule domain.LoanRepaymentSchedule
	err := r.DB.WithContext(ctx).
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
	err := r.DB.WithContext(ctx).
		Where("transaction_id IN ?", transactionIDs).
		Find(&schedules).Error

	return schedules, err
}

func (r *LoanRepaymentScheduleRepository) Update(ctx context.Context, schedule *domain.LoanRepaymentSchedule) error {
	return r.DB.WithContext(ctx).Save(schedule).Error
}

func (r *LoanRepaymentScheduleRepository) Delete(ctx context.Context, id uint) error {
	return r.DB.WithContext(ctx).Delete(&domain.LoanRepaymentSchedule{}, id).Error
}
