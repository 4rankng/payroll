package utils

import (
	"testing"
)

func TestNewVietnameseNormalizer(t *testing.T) {
	normalizer := NewVietnameseNormalizer()
	if normalizer == nil {
		t.Fatal("NewVietnameseNormalizer() returned nil")
		return
	}

	if normalizer.cache == nil {
		t.Error("NewVietnameseNormalizer() created normalizer with nil cache")
	}

	if len(normalizer.cache) != 0 {
		t.Errorf("NewVietnameseNormalizer() cache should be empty, got %d items", len(normalizer.cache))
	}
}

func TestVietnameseNormalizer_Normalize(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "english text",
			input: "Hello World",
			want:  "hello world",
		},
		{
			name:  "vietnamese with accents",
			input: "Nguyễn Văn A",
			want:  "nguyen van a",
		},
		{
			name:  "mixed case vietnamese",
			input: "HỒNG HÀ",
			want:  "hong ha",
		},
		{
			name:  "already normalized",
			input: "abc",
			want:  "abc",
		},
		{
			name:  "numbers and text",
			input: "Số 123",
			want:  "so 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVietnameseNormalizer_NormalizeForSearch(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple search",
			input: "Nguyễn",
			want:  "%nguyen%",
		},
		{
			name:  "empty search",
			input: "",
			want:  "%%",
		},
		{
			name:  "english search",
			input: "Test",
			want:  "%test%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.NormalizeForSearch(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeForSearch(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVietnameseNormalizer_NormalizeBatch(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "empty batch",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "single item",
			input: []string{"Hà Nội"},
			want:  []string{"ha noi"},
		},
		{
			name:  "multiple items",
			input: []string{"Hà Nội", "Hồ Chí Minh", "Đà Nẵng"},
			want:  []string{"ha noi", "ho chi minh", "da nang"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizer.NormalizeBatch(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("NormalizeBatch() returned %d items, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("NormalizeBatch()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestVietnameseNormalizer_Cache(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	// Normalize a string
	input := "Nguyễn Văn A"
	normalizer.Normalize(input)

	// Check cache size
	if normalizer.GetCacheSize() != 1 {
		t.Errorf("GetCacheSize() = %d, want 1", normalizer.GetCacheSize())
	}

	// Normalize the same string again (should use cache)
	normalizer.Normalize(input)

	// Cache size should still be 1
	if normalizer.GetCacheSize() != 1 {
		t.Errorf("GetCacheSize() after duplicate = %d, want 1", normalizer.GetCacheSize())
	}

	// Add another string
	normalizer.Normalize("Trần Văn B")

	// Cache size should be 2
	if normalizer.GetCacheSize() != 2 {
		t.Errorf("GetCacheSize() after second string = %d, want 2", normalizer.GetCacheSize())
	}
}

func TestVietnameseNormalizer_ClearCache(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	// Add some items to cache
	normalizer.Normalize("Test 1")
	normalizer.Normalize("Test 2")

	if normalizer.GetCacheSize() != 2 {
		t.Errorf("GetCacheSize() before clear = %d, want 2", normalizer.GetCacheSize())
	}

	// Clear cache
	normalizer.ClearCache()

	if normalizer.GetCacheSize() != 0 {
		t.Errorf("GetCacheSize() after clear = %d, want 0", normalizer.GetCacheSize())
	}
}

func TestNormalizeVietnamese(t *testing.T) {
	// Test global convenience function
	result := NormalizeVietnamese("Hà Nội")
	expected := "ha noi"

	if result != expected {
		t.Errorf("NormalizeVietnamese() = %q, want %q", result, expected)
	}
}

func TestNormalizeVietnameseForSearch(t *testing.T) {
	// Test global convenience function
	result := NormalizeVietnameseForSearch("Hà Nội")
	expected := "%ha noi%"

	if result != expected {
		t.Errorf("NormalizeVietnameseForSearch() = %q, want %q", result, expected)
	}
}

func TestVietnameseNormalizer_ConcurrentAccess(t *testing.T) {
	normalizer := NewVietnameseNormalizer()

	done := make(chan bool)

	// Simulate concurrent access
	for i := 0; i < 10; i++ {
		go func(n int) {
			text := "Test"
			normalizer.Normalize(text)
			_ = normalizer.GetCacheSize()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and cache should have the item
	if normalizer.GetCacheSize() == 0 {
		t.Error("Expected cache to have items after concurrent access")
	}
}
