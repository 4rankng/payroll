package main

import (
	"fmt"
	"reflect"
	"strings"
)

func AssertEqual(field string, expected, actual interface{}) error {
	if expected != actual {
		return fmt.Errorf("%s: expected %v, got %v", field, expected, actual)
	}
	return nil
}

func AssertNotEqual(field string, expected, actual interface{}) error {
	if expected == actual {
		return fmt.Errorf("%s: expected != %v, but got equal", field, expected)
	}
	return nil
}

func AssertNotNil(field string, value interface{}) error {
	if value == nil {
		return fmt.Errorf("%s: expected non-nil, got nil", field)
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		return fmt.Errorf("%s: expected non-nil pointer, got nil", field)
	}
	return nil
}

func AssertTrue(field string, value bool) error {
	if !value {
		return fmt.Errorf("%s: expected true, got false", field)
	}
	return nil
}

func AssertFalse(field string, value bool) error {
	if value {
		return fmt.Errorf("%s: expected false, got true", field)
	}
	return nil
}

func AssertGreaterThan(field string, threshold, actual interface{}) error {
	switch t := threshold.(type) {
	case int:
		a, ok := actual.(int)
		if !ok || a <= t {
			return fmt.Errorf("%s: expected > %v, got %v", field, threshold, actual)
		}
	case int64:
		a, ok := actual.(int64)
		if !ok || a <= t {
			return fmt.Errorf("%s: expected > %v, got %v", field, threshold, actual)
		}
	case uint64:
		a, ok := actual.(uint64)
		if !ok || a <= t {
			return fmt.Errorf("%s: expected > %v, got %v", field, threshold, actual)
		}
	case uint:
		a, ok := actual.(uint)
		if !ok || a <= t {
			return fmt.Errorf("%s: expected > %v, got %v", field, threshold, actual)
		}
	case float64:
		a, ok := actual.(float64)
		if !ok || a <= t {
			return fmt.Errorf("%s: expected > %v, got %v", field, threshold, actual)
		}
	default:
		return fmt.Errorf("%s: unsupported type for comparison: %T", field, threshold)
	}
	return nil
}

func AssertGreaterOrEqual(field string, threshold, actual interface{}) error {
	switch t := threshold.(type) {
	case int:
		a, ok := actual.(int)
		if !ok || a < t {
			return fmt.Errorf("%s: expected >= %v, got %v", field, threshold, actual)
		}
	case int64:
		a, ok := actual.(int64)
		if !ok || a < t {
			return fmt.Errorf("%s: expected >= %v, got %v", field, threshold, actual)
		}
	case uint64:
		a, ok := actual.(uint64)
		if !ok || a < t {
			return fmt.Errorf("%s: expected >= %v, got %v", field, threshold, actual)
		}
	case uint:
		a, ok := actual.(uint)
		if !ok || a < t {
			return fmt.Errorf("%s: expected >= %v, got %v", field, threshold, actual)
		}
	case float64:
		a, ok := actual.(float64)
		if !ok || a < t {
			return fmt.Errorf("%s: expected >= %v, got %v", field, threshold, actual)
		}
	default:
		return fmt.Errorf("%s: unsupported type for comparison: %T", field, threshold)
	}
	return nil
}

func AssertContains(field, haystack, needle string) error {
	if !strings.Contains(haystack, needle) {
		return fmt.Errorf("%s: expected to contain %q, got %q", field, needle, haystack)
	}
	return nil
}

func AssertStatusIn(field string, status string, allowed []string) error {
	for _, a := range allowed {
		if status == a {
			return nil
		}
	}
	return fmt.Errorf("%s: status %q not in allowed values %v", field, status, allowed)
}

func AssertSliceLen(field string, length, expected int) error {
	if length != expected {
		return fmt.Errorf("%s: expected length %d, got %d", field, expected, length)
	}
	return nil
}

func AssertSliceMinLen(field string, length, minLen int) error {
	if length < minLen {
		return fmt.Errorf("%s: expected min length %d, got %d", field, minLen, length)
	}
	return nil
}

func AssertLessThan(field string, threshold, actual interface{}) error {
	switch t := threshold.(type) {
	case int:
		a, ok := actual.(int)
		if !ok || a >= t {
			return fmt.Errorf("%s: expected < %v, got %v", field, threshold, actual)
		}
	case int64:
		a, ok := actual.(int64)
		if !ok || a >= t {
			return fmt.Errorf("%s: expected < %v, got %v", field, threshold, actual)
		}
	case uint64:
		a, ok := actual.(uint64)
		if !ok || a >= t {
			return fmt.Errorf("%s: expected < %v, got %v", field, threshold, actual)
		}
	case float64:
		a, ok := actual.(float64)
		if !ok || a >= t {
			return fmt.Errorf("%s: expected < %v, got %v", field, threshold, actual)
		}
	default:
		return fmt.Errorf("%s: unsupported type for comparison: %T", field, threshold)
	}
	return nil
}

func AssertLen(field string, actualLen, expected int) error {
	if actualLen != expected {
		return fmt.Errorf("%s: expected length %d, got %d", field, expected, actualLen)
	}
	return nil
}

func AssertHasPrefix(field, full, prefix string) error {
	if !strings.HasPrefix(full, prefix) {
		return fmt.Errorf("%s: expected prefix %q, got %q", field, prefix, full)
	}
	return nil
}
