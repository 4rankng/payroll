package workers

import (
	"testing"
	"time"

	"api-server/internal/pkg/clock"
)

func TestResolveImportForMonth_PrefersSelectedMonth(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation))

	got := resolveImportForMonth("2026-06")
	if got != "2026-06" {
		t.Fatalf("expected selected month 2026-06, got %s", got)
	}
}

func TestResolveImportForMonth_FallsBackToCurrentAdvanceMonth(t *testing.T) {
	fake := clock.NewAutoFake()
	clock.SetGlobal(fake)
	t.Cleanup(func() {
		clock.SetGlobal(clock.New())
	})

	fake.Set(time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation))

	got := resolveImportForMonth("")
	if got != "2026-05" {
		t.Fatalf("expected fallback month 2026-05 on 2026-06-20, got %s", got)
	}
}
