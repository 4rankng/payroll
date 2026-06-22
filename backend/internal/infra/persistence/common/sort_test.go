package common

import "testing"

func TestSanitizeSortColumn(t *testing.T) {
	const def = "created_at"

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty uses default", "", def},
		{"whitespace uses default", "   ", def},
		{"plain identifier kept", "amount", "amount"},
		{"underscore identifier kept", "created_at", "created_at"},
		{"qualified identifier kept", "timesheets.date", "timesheets.date"},
		{"qualified with underscore kept", "t.paid_amount", "t.paid_amount"},
		{"leading digit rejected", "1col", def},
		{"space rejected", "col DESC", def},
		{"subquery rejected", "(SELECT 1)", def},
		{"sleep injection rejected", "(SELECT SLEEP(5))", def},
		{"comma list rejected", "a,b", def},
		{"semicolon rejected", "a; DROP TABLE u", def},
		{"quote rejected", "a'b", def},
		{"comment marker rejected", "a--", def},
		{"comment marker mid rejected", "a--b", def},
		{"double dot rejected", "a.b.c", def},
		{"slash rejected", "../etc", def},
		{"paren in payload rejected", "IF(1=1,1,0)", def},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeSortColumn(tt.in, def); got != tt.want {
				t.Errorf("SanitizeSortColumn(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeSortOrder(t *testing.T) {
	tests := []struct {
		in         string
		defaultDir string
		want       string
	}{
		{"asc", "DESC", "ASC"},
		{"DESC", "ASC", "DESC"},
		{"desc", "DESC", "DESC"},
		{"", "DESC", "DESC"},
		{"", "", "DESC"},
		{"", "asc", "ASC"},
		{"random", "DESC", "DESC"},
		{"random", "", "DESC"},
		{"random", "weird", "DESC"},
		{"; DROP", "DESC", "DESC"},
	}
	for _, tt := range tests {
		if got := SanitizeSortOrder(tt.in, tt.defaultDir); got != tt.want {
			t.Errorf("SanitizeSortOrder(%q, %q) = %q, want %q", tt.in, tt.defaultDir, got, tt.want)
		}
	}
}
