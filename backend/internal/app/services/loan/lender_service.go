package loan

import (
	"api-server/internal/constants"
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
)

type LenderService struct {
	logger       *slog.Logger
	LenderRepo   domain.LenderRepository
	CacheService *infrastructure.CacheService
	EventBus     domain.EventBus
}

func NewLenderService(
	lenderRepo domain.LenderRepository,
	cacheService *infrastructure.CacheService,
	eventBus domain.EventBus,
) *LenderService {
	return &LenderService{
		logger:       slog.Default(),
		LenderRepo:   lenderRepo,
		CacheService: cacheService,
		EventBus:     eventBus,
	}
}

// CreateLender creates a new lender
func (s *LenderService) CreateLender(ctx context.Context, lender *domain.Lender) (*domain.Lender, error) {
	// Validate lender
	if err := lender.Validate(); err != nil {
		return nil, err
	}

	// Create lender
	if err := s.LenderRepo.Create(ctx, lender); err != nil {
		return nil, fmt.Errorf("failed to create lender: %w", err)
	}

	// Publish event
	if err := s.EventBus.Publish(ctx, domain.NewLenderCreatedEvent(ctx, lender)); err != nil {
		s.logger.Warn("Failed to publish lender created event", "lenderID", lender.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLenderCache(ctx)

	s.logger.Info("Lender created successfully", "lenderID", lender.ID, "name", lender.Name)

	return lender, nil
}

// GetLender retrieves a lender by ID
func (s *LenderService) GetLender(ctx context.Context, id uint) (*domain.Lender, error) {
	lender, err := s.LenderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return lender, nil
}

// ListLenders retrieves a paginated list of lenders
func (s *LenderService) ListLenders(ctx context.Context, filters domain.LenderFilters) ([]*domain.Lender, int64, error) {
	lenders, total, err := s.LenderRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list lenders: %w", err)
	}

	return lenders, total, nil
}

// UpdateLender updates a lender
func (s *LenderService) UpdateLender(ctx context.Context, id uint, updates map[string]interface{}) (*domain.Lender, error) {
	// Get existing lender
	lender, err := s.LenderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	originalLender := *lender

	// Apply updates
	if name, ok := updates["name"]; ok && name != nil {
		lender.Name = name.(string)
	}
	if cccd, ok := updates["cccd"]; ok {
		if cccd == nil {
			lender.CCCD = nil
		} else {
			cccdStr := cccd.(string)
			lender.CCCD = &cccdStr
		}
	}
	if email, ok := updates["email"]; ok {
		if email == nil {
			lender.Email = nil
		} else {
			emailStr := email.(string)
			lender.Email = &emailStr
		}
	}
	if mobile, ok := updates["mobile"]; ok {
		if mobile == nil {
			lender.Mobile = nil
		} else {
			mobileStr := mobile.(string)
			lender.Mobile = &mobileStr
		}
	}
	if notes, ok := updates["notes"]; ok {
		if notes == nil {
			lender.Notes = nil
		} else {
			notesStr := notes.(string)
			lender.Notes = &notesStr
		}
	}
	if bankID, ok := updates["bank_id"]; ok {
		if bankID == nil {
			lender.BankID = nil
		} else {
			bankIDVal := bankID.(uint)
			lender.BankID = &bankIDVal
		}
	}
	if bankAccountNumber, ok := updates["bank_account_number"]; ok {
		if bankAccountNumber == nil {
			lender.BankAccountNumber = nil
		} else {
			bankAccountNumberStr := bankAccountNumber.(string)
			lender.BankAccountNumber = &bankAccountNumberStr
		}
	}
	if bankAccountName, ok := updates["bank_account_name"]; ok {
		if bankAccountName == nil {
			lender.BankAccountName = nil
		} else {
			bankAccountNameStr := bankAccountName.(string)
			lender.BankAccountName = &bankAccountNameStr
		}
	}

	// Validate lender
	if err := lender.Validate(); err != nil {
		return nil, err
	}

	// Update lender
	if err := s.LenderRepo.Update(ctx, lender); err != nil {
		return nil, fmt.Errorf("failed to update lender: %w", err)
	}

	// Publish event
	if err := s.EventBus.Publish(ctx, domain.NewLenderUpdatedEvent(ctx, lender, &originalLender)); err != nil {
		s.logger.Warn("Failed to publish lender updated event", "lenderID", lender.ID, "error", err)
	}

	// Invalidate cache
	s.invalidateLenderCache(ctx)

	s.logger.Info("Lender updated successfully", "lenderID", lender.ID)

	return lender, nil
}

// DeleteLender deletes a lender if no disbursed loans exist.
func (s *LenderService) DeleteLender(ctx context.Context, id uint) error {
	// Load existing lender (ensures NotFound surfaces and enables audit old values)
	existing, err := s.LenderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if lender has any disbursed loans (active or closed)
	hasDisbursed, err := s.LenderRepo.HasDisbursedLoans(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check disbursed loans: %w", err)
	}

	if hasDisbursed {
		return domain.NewValidationError(constants.MsgCannotDeleteLenderWithDisbursedLoansVN)
	}

	// Soft-delete any undisbursed (draft) loans for this lender to avoid orphans
	if err := s.LenderRepo.DeleteUndisbursedLoansByLender(ctx, id); err != nil {
		return fmt.Errorf("failed to remove undisbursed loans for lender: %w", err)
	}

	// Delete lender (soft delete per GORM DeletedAt)
	if err := s.LenderRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete lender: %w", err)
	}

	// Publish event
	if err := s.EventBus.Publish(ctx, domain.NewLenderDeletedEvent(ctx, existing)); err != nil {
		s.logger.Warn("Failed to publish lender deleted event", "lenderID", id, "error", err)
	}

	// Invalidate cache
	s.invalidateLenderCache(ctx)

	s.logger.Info("Lender deleted successfully", "lenderID", id)

	return nil
}

// invalidateLenderCache invalidates lender-related cache entries
func (s *LenderService) invalidateLenderCache(ctx context.Context) {
	patterns := []string{
		"lender:*",
		"lenders:*",
	}

	for _, pattern := range patterns {
		if err := s.CacheService.DeletePattern(ctx, pattern); err != nil {
			s.logger.Error("Failed to invalidate lender cache", "pattern", pattern, "error", err)
		}
	}
}
