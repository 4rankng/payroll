package db

import (
	"context"

	"gorm.io/gorm"
)

// BatchLoader provides utilities for batch loading related entities to prevent N+1 queries.
// This pattern is commonly used when loading relationships for a collection of entities.
type BatchLoader struct {
	ctx context.Context
	db  *gorm.DB
}

// NewBatchLoader creates a new BatchLoader instance
func NewBatchLoader(ctx context.Context, db *gorm.DB) *BatchLoader {
	return &BatchLoader{
		ctx: ctx,
		db:  db,
	}
}

// ExtractIDs extracts unique IDs from a collection based on a key extractor function.
// This is typically used to collect foreign keys before batch loading.
//
// Example:
//
//	employeeIDs := ExtractIDs(timesheets, func(ts *Timesheet) uint { return ts.EmployeeID })
func ExtractIDs[T any](items []T, keyExtractor func(T) uint) []uint {
	if len(items) == 0 {
		return nil
	}

	// Use map to ensure uniqueness
	idMap := make(map[uint]bool)
	for _, item := range items {
		id := keyExtractor(item)
		if id > 0 { // Skip zero IDs
			idMap[id] = true
		}
	}

	// Convert map keys to slice
	ids := make([]uint, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}

	return ids
}

// LoadToMap loads entities by IDs and returns them as a map keyed by ID.
// This is useful for quick lookups when assigning relationships.
//
// Example:
//
//	var employees []*Employee
//	employeeMap := LoadToMap(ctx, db, employeeIDs, &employees, "id")
func LoadToMap[T any](ctx context.Context, db *gorm.DB, ids []uint, dest *[]T, idField string) map[uint]*T {
	if len(ids) == 0 {
		return make(map[uint]*T)
	}

	// Load entities
	db.WithContext(ctx).Where(idField+" IN ?", ids).Find(dest)

	// Convert to map
	entityMap := make(map[uint]*T)
	for i := range *dest {
		item := &(*dest)[i]
		// Use reflection to get ID field value
		// For simplicity, we'll use a callback approach
		entityMap[getIDFromEntity(item)] = item
	}

	return entityMap
}

// getIDFromEntity is a helper to extract ID from an entity
// This is a simplified version - in production you might want to use reflection
// or require entities to implement an interface like type Identifiable interface { GetID() uint }
func getIDFromEntity[T any](entity *T) uint {
	// This is a placeholder - actual implementation would use reflection or interface
	// For now, we'll return 0 and let the caller handle the mapping
	return 0
}

// BatchLoadProjects is a concrete batch loader for projects
// This demonstrates the pattern that can be used for other entities
func (bl *BatchLoader) BatchLoadProjects(projectIDs []uint, preloads []string) map[uint]interface{} {
	if len(projectIDs) == 0 {
		return make(map[uint]interface{})
	}

	query := bl.db.WithContext(bl.ctx)
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	var projects []interface{} // Replace with actual Project type
	query.Where("id IN ?", projectIDs).Find(&projects)

	projectMap := make(map[uint]interface{})
	// Map projects by ID
	// This is a template - actual implementation needs proper type handling
	return projectMap
}

// CreateIDMap creates a map of entities keyed by their ID
// This is a generic helper for creating lookup maps
func CreateIDMap[T any, K comparable](items []T, keyExtractor func(T) K) map[K]T {
	idMap := make(map[K]T, len(items))
	for _, item := range items {
		key := keyExtractor(item)
		idMap[key] = item
	}
	return idMap
}

// CreatePtrIDMap creates a map of entity pointers keyed by their ID
// This is useful when you need to return pointers from the map
func CreatePtrIDMap[T any, K comparable](items []*T, keyExtractor func(*T) K) map[K]*T {
	idMap := make(map[K]*T, len(items))
	for _, item := range items {
		key := keyExtractor(item)
		idMap[key] = item
	}
	return idMap
}

// CollectIDs collects IDs from items where the condition is true
// This is useful for conditional ID extraction
func CollectIDs[T any](items []T, condition func(T) bool, keyExtractor func(T) uint) []uint {
	idMap := make(map[uint]bool)

	for _, item := range items {
		if condition(item) {
			id := keyExtractor(item)
			if id > 0 {
				idMap[id] = true
			}
		}
	}

	ids := make([]uint, 0, len(idMap))
	for id := range idMap {
		ids = append(ids, id)
	}

	return ids
}
