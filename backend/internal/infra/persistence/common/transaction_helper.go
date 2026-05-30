package common

import (
	"fmt"

	"gorm.io/gorm"
)

// ExtractDB safely extracts a *gorm.DB from an interface{}.
// Returns an error if tx is nil or not a *gorm.DB.
func ExtractDB(tx interface{}) (*gorm.DB, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction is nil")
	}
	db, ok := tx.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("invalid transaction type: expected *gorm.DB, got %T", tx)
	}
	return db, nil
}
