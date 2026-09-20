package persistence

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/secret"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettingsRepository struct {
	*BaseRepository
	// cipher seals the values of secret.ProtectedKeys rows at rest. Nil means no
	// key is configured: values are read and written exactly as before.
	cipher *secret.Cipher
}

// NewSettingsRepository builds the settings repository. It returns the concrete
// type (which satisfies domain.SettingsRepository) so callers can reach
// EncryptProtectedValues for the startup backfill.
func NewSettingsRepository(db *Database) *SettingsRepository {
	return &SettingsRepository{
		BaseRepository: NewBaseRepository(db),
		cipher:         secret.Default(),
	}
}

// NewSettingsRepositoryWithCipher is the explicit-wiring constructor: the caller
// supplies the cipher (e.g. built from application config) instead of relying on
// the process-wide cipher resolved from SETTINGS_SECRET_KEY. A nil cipher keeps
// protected values in plaintext.
func NewSettingsRepositoryWithCipher(db *Database, cipher *secret.Cipher) *SettingsRepository {
	return &SettingsRepository{
		BaseRepository: NewBaseRepository(db),
		cipher:         cipher,
	}
}

func (r *SettingsRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}

func (r *SettingsRepository) Create(ctx context.Context, settings *domain.Settings) error {
	stored, err := r.sealedCopy(settings)
	if err != nil {
		return err
	}
	if err := r.getDB(ctx).Create(stored).Error; err != nil {
		return err
	}
	// GORM fills the generated primary key and timestamps on the copy.
	settings.ID = stored.ID
	settings.UpdatedAt = stored.UpdatedAt
	return nil
}

// sealedCopy returns a shallow copy of settings holding the value to persist:
// the sealed form for protected keys, the value itself otherwise. The caller's
// struct keeps the plaintext — the API layer must never receive ciphertext, and
// must never write ciphertext back as if it were the credential.
func (r *SettingsRepository) sealedCopy(settings *domain.Settings) (*domain.Settings, error) {
	out := *settings
	if out.Value == nil || !secret.IsProtectedKey(out.Key) {
		return &out, nil
	}
	sealed, err := r.cipher.Encrypt(*out.Value)
	if err != nil {
		return nil, fmt.Errorf("seal setting %q: %w", out.Key, err)
	}
	out.Value = &sealed
	return &out, nil
}

// opened returns settings with its protected value decrypted in place. Legacy
// plaintext values pass through unchanged, so rows written before the key
// existed keep working.
func (r *SettingsRepository) opened(settings *domain.Settings) (*domain.Settings, error) {
	if settings == nil || settings.Value == nil || !secret.IsProtectedKey(settings.Key) {
		return settings, nil
	}
	plaintext, err := r.cipher.Decrypt(*settings.Value)
	if err != nil {
		return nil, fmt.Errorf("open setting %q: %w", settings.Key, err)
	}
	settings.Value = &plaintext
	return settings, nil
}

func (r *SettingsRepository) openedAll(settings []*domain.Settings) ([]*domain.Settings, error) {
	for _, row := range settings {
		if _, err := r.opened(row); err != nil {
			return nil, err
		}
	}
	return settings, nil
}

func (r *SettingsRepository) GetByID(ctx context.Context, id uint) (*domain.Settings, error) {
	return r.getByID(ctx, id, false)
}

func (r *SettingsRepository) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Settings, error) {
	return r.getByID(ctx, id, true)
}

func (r *SettingsRepository) getByID(ctx context.Context, id uint, forUpdate bool) (*domain.Settings, error) {
	var settings domain.Settings
	query := r.getDB(ctx)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&settings, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("setting not found")
		}
		return nil, err
	}

	return r.opened(&settings)
}

func (r *SettingsRepository) GetByKey(ctx context.Context, key string) (*domain.Settings, error) {
	return r.getByKey(ctx, key, false)
}

func (r *SettingsRepository) GetByKeyForUpdate(ctx context.Context, key string) (*domain.Settings, error) {
	return r.getByKey(ctx, key, true)
}

