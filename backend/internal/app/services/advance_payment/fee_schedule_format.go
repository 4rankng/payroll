package advance_payment

import (
	"fmt"
	"strings"

	"api-server/internal/domain"
)

// FormatScheduleSummary builds a Vietnamese one-line description of an entry's
// tier structure for use in audit log messages and admin UI summaries.
//
// Examples:
//
//	[{0, 2.0}]                              -> "2% (tối thiểu 10.000 VND)"
//	[{0, 2.0}, {3500000, 1.3}]              -> "2% / 1,3% trên 3.500.000 VND (tối thiểu 10.000 VND)"
//	[{0, 2.0}, {1000000, 1.5}, {5000000, 1.0}]
//	                                        -> "2% / 1,5% trên 1.000.000 VND / 1% trên 5.000.000 VND (tối thiểu 10.000 VND)"
func FormatScheduleSummary(entry domain.FeeScheduleEntry) string {
	if len(entry.Tiers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(entry.Tiers))
	for i, tier := range entry.Tiers {
		pct := formatPercentage(tier.Percentage)
		if i == 0 {
			parts = append(parts, pct)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s trên %s VND", pct, formatVNDInt(tier.MinAmount)))
	}
	// MinFeeVND = 0 means no minimum-fee floor (e.g. a fully free schedule);
	// the "tối thiểu 0 VND" clause would be misleading, so omit it.
	if entry.MinFeeVND == 0 {
		return strings.Join(parts, " / ")
	}
	return fmt.Sprintf("%s (tối thiểu %s VND)", strings.Join(parts, " / "), formatVNDInt(entry.MinFeeVND))
}

func formatPercentage(p float64) string {
	// Drop trailing zeros, swap decimal separator to comma per Vietnamese
	// locale convention. 2.0 -> "2%", 1.3 -> "1,3%".
	s := fmt.Sprintf("%g", p)
	s = strings.ReplaceAll(s, ".", ",")
	return s + "%"
}

func formatVNDInt(amount uint64) string {
	// Insert dots as thousands separators: 3500000 -> "3.500.000".
	s := fmt.Sprintf("%d", amount)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte('.')
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte('.')
		}
	}
	return b.String()
}
