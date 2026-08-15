package domain

import (
	"context"

	"api-server/internal/pkg/clock"
)

// LoanStrategy defines the interface for different loan calculation strategies
type LoanStrategy interface {
	// CalculateNextPayment calculates the next payment amount and due date
	CalculateNextPayment(loan *Loan) (*ScheduleItem, error)

	// ValidateLoan validates the loan according to the strategy rules
	ValidateLoan(loan *Loan) error

	// GenerateSchedule generates the complete payment schedule for the loan
	GenerateSchedule(loan *Loan) ([]ScheduleItem, error)

	// ProcessPayment processes a payment according to the strategy
	ProcessPayment(loan *Loan, paymentAmount int64, scheduleID uint) error
}

// CustomScheduleStrategy handles loans with custom repayment schedules
type CustomScheduleStrategy struct {
	LoanRepaymentScheduleRepo LoanRepaymentScheduleRepository
}

func NewCustomScheduleStrategy(scheduleRepo LoanRepaymentScheduleRepository) *CustomScheduleStrategy {
	return &CustomScheduleStrategy{
		LoanRepaymentScheduleRepo: scheduleRepo,
	}
}

func (s *CustomScheduleStrategy) ValidateLoan(loan *Loan) error {
	return nil
}

func (s *CustomScheduleStrategy) CalculateNextPayment(loan *Loan) (*ScheduleItem, error) {
	// Get the next pending schedule item from the repayment schedule
	schedules, err := s.LoanRepaymentScheduleRepo.GetPendingSchedulesByLoan(context.Background(), loan.ID)
	if err != nil {
		return nil, err
	}

	if len(schedules) == 0 {
		return nil, NewNotFoundError("Không có lịch thanh toán nào đang chờ")
	}

	// Return the next scheduled payment
	nextSchedule := schedules[0]
	return &ScheduleItem{
		Period:  nextSchedule.Period,
		DueDate: nextSchedule.DueDate,
		Type:    "custom",
		Amount:  nextSchedule.Amount,
		Status:  string(nextSchedule.Status),
	}, nil
}

func (s *CustomScheduleStrategy) GenerateSchedule(loan *Loan) ([]ScheduleItem, error) {
	// Retrieve all schedule items from the repayment schedule table
	schedules, err := s.LoanRepaymentScheduleRepo.GetByLoanID(context.Background(), loan.ID)
	if err != nil {
		return nil, err
	}

	var scheduleItems []ScheduleItem
	for _, schedule := range schedules {
		scheduleItems = append(scheduleItems, ScheduleItem{
			Period:  schedule.Period,
			DueDate: schedule.DueDate,
			Type:    "custom",
			Amount:  schedule.Amount,
			Status:  string(schedule.Status),
		})
	}

	return scheduleItems, nil
}

func (s *CustomScheduleStrategy) ProcessPayment(loan *Loan, paymentAmount int64, scheduleID uint) error {
	// Update the specific schedule item as paid and update loan outstanding principal
	schedule, err := s.LoanRepaymentScheduleRepo.GetByID(context.Background(), scheduleID)
	if err != nil {
		return err
	}

	// Verify the schedule belongs to the loan
	if schedule.LoanID != loan.ID {
		return NewValidationError("Lịch thanh toán không thuộc về khoản vay này")
	}

	// Verify the payment amount matches the scheduled amount
	if paymentAmount != schedule.Amount {
		return NewValidationError("Số tiền thanh toán không khớp với số tiền trong lịch")
	}
	if err := schedule.ValidateComponents(); err != nil {
		return err
	}

	// Update the schedule as paid
	now := clock.Now()
	schedule.Status = ScheduleStatusPaid
	schedule.PaidAt = &now

	if err := s.LoanRepaymentScheduleRepo.Update(context.Background(), schedule); err != nil {
		return err
	}

	return loan.ApplyScheduledPayment(schedule.PrincipalAmount, schedule.InterestAmount)
}