func (r *SettingsRepository) getByKey(ctx context.Context, key string, forUpdate bool) (*domain.Settings, error) {
	var settings domain.Settings
	query := r.getDB(ctx)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.Where("`key` = ?", key).First(&settings).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("setting not found")
		}
		return nil, err
	}

	return r.opened(&settings)
}

func (r *SettingsRepository) Update(ctx context.Context, settings *domain.Settings) error {
	stored, err := r.sealedCopy(settings)
	if err != nil {
		return err
	}
	if err := r.getDB(ctx).Save(stored).Error; err != nil {
		return err
	}
	settings.UpdatedAt = stored.UpdatedAt
	return nil
}

// CompareAndSwapValue updates a setting only when its JSON value still equals
// the value read by the caller. Zalo credentials use this to prevent a stale
// status/error writer from restoring a refresh token that another request has
// already consumed and replaced.
func (r *SettingsRepository) CompareAndSwapValue(
	ctx context.Context,
	key string,
	currentValue string,
	nextValue string,
	valueType domain.SettingsValueType,
) (bool, error) {
	if r.cipher.Configured() && secret.IsProtectedKey(key) {
		return r.compareAndSwapSealed(ctx, key, currentValue, nextValue, valueType)
	}
	return r.compareAndSwapLiteral(ctx, key, currentValue, nextValue, valueType)
}

