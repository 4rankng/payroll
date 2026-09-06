package excelkit

import (
	"strings"

	"github.com/gosimple/unidecode"
)

// AliasTable maps raw header-cell aliases to a canonical field name. Tables
// stay local to the owning importer; this type only standardizes resolution.
//
// Keys are matched with the union of the in-repo resolution idioms, in
// priority order:
//  1. whole cell, trimmed + lowercased (wallet-bulk style single-line headers)
//  2. whole cell with internal whitespace collapsed (OnePay style)
//  3. collapse of the diacritic-free form (OnePay unicode retry — Vietnamese
//     text may arrive in either unicode form)
//  4. each newline-separated segment, trimmed + lowercased (bilingual
//     multi-line headers like "STT\n(Ord. No.)")
//
// The segment step runs last so a segment can never preempt a whole-cell
// match: the OnePay island resolves collapsed-first, and a stacked header
// whose stray segment happens to equal another field's alias must not steal
// the cell. Segments remain a pure fallback for stacked headers whose
// whole-cell forms match nothing.
type AliasTable map[string]string

// Resolve maps one raw header cell to its canonical field name. The second
// return value reports whether any alias matched.
func (t AliasTable) Resolve(cell string) (string, bool) {
	if canonical, ok := t[strings.ToLower(strings.TrimSpace(cell))]; ok {
		return canonical, true
	}
	collapsed := CollapseHeader(cell)
	if canonical, ok := t[collapsed]; ok {
		return canonical, true
	}
	if canonical, ok := t[CollapseHeader(unidecode.Unidecode(cell))]; ok {
		return canonical, true
	}
	for line := range strings.SplitSeq(cell, "\n") {
		if canonical, ok := t[strings.ToLower(strings.TrimSpace(line))]; ok {
			return canonical, true
		}
	}
	return "", false
}
