package phone

import "testing"

func TestNormalizeVietnameseMobile(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "domestic", input: "0901234567", want: "0901234567", ok: true},
		{name: "country code", input: "+84 901 234 567", want: "0901234567", ok: true},
		{name: "country code without plus", input: "84901234567", want: "0901234567", ok: true},
		{name: "formatted", input: "090-123-4567", want: "0901234567", ok: true},
		{name: "landline", input: "02812345678", ok: false},
		{name: "invalid prefix", input: "0101234567", ok: false},
		{name: "letters", input: "09012abc67", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeVietnameseMobile(tt.input)
			if tt.ok && err != nil {
				t.Fatalf("NormalizeVietnameseMobile() error = %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("NormalizeVietnameseMobile() expected an error, got %q", got)
			}
			if got != tt.want {
				t.Fatalf("NormalizeVietnameseMobile() = %q, want %q", got, tt.want)
			}
		})
	}
}
