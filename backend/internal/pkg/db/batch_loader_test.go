package db

import (
	"testing"
)

type TestEntity struct {
	ID   uint
	Name string
}

func TestExtractIDs(t *testing.T) {
	tests := []struct {
		name     string
		items    []*TestEntity
		expected []uint
	}{
		{
			name: "extract unique IDs",
			items: []*TestEntity{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
				{ID: 3, Name: "C"},
			},
			expected: []uint{1, 2, 3},
		},
		{
			name: "handle duplicates",
			items: []*TestEntity{
				{ID: 1, Name: "A"},
				{ID: 2, Name: "B"},
				{ID: 1, Name: "C"},
			},
			expected: []uint{1, 2},
		},
		{
			name: "skip zero IDs",
			items: []*TestEntity{
				{ID: 1, Name: "A"},
				{ID: 0, Name: "B"},
				{ID: 2, Name: "C"},
			},
			expected: []uint{1, 2},
		},
		{
			name:     "empty slice",
			items:    []*TestEntity{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractIDs(tt.items, func(e *TestEntity) uint { return e.ID })

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d IDs, got %d", len(tt.expected), len(result))
				return
			}

			// Convert to map for easier comparison (order doesn't matter)
			resultMap := make(map[uint]bool)
			for _, id := range result {
				resultMap[id] = true
			}

			for _, expectedID := range tt.expected {
				if !resultMap[expectedID] {
					t.Errorf("expected ID %d not found in result", expectedID)
				}
			}
		})
	}
}

func TestCreateIDMap(t *testing.T) {
	items := []TestEntity{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}

	result := CreateIDMap(items, func(e TestEntity) uint { return e.ID })

	if len(result) != 3 {
		t.Errorf("expected map with 3 items, got %d", len(result))
	}

	if result[1].Name != "A" {
		t.Errorf("expected Name 'A', got '%s'", result[1].Name)
	}
	if result[2].Name != "B" {
		t.Errorf("expected Name 'B', got '%s'", result[2].Name)
	}
	if result[3].Name != "C" {
		t.Errorf("expected Name 'C', got '%s'", result[3].Name)
	}
}

func TestCreatePtrIDMap(t *testing.T) {
	items := []*TestEntity{
		{ID: 1, Name: "A"},
		{ID: 2, Name: "B"},
		{ID: 3, Name: "C"},
	}

	result := CreatePtrIDMap(items, func(e *TestEntity) uint { return e.ID })

	if len(result) != 3 {
		t.Errorf("expected map with 3 items, got %d", len(result))
	}

	if result[1] == nil || result[1].Name != "A" {
		t.Errorf("expected entity with Name 'A'")
	}
	if result[2] == nil || result[2].Name != "B" {
		t.Errorf("expected entity with Name 'B'")
	}
	if result[3] == nil || result[3].Name != "C" {
		t.Errorf("expected entity with Name 'C'")
	}
}

func TestCollectIDs(t *testing.T) {
	items := []TestEntity{
		{ID: 1, Name: "Active"},
		{ID: 2, Name: "Inactive"},
		{ID: 3, Name: "Active"},
		{ID: 4, Name: "Inactive"},
	}

	// Collect IDs where Name is "Active"
	result := CollectIDs(
		items,
		func(e TestEntity) bool { return e.Name == "Active" },
		func(e TestEntity) uint { return e.ID },
	)

	if len(result) != 2 {
		t.Errorf("expected 2 IDs, got %d", len(result))
		return
	}

	// Convert to map for easier comparison
	resultMap := make(map[uint]bool)
	for _, id := range result {
		resultMap[id] = true
	}

	if !resultMap[1] || !resultMap[3] {
		t.Errorf("expected IDs 1 and 3, got %v", result)
	}
}
