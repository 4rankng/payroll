package llmchat

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Small pure helpers shared by the catalog and prompt builders.

func jsonUnmarshal(s string, dest any) error { return json.Unmarshal([]byte(s), dest) }

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func containsInt(list []int, v int) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// joinCapped joins up to cap items and reports whether anything was truncated.
func joinCapped(items []string, cap int) (string, bool) {
	if len(items) <= cap {
		return strings.Join(items, ", "), false
	}
	return strings.Join(items[:cap], ", "), true
}

func truncatedSuffix(truncated bool) string {
	if truncated {
		return "…"
	}
	return ""
}

func intSliceToStrings(items []int) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strconv.Itoa(item))
	}
	return out
}

func itoa(v int) string { return strconv.Itoa(v) }

// truncateRunes bounds a string for storage/display (content entering prompts
// and audit rows is never unbounded).
func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
