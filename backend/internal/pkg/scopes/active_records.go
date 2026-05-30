package scopes

import (
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ActiveScoped is a marker interface for models that want automatic active filtering
// Models implementing this interface will have active status filter applied automatically
type ActiveScoped interface {
	// ActiveColumn returns the column name and filtering configuration
	// Return ("column_name", "active_value") for equality filter (column = active_value)
	// Return ("column_name", "!inactive_value") for exclusion filter (column != inactive_value)
	// The "!" prefix indicates exclusion filtering
	ActiveColumn() (columnName string, filterValue interface{})
}

// ActiveRecordsPlugin automatically adds active status filter for models implementing ActiveScoped
type ActiveRecordsPlugin struct{}

// Name returns the plugin name
func (p ActiveRecordsPlugin) Name() string {
	return "active_records_plugin"
}

// Initialize registers the plugin with GORM
func (p ActiveRecordsPlugin) Initialize(db *gorm.DB) error {
	// Register before the core query callback to ensure our filter is applied
	return db.Callback().Query().
		Before("gorm:query").
		Register("active_records_plugin:filter_active", p.filterActive)
}

// filterActive is the callback function that adds the active filter
func (p ActiveRecordsPlugin) filterActive(db *gorm.DB) {
	// Allow opt-out per query: db.Set("skip_active_filter", true)
	if v, ok := db.Get("skip_active_filter"); ok {
		if skip, _ := v.(bool); skip {
			return
		}
	}

	// Don't interfere with Unscoped queries
	if db.Statement.Unscoped {
		return
	}

	// Only apply to models that have a schema and model type
	if db.Statement.Schema == nil || db.Statement.Schema.ModelType == nil {
		return
	}

	// Create a new instance of the model to check if it implements ActiveScoped
	modelType := db.Statement.Schema.ModelType
	if modelType.Kind() == reflect.Pointer {
		modelType = modelType.Elem()
	}

	model := reflect.New(modelType).Interface()

	// Check if the model implements ActiveScoped
	if scoped, ok := model.(ActiveScoped); ok {
		columnName, filterValue := scoped.ActiveColumn()

		// Check if there's already a WHERE clause for this column
		// If there is, don't override it (allow explicit filtering)
		hasExplicitFilter := false
		if whereClause, exists := db.Statement.Clauses["WHERE"]; exists {
			if where, ok := whereClause.Expression.(clause.Where); ok {
				for _, expr := range where.Exprs {
					if eq, ok := expr.(clause.Eq); ok {
						if col, ok := eq.Column.(clause.Column); ok && col.Name == columnName {
							hasExplicitFilter = true
							break
						}
					}
					if neq, ok := expr.(clause.Neq); ok {
						if col, ok := neq.Column.(clause.Column); ok && col.Name == columnName {
							hasExplicitFilter = true
							break
						}
					}
					// Also check for other expressions that might contain our column
					if and, ok := expr.(clause.AndConditions); ok {
						for _, condition := range and.Exprs {
							if eq, ok := condition.(clause.Eq); ok {
								if col, ok := eq.Column.(clause.Column); ok && col.Name == columnName {
									hasExplicitFilter = true
									break
								}
							}
							if neq, ok := condition.(clause.Neq); ok {
								if col, ok := neq.Column.(clause.Column); ok && col.Name == columnName {
									hasExplicitFilter = true
									break
								}
							}
						}
					}
				}
			}
		}

		// Only add our filter if there's no explicit filter for this column
		if !hasExplicitFilter {
			// Check if this is exclusion filtering (starts with "!")
			if filterValueStr, ok := filterValue.(string); ok && len(filterValueStr) > 0 && filterValueStr[0] == '!' {
				// Exclusion filter: column != value
				excludeValue := filterValueStr[1:] // Remove the "!" prefix
				db.Where(clause.Neq{
					Column: clause.Column{Name: columnName},
					Value:  excludeValue,
				})
			} else {
				// Equality filter: column = value
				db.Where(clause.Eq{
					Column: clause.Column{Name: columnName},
					Value:  filterValue,
				})
			}
		}
	}
}

// SkipActiveFilter can be used to bypass the active filter for a specific query
// Usage: db.Set("skip_active_filter", true).Find(&records)
func SkipActiveFilter(db *gorm.DB) *gorm.DB {
	return db.Set("skip_active_filter", true)
}
