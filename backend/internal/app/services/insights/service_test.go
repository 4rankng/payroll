package insights

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-server/internal/domain"
)

type fakeFindings struct {
	stored         []*domain.DailyFinding
	seen           map[string]bool
	dismissedTypes map[string]bool
	dismissCalls   []uint
}

func (f *fakeFindings) Create(_ context.Context, finding *domain.DailyFinding) error {
	for _, s := range f.stored {
		if s.Date == finding.Date && s.Fingerprint == finding.Fingerprint {
			return nil
		}
	}
	f.stored = append(f.stored, finding)
	return nil
}

func (f *fakeFindings) ListByDate(_ context.Context, date string) ([]*domain.DailyFinding, error) {
	var out []*domain.DailyFinding
	for _, s := range f.stored {
		if s.Date == date {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeFindings) RecentlySeenFingerprints(context.Context, string, int) (map[string]bool, error) {
	return f.seen, nil
}

func (f *fakeFindings) DismissedTypes(context.Context, string, int) (map[string]bool, error) {
	return f.dismissedTypes, nil
}

func (f *fakeFindings) GetByID(_ context.Context, id uint) (*domain.DailyFinding, error) {
	for _, s := range f.stored {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, domain.NewNotFoundError("finding")
}

func (f *fakeFindings) Dismiss(_ context.Context, id uint, _ time.Time) error {
	f.dismissCalls = append(f.dismissCalls, id)
	return nil
}

type fakeCollector struct{ candidates []Candidate }

func (f *fakeCollector) Name() string { return "fake" }
func (f *fakeCollector) Collect(context.Context, time.Time) ([]Candidate, error) {
	return f.candidates, nil
}

type fakeNarrator struct {
	text string
	err  error
}

func (f *fakeNarrator) WriteFindingNarrative(context.Context, string, map[string]any) (string, error) {
	return f.text, f.err
}

type fakeNotifier struct{ calls int }

func (f *fakeNotifier) CreateNotification(context.Context, uint, domain.NotificationType, string, string) error {
	f.calls++
	return nil
}

type fakeUsers struct{}

func (fakeUsers) ListByRole(_ context.Context, _ domain.UserRole) ([]*domain.User, error) {
	return []*domain.User{{ID: 1}}, nil
}

type fakeClock struct{ t time.Time }

func (f fakeClock) Now() time.Time    { return f.t }
func (f fakeClock) NowUTC() time.Time { return f.t.UTC() }
func (f fakeClock) TodayStart() time.Time {
	y, m, d := f.t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, f.t.Location())
}
func (f fakeClock) TodayEnd() time.Time {
	y, m, d := f.t.Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, f.t.Location())
}
func (f fakeClock) UnixNow() int64 { return f.t.Unix() }

func newDigest(t *testing.T, candidates []Candidate, findings *fakeFindings, narrator *fakeNarrator, notifier *fakeNotifier) *Service {
	t.Helper()
	return NewService(Deps{
		Findings: findings,
		Clk:      fakeClock{t: time.Date(2026, 10, 11, 7, 30, 0, 0, time.Local)},
		LLM:      narrator,
		Notifier: notifier,
	}, fakeUsers{}, &fakeCollector{candidates: candidates})
}

func digestCandidates() []Candidate {
	return []Candidate{
		{Fingerprint: "a", Type: "cron_failure", Severity: domain.DailySeverityWarning, Facts: map[string]any{"summary": "cron A thất bại"}},
		{Fingerprint: "b", Type: "import_failure", Severity: domain.DailySeverityCritical, Facts: map[string]any{"summary": "12 import lỗi"}},
		{Fingerprint: "c", Type: "other", Severity: domain.DailySeverityInfo, Facts: map[string]any{"summary": "thông tin"}},
	}
}

func TestDigestStoresNarrativesAndPushesCriticalOnly(t *testing.T) {
	findings := &fakeFindings{}
	notifier := &fakeNotifier{}
	svc := newDigest(t, digestCandidates(), findings, &fakeNarrator{text: "Một câu tóm tắt."}, notifier)
	if err := svc.RunDailyDigest(context.Background()); err != nil {
		t.Fatalf("digest: %v", err)
	}
	if len(findings.stored) != 3 {
		t.Fatalf("stored = %d, want 3", len(findings.stored))
	}
	for _, f := range findings.stored {
		if f.Narrative != "Một câu tóm tắt." {
			t.Fatalf("narrative missing on %s", f.Fingerprint)
		}
	}
	if notifier.calls != 1 {
		t.Fatalf("push calls = %d, want 1 (critical only)", notifier.calls)
	}
}

func TestDigestDedupAndSuppression(t *testing.T) {
	findings := &fakeFindings{
		seen:           map[string]bool{"a": true},
		dismissedTypes: map[string]bool{"import_failure": true},
	}
	svc := newDigest(t, digestCandidates(), findings, &fakeNarrator{}, &fakeNotifier{})
	if err := svc.RunDailyDigest(context.Background()); err != nil {
		t.Fatalf("digest: %v", err)
	}
	if len(findings.stored) != 1 || findings.stored[0].Fingerprint != "c" {
		t.Fatalf("stored = %+v, want only the unseen/unsuppressed candidate", findings.stored)
	}
}

func TestDigestCapsAtFiveBySeverity(t *testing.T) {
	var candidates []Candidate
	for i := 0; i < 8; i++ {
		sev := domain.DailySeverityInfo
		if i < 2 {
			sev = domain.DailySeverityCritical
		} else if i < 4 {
			sev = domain.DailySeverityWarning
		}
		candidates = append(candidates, Candidate{Fingerprint: string(rune('a' + i)), Type: "t", Severity: sev, Facts: map[string]any{}})
	}
	findings := &fakeFindings{}
	svc := newDigest(t, candidates, findings, &fakeNarrator{}, &fakeNotifier{})
	if err := svc.RunDailyDigest(context.Background()); err != nil {
		t.Fatalf("digest: %v", err)
	}
	if len(findings.stored) != maxPerDigest {
		t.Fatalf("stored = %d, want cap %d", len(findings.stored), maxPerDigest)
	}
	critical := 0
	for _, f := range findings.stored {
		if f.Severity == domain.DailySeverityCritical {
			critical++
		}
	}
	if critical != 2 {
		t.Fatalf("critical kept = %d, want 2 (severity order preserves criticals under the cap)", critical)
	}
}

func TestDigestSurvivesNarrativeOutage(t *testing.T) {
	findings := &fakeFindings{}
	svc := newDigest(t, digestCandidates()[:1], findings, &fakeNarrator{err: errors.New("provider down")}, &fakeNotifier{})
	if err := svc.RunDailyDigest(context.Background()); err != nil {
		t.Fatalf("digest must survive a narrative outage: %v", err)
	}
	if len(findings.stored) != 1 || findings.stored[0].Narrative != "" {
		t.Fatalf("finding stored without narrative: %+v", findings.stored)
	}
}
