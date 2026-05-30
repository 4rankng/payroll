package common

import (
	"context"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"errors"
	"log/slog"

	"gorm.io/gorm"
)

// RepoErrorHandler provides standardized error handling for repositories.
// It ensures consistent error messages, proper error wrapping, and centralized logging.
type RepoErrorHandler struct {
	logger *slog.Logger
}

// NewRepoErrorHandler creates a new RepoErrorHandler instance.
func NewRepoErrorHandler() *RepoErrorHandler {
	return &RepoErrorHandler{
		logger: observability.GetLogger(),
	}
}

// HandleGetError handles errors from Get/Find operations.
// Returns a proper domain.NotFoundError for GORM's ErrRecordNotFound.
// Wraps other errors with context about the operation.
func (reh *RepoErrorHandler) HandleGetError(err error, entityType string, entityID interface{}) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.NewNotFoundError(fmt.Sprintf("%s with id %v not found", entityType, entityID))
	}

	return reh.WrapDBError(err, fmt.Sprintf("get %s", entityType))
}

// HandleListError handles errors from List operations.
// Wraps errors with context about the list operation.
func (reh *RepoErrorHandler) HandleListError(err error, entityType string) error {
	if err == nil {
		return nil
	}

	return reh.WrapDBError(err, fmt.Sprintf("list %s", entityType))
}

// HandleCreateError handles errors from Create operations.
// Wraps errors with context about the create operation.
func (reh *RepoErrorHandler) HandleCreateError(err error, entityType string) error {
	if err == nil {
		return nil
	}

	return reh.WrapDBError(err, fmt.Sprintf("create %s", entityType))
}

// HandleUpdateError handles errors from Update operations.
// Returns a proper domain.NotFoundError for GORM's ErrRecordNotFound.
// Wraps other errors with context about the update operation.
func (reh *RepoErrorHandler) HandleUpdateError(err error, entityType string, entityID interface{}) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.NewNotFoundError(fmt.Sprintf("%s with id %v not found", entityType, entityID))
	}

	return reh.WrapDBError(err, fmt.Sprintf("update %s", entityType))
}

// HandleDeleteError handles errors from Delete operations.
// Returns a proper domain.NotFoundError for GORM's ErrRecordNotFound.
// Wraps other errors with context about the delete operation.
func (reh *RepoErrorHandler) HandleDeleteError(err error, entityType string, entityID interface{}) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.NewNotFoundError(fmt.Sprintf("%s with id %v not found", entityType, entityID))
	}

	return reh.WrapDBError(err, fmt.Sprintf("delete %s", entityType))
}

// WrapDBError wraps a database error with operation context.
// Provides consistent error wrapping and logging for all DB operations.
func (reh *RepoErrorHandler) WrapDBError(err error, operation string) error {
	if err == nil {
		return nil
	}

	wrapped := fmt.Errorf("database error during %s: %w", operation, err)

	// Log the error for debugging
	// Note: Using logger.Error to ensure visibility
	reh.logger.Error("database_operation_failed",
		"operation", operation,
		"error", err,
	)

	return wrapped
}

// LogOperation logs a repository operation for debugging and monitoring.
// This should be called at the end of operations to provide observability.
//
// Parameters:
//   - ctx: Context for the operation
//   - operation: Type of operation (e.g., "list", "get", "create", "update", "delete")
//   - entityType: Type of entity being operated on (e.g., "project", "timesheet")
//   - entityID: ID of the entity (if applicable)
//   - duration: Duration of the operation
//   - err: Any error that occurred (nil if successful)
func (reh *RepoErrorHandler) LogOperation(
	ctx context.Context,
	operation string,
	entityType string,
	entityID *uint,
	duration int64,
	err error,
) {
	if err != nil {
		if entityID != nil {
			reh.logger.Error("repository_operation_failed",
				"operation", operation,
				"entity_type", entityType,
				"entity_id", *entityID,
				"duration_ms", duration,
				"error", err,
			)
		} else {
			reh.logger.Error("repository_operation_failed",
				"operation", operation,
				"entity_type", entityType,
				"duration_ms", duration,
				"error", err,
			)
		}
	} else {
		if entityID != nil {
			reh.logger.Info("repository_operation_completed",
				"operation", operation,
				"entity_type", entityType,
				"entity_id", *entityID,
				"duration_ms", duration,
			)
		} else {
			reh.logger.Info("repository_operation_completed",
				"operation", operation,
				"entity_type", entityType,
				"duration_ms", duration,
			)
		}
	}
}
