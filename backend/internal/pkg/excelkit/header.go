package excelkit

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// NormalizeHeader normalizes a header cell for matching: trim, NFC unicode
// form, lower-case. Vietnamese text in xlsx files exported from macOS
// sometimes arrives in NFD form, so NFC normalization prevents false misses
// like "ho va ten" vs "họ và tên".
//
// Verbatim port of the BCC normHeader (services/excel). Internal whitespace
// is preserved as-is — use CollapseHeader when the owning format matches on
// collapsed whitespace instead.
func NormalizeHeader(s string) string {
	return strings.ToLower(strings.TrimSpace(norm.NFC.String(s)))
}

// CollapseHeader normalizes a header cell the way the OnePay fee importer
// does: trim, lower-case, internal whitespace collapsed to single spaces.
//
// Verbatim port of settlement's normalizeHeader. Unlike NormalizeHeader it
// does not apply NFC — keep the two distinct; each island's alias table is
// keyed to its own normalization.
func CollapseHeader(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}
