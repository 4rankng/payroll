package dashboard

import "testing"

func TestExpectedMaxAdvUsesConfiguredPercent(t *testing.T) {
	tests := []struct {
		name     string
		salary   int64
		percent  uint64
		expected int64
	}{
		{name: "default 70", salary: 1_000_000, percent: 70, expected: 700_000},
		{name: "higher percent", salary: 1_000_000, percent: 85, expected: 850_000},
		{name: "flooring", salary: 123_457, percent: 73, expected: 90_123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expectedMaxAdv(tt.salary, tt.percent); got != tt.expected {
				t.Fatalf("expectedMaxAdv(%d, %d) = %d, want %d", tt.salary, tt.percent, got, tt.expected)
			}
		})
	}
}