// compareAndSwapLiteral is the compare-and-swap for values stored as-is.
func (r *SettingsRepository) compareAndSwapLiteral(
	ctx context.Context,
	key string,
	currentValue string,
	nextValue string,
	valueType domain.SettingsValueType,
) (bool, error) {
	db := r.getDB(ctx)
	query := db.Model(&domain.Settings{}).Where("`key` = ?", key)
	if currentValue == "" {
		query = query.Where("(`value` = ? OR `value` IS NULL)", currentValue)
	} else if db.Dialector.Name() == "mysql" { //nolint:staticcheck // QF1008: explicit selector documents the driver check
		// Tokens are case-sensitive. MySQL TEXT equality otherwise inherits the
		// database collation, which is commonly case-insensitive and can let a
		// stale value differing only by case pass the CAS predicate.
		query = query.Where("BINARY `value` = BINARY ?", currentValue)
	} else {
		query = query.Where("`value` = ?", currentValue)
	}
	result := query.
		Updates(map[string]any{
			"value":      nextValue,
			"value_type": valueType,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// compareAndSwapSealed is the compare-and-swap for a sealed key. GCM ciphertext
// is non-deterministic — sealing the same plaintext twice yields different
// bytes — so a freshly sealed current value can never be matched in SQL against
// the stored one. The stored value is therefore read, opened, and compared in
// Go, and the write is guarded by the exact stored bytes that were read. That
// guard keeps the swap atomic against a concurrent writer.
func (r *SettingsRepository) compareAndSwapSealed(
	ctx context.Context,
	key string,
	currentValue string,
	nextValue string,
	valueType domain.SettingsValueType,
) (bool, error) {
	db := r.getDB(ctx)

	var stored domain.Settings
	if err := db.Where("`key` = ?", key).First(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if stored.Value == nil {
		// Mirrors the literal path: a NULL value only matches an empty current.
		if currentValue != "" {
			return false, nil
		}
		sealed, err := r.cipher.Encrypt(nextValue)
		if err != nil {
			return false, fmt.Errorf("seal setting %q: %w", key, err)
		}
		result := db.Model(&domain.Settings{}).
			Where("`key` = ?", key).
			Where("`value` IS NULL").
			Updates(map[string]any{"value": sealed, "value_type": valueType})
		if result.Error != nil {
			return false, result.Error
		}
		return result.RowsAffected == 1, nil
	}

	plaintext, err := r.cipher.Decrypt(*stored.Value)
	if err != nil {
		return false, fmt.Errorf("open setting %q: %w", key, err)
	}
	if plaintext != currentValue {
		return false, nil
	}

	sealed, err := r.cipher.Encrypt(nextValue)
	if err != nil {
		return false, fmt.Errorf("seal setting %q: %w", key, err)
	}
	return r.writeWhenUnchanged(ctx, key, *stored.Value, sealed, valueType)
}

// writeWhenUnchanged writes nextValue only while the stored bytes still equal
// expectedRaw — the exact bytes read a moment earlier. Comparing the stored
// representation (rather than a re-encrypted plaintext) is what makes the
// compare half of a compare-and-swap work on non-deterministic ciphertext.
func (r *SettingsRepository) writeWhenUnchanged(
	ctx context.Context,
	key string,
	expectedRaw string,
	nextValue string,
	valueType domain.SettingsValueType,
) (bool, error) {
	db := r.getDB(ctx)
	query := db.Model(&domain.Settings{}).Where("`key` = ?", key)
	if db.Dialector.Name() == "mysql" { //nolint:staticcheck // QF1008: explicit selector documents the driver check
		query = query.Where("BINARY `value` = BINARY ?", expectedRaw)
	} else {
		query = query.Where("`value` = ?", expectedRaw)
	}
	result := query.Updates(map[string]any{
		"value":      nextValue,
		"value_type": valueType,
	})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// EncryptProtectedValues seals every protected setting still stored in
// plaintext and returns the keys it converted, in order. It is the startup
// backfill: idempotent (an already sealed row is skipped), safe to call with no
// key configured (no-op), and safe to run concurrently with live traffic — the
// write is guarded by the bytes that were read. Values are never returned or
// logged, only key names.
func (r *SettingsRepository) EncryptProtectedValues(ctx context.Context) ([]string, error) {
	if !r.cipher.Configured() {
		return nil, nil
	}

	db := r.getDB(ctx)
	var converted []string
	for _, key := range secret.ProtectedKeys {
		// Read the stored form directly: the repository read path decrypts, so it
		// cannot tell a sealed row from a legacy plaintext one.
		var stored domain.Settings
		if err := db.Where("`key` = ?", key).First(&stored).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return converted, err
		}
		if stored.Value == nil || *stored.Value == "" || secret.IsEncrypted(*stored.Value) {
			continue
		}

		sealed, err := r.cipher.Encrypt(*stored.Value)
		if err != nil {
			return converted, fmt.Errorf("seal setting %q: %w", key, err)
		}
		swapped, err := r.writeWhenUnchanged(ctx, key, *stored.Value, sealed, stored.ValueType)
		if err != nil {
			return converted, fmt.Errorf("seal setting %q: %w", key, err)
		}
		if !swapped {
			continue
		}
		converted = append(converted, key)
		observability.GetLogger().Info("settings: sealed protected value at rest", "key", key)
	}
	return converted, nil
}

func (r *SettingsRepository) Delete(ctx context.Context, id uint) error {
	// Use GORM soft delete
	return r.getDB(ctx).Delete(&domain.Settings{}, id).Error
}

func (r *SettingsRepository) List(ctx context.Context, filters domain.SettingsFilters) ([]*domain.Settings, error) {
	var settings []*domain.Settings
	query := r.getDB(ctx)

	// Apply filters
	query = r.applyFilters(query, filters)

	// Apply sorting
	sortBy := "updated_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "updated_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if err := query.Find(&settings).Error; err != nil {
		return nil, err
	}
	return r.openedAll(settings)
}

func (r *SettingsRepository) Count(ctx context.Context, filters domain.SettingsFilters) (int64, error) {
	var count int64
	query := r.getDB(ctx).Model(&domain.Settings{})

	// Apply filters
	query = r.applyFilters(query, filters)

	err := query.Count(&count).Error
	return count, err
}

func (r *SettingsRepository) GetActiveSettings(ctx context.Context) ([]*domain.Settings, error) {
	var settings []*domain.Settings
	err := r.getDB(ctx).
		Order("`key` ASC").
		Find(&settings).Error
	if err != nil {
		return nil, err
	}
	return r.openedAll(settings)
}

// applyFilters applies the common filters to the query
func (r *SettingsRepository) applyFilters(query *gorm.DB, filters domain.SettingsFilters) *gorm.DB {
	// Filter by value type
	if len(filters.ValueType) > 0 {
		query = query.Where("value_type IN ?", filters.ValueType)
	}

	// Search functionality
	if filters.Search != "" {
		searchTerm := "%" + filters.Search + "%"
		query = query.Where("LOWER(`key`) LIKE LOWER(?)", searchTerm)
	}

	return query
}
