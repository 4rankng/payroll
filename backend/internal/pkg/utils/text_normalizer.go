package utils

import (
	"strings"
	"sync"

	"github.com/gosimple/unidecode"
)

// VietnameseNormalizer handles Vietnamese text normalization for search purposes
type VietnameseNormalizer struct {
	cache map[string]string
	mutex sync.RWMutex
}

// NewVietnameseNormalizer creates a new instance of Vietnamese normalizer
func NewVietnameseNormalizer() *VietnameseNormalizer {
	return &VietnameseNormalizer{
		cache: make(map[string]string),
	}
}

// Normalize converts Vietnamese text to ASCII representation for search
// This function:
// 1. Converts to lowercase
// 2. Removes Vietnamese diacritics (áàãạả -> a, éèẽẹẻ -> e, etc.)
// 3. Caches results for performance
func (v *VietnameseNormalizer) Normalize(text string) string {
	if text == "" {
		return ""
	}

	// Check cache first
	v.mutex.RLock()
	if normalized, exists := v.cache[text]; exists {
		v.mutex.RUnlock()
		return normalized
	}
	v.mutex.RUnlock()

	// Normalize the text
	lowercased := strings.ToLower(text)
	normalized := unidecode.Unidecode(lowercased)

	// Cache the result
	v.mutex.Lock()
	v.cache[text] = normalized
	v.mutex.Unlock()

	return normalized
}

// NormalizeForSearch prepares text for database search operations
// Returns the normalized text wrapped with SQL wildcards
func (v *VietnameseNormalizer) NormalizeForSearch(searchTerm string) string {
	normalized := v.Normalize(searchTerm)
	return "%" + normalized + "%"
}

// NormalizeBatch normalizes multiple strings at once for efficiency
func (v *VietnameseNormalizer) NormalizeBatch(texts []string) []string {
	results := make([]string, len(texts))
	for i, text := range texts {
		results[i] = v.Normalize(text)
	}
	return results
}

// ClearCache clears the normalization cache
func (v *VietnameseNormalizer) ClearCache() {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.cache = make(map[string]string)
}

// GetCacheSize returns the current cache size
func (v *VietnameseNormalizer) GetCacheSize() int {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	return len(v.cache)
}

// Global instance for convenience
var globalNormalizer = NewVietnameseNormalizer()

// NormalizeVietnamese is a convenience function that uses the global normalizer instance
func NormalizeVietnamese(text string) string {
	return globalNormalizer.Normalize(text)
}

// NormalizeVietnameseForSearch is a convenience function for search operations
func NormalizeVietnameseForSearch(searchTerm string) string {
	return globalNormalizer.NormalizeForSearch(searchTerm)
}
