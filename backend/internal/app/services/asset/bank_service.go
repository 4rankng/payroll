package asset

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

type BankService struct {
	BankRepo     domain.BankRepository
	CacheService *infrastructure.CacheService
	EventBus     domain.EventBus
	logger       *slog.Logger
}

func NewBankService(bankRepo domain.BankRepository, cacheService *infrastructure.CacheService, eventBus domain.EventBus) *BankService {
	return &BankService{
		BankRepo:     bankRepo,
		CacheService: cacheService,
		EventBus:     eventBus,
		logger:       observability.GetLogger(),
	}
}

func (s *BankService) CreateBank(ctx context.Context, bank *domain.Bank, createdBy uint) (*domain.Bank, error) {
	// Validate bank
	if err := bank.IsValid(); err != nil {
		return nil, err
	}

	if err := s.BankRepo.Create(ctx, bank); err != nil {
		return nil, fmt.Errorf("failed to create bank: %w", err)
	}

	// Publish domain event (cache will be invalidated asynchronously by event handler)
	event := domain.NewBankCreatedEvent(ctx, bank)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish BankCreatedEvent", "bank_id", bank.ID, "error", err)
	}

	return bank, nil
}

func (s *BankService) GetBank(ctx context.Context, id uint) (*domain.Bank, error) {
	return s.BankRepo.GetByID(ctx, id)
}

func (s *BankService) UpdateBank(ctx context.Context, bankID uint, updateData map[string]any, updatedBy uint) (*domain.Bank, error) {
	// Get existing bank
	existingBank, err := s.BankRepo.GetByID(ctx, bankID)
	if err != nil {
		return nil, err
	}

	originalBank := *existingBank

	// Apply updates
	if branchName, ok := updateData["branch_name"].(string); ok {
		existingBank.BranchName = branchName
	}
	if bin, ok := updateData["bin"].(string); ok {
		existingBank.Bin = bin
	}
	if bankCode, ok := updateData["bank_code"].(string); ok {
		existingBank.BankCode = bankCode
	}
	if swiftCode, ok := updateData["swift_code"].(string); ok {
		existingBank.SwiftCode = swiftCode
	}

	// Validate updated bank
	if err := existingBank.IsValid(); err != nil {
		return nil, err
	}

	if err := s.BankRepo.Update(ctx, existingBank); err != nil {
		return nil, fmt.Errorf("failed to update bank: %w", err)
	}

	// Publish domain event (cache will be invalidated asynchronously by event handler)
	event := domain.NewBankUpdatedEvent(ctx, existingBank, &originalBank)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish BankUpdatedEvent", "bank_id", existingBank.ID, "error", err)
	}

	return existingBank, nil
}

func (s *BankService) DeleteBank(ctx context.Context, id uint, deletedBy uint) error {
	// Get bank details before deletion for event
	deletedBank, err := s.BankRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get bank for deletion: %w", err)
	}

	if err := s.BankRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete bank: %w", err)
	}

	// Publish domain event (cache will be invalidated asynchronously by event handler)
	event := domain.NewBankDeletedEvent(ctx, deletedBank)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish BankDeletedEvent", "bank_id", id, "error", err)
	}

	return nil
}

func (s *BankService) ListBanks(ctx context.Context, filters domain.BankFilters) ([]*domain.Bank, int64, error) {
	// Generate cache key based on filters
	cacheKey := s.CacheService.GenerateBankCacheKey("list",
		fmt.Sprintf("limit:%d", filters.Limit),
		fmt.Sprintf("offset:%d", filters.Offset),
		fmt.Sprintf("search:%s", filters.Search),
		fmt.Sprintf("sortBy:%s", filters.SortBy),
		fmt.Sprintf("sortOrder:%s", filters.SortOrder))

	// Try to get from cache first
	type cachedResult struct {
		Banks []*domain.Bank `json:"banks"`
		Count int64          `json:"count"`
	}

	var cached cachedResult
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		return cached.Banks, cached.Count, nil
	}

	// Cache miss - get from database
	banks, err := s.BankRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list banks: %w", err)
	}

	count, err := s.BankRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count banks: %w", err)
	}

	// Store in cache
	result := cachedResult{Banks: banks, Count: count}
	if cacheErr := s.CacheService.Set(ctx, cacheKey, result, infrastructure.BankListCacheTTL); cacheErr != nil {
		// Log cache error but don't fail the request
		s.logger.Warn("Failed to cache bank list", "error", cacheErr)
	}

	return banks, count, nil
}

func (s *BankService) SearchBanksByBranchName(ctx context.Context, searchTerm string, limit int) ([]*domain.Bank, error) {
	// Validate search term length (minimum 3 characters)
	searchTerm = strings.TrimSpace(searchTerm)
	if len(searchTerm) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchTermMinLengthVN)
	}

	// Generate cache key for search
	cacheKey := s.CacheService.GenerateBankCacheKey("search",
		fmt.Sprintf("term:%s", strings.ToLower(searchTerm)),
		fmt.Sprintf("limit:%d", limit))

	// Try to get from cache first
	var cachedBanks []*domain.Bank
	if err := s.CacheService.Get(ctx, cacheKey, &cachedBanks); err == nil {
		return cachedBanks, nil
	}

	// Cache miss - get from database
	banks, err := s.BankRepo.SearchByBranchName(ctx, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search banks: %w", err)
	}

	// Store in cache
	if cacheErr := s.CacheService.Set(ctx, cacheKey, banks, infrastructure.BankSearchCacheTTL); cacheErr != nil {
		// Log cache error but don't fail the request
		s.logger.Warn("Failed to cache bank search", "error", cacheErr)
	}

	return banks, nil
}
