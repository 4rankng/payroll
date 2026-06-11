package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// SettingsStore provides generic JSON-in-Settings-row CRUD with FOR UPDATE
// locking. It eliminates the duplicated executeWrite/loadEntries/decodeEntries
// boilerplate found in fee schedule services.
//
// Type parameter T is the domain entry type stored as a JSON array in a single
// Settings row (e.g. FeeScheduleEntry, DisbursementFeeScheduleEntry).
type SettingsStore[T any] struct {
	DB           *gorm.DB
	SettingsKey  string
	SortFn       func([]T)
	NotFoundMsg  string // error message when settings row is missing
	EntityName   string // human-readable name for error messages ("fee schedule entry")
	DateLayout   string // time format for effective-date parsing
}

// ExecuteWrite runs a transaction that locks the settings row, reads the
// current entry list, lets mutate produce the new list, sorts and writes it
// back. mutate may return an error to abort.
func (s *SettingsStore[T]) ExecuteWrite(ctx context.Context, mutate func(tx *gorm.DB, entries []T) ([]T, error)) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var setting domain.Settings
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("`key` = ?", s.SettingsKey).
			First(&setting).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.NewNotFoundError(s.NotFoundMsg)
			}
			return fmt.Errorf("lock settings row (%s): %w", s.SettingsKey, err)
		}

		entries, err := DecodeSettingsEntries[T](setting.Value, s.EntityName)
		if err != nil {
			return err
		}

		next, err := mutate(tx, entries)
		if err != nil {
			return err
		}

		s.SortFn(next)
		encoded, err := json.Marshal(next)
		if err != nil {
			return fmt.Errorf("encode %s entries: %w", s.EntityName, err)
		}
		val := string(encoded)
		setting.Value = &val
		if err := tx.Save(&setting).Error; err != nil {
			return fmt.Errorf("save %s settings row: %w", s.EntityName, err)
		}
		return nil
	})
}

// LoadEntries reads and decodes the entry list outside any transaction.
// Callers that mutate must use ExecuteWrite, which acquires FOR UPDATE.
func (s *SettingsStore[T]) LoadEntries(ctx context.Context) ([]T, error) {
	var setting domain.Settings
	err := s.DB.WithContext(ctx).
		Where("`key` = ?", s.SettingsKey).
		First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError(s.NotFoundMsg)
		}
		return nil, fmt.Errorf("load settings row (%s): %w", s.SettingsKey, err)
	}
	return DecodeSettingsEntries[T](setting.Value, s.EntityName)
}

// DecodeSettingsEntries unmarshals a JSON array from a Settings.Value pointer.
func DecodeSettingsEntries[T any](value *string, entityName string) ([]T, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	var entries []T
	if err := json.Unmarshal([]byte(*value), &entries); err != nil {
		return nil, fmt.Errorf("decode %s entries: %w", entityName, err)
	}
	return entries, nil
}

// IndexOfEntryID returns the index of the entry with the given ID, or -1.
// idFn extracts the ID field from a domain entry.
func IndexOfEntryID[T any](entries []T, id string, idFn func(T) string) int {
	for i := range entries {
		if idFn(entries[i]) == id {
			return i
		}
	}
	return -1
}

// ValidateEffectiveDateNotPast rejects effective dates before today.
// Today is allowed — an admin announcing today's rate change should not be blocked.
func ValidateEffectiveDateNotPast(dateLayout, effectiveDate string) error {
	d, err := time.Parse(dateLayout, effectiveDate)
	if err != nil {
		return domain.NewValidationError(constants.MsgInvalidDateFormatVN)
	}
	today := clock.Now().Format(dateLayout)
	if d.Format(dateLayout) < today {
		return domain.NewValidationError("effective_date cannot be in the past")
	}
	return nil
}

// IsFutureDate reports whether the effective date is strictly after today.
func IsFutureDate(dateLayout, effectiveDate string) bool {
	d, err := time.Parse(dateLayout, effectiveDate)
	if err != nil {
		return false
	}
	today := clock.Now().Format(dateLayout)
	return d.Format(dateLayout) > today
}

// NonZeroUserID returns a *uint for non-zero IDs, nil otherwise.
func NonZeroUserID(id uint) *uint {
	if id == 0 {
		return nil
	}
	return &id
}
