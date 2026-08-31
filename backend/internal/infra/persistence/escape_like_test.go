package persistence

import "testing"

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		name string
		term string
		want string
	}{
		{
			name: "escapes all three metacharacters mixed",
			// A literal backslash is doubled first; then % and _ get their own
			// backslash each. Wrong escape order would double the freshly
			// inserted backslashes and leave live wildcards behind.
			term: `a\b%c_d`,
			want: `a\\b\%c\_d`,
		},
		{
			name: "backslash before percent",
			term: `100\%`,
			want: `100\\\%`,
		},
		{
			name: "plain term unchanged",
			term: "bang cong",
			want: "bang cong",
		},
		{
			name: "standalone wildcards each escaped",
			term: "%_",
			want: `\%\_`,
		},
		{
			name: "empty term unchanged",
			term: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeLike(tt.term); got != tt.want {
				t.Errorf("escapeLike(%q) = %q, want %q", tt.term, got, tt.want)
			}
		})
	}
}
