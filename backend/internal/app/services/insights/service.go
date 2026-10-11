// Package insights builds the daily Tổng Quan digest: deterministic DB-signal
// collectors compute candidate findings, hard anti-spam filters decide what
// survives, and the LLM only writes the Vietnamese narrative. Facts are
// computed in Go; the model never derives numbers.
package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	llmsvc "api-server/internal/app/services/llm"
	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

const (
	// Hard anti-spam bounds (structural, not prompt-based).
	dedupWindowDays = 7
	maxPerDigest    = 5
	narrativeModel  = "mimo-v2.6-flash"
)

// Severity ranks survivors; only critical may push.
func severityRank(s domain.DailyFindingSeverity) int {
	switch s {
	case domain.DailySeverityCritical:
		return 0
	case domain.DailySeverityWarning:
		return 1
	}
	return 2
}

// Candidate is one raw collector signal before filtering.
type Candidate struct {
	Fingerprint string
	Type        string
	Severity    domain.DailyFindingSeverity
	Facts       map[string]any
}

// Collector reads one deterministic signal family.
type Collector interface {
	Name() string
	Collect(ctx context.Context, since time.Time) ([]Candidate, error)
}

// Deps bundles the digest dependencies.
type Deps struct {
	Findings domain.DailyFindingRepository
	Clk      clock.Clock
	LLM      NarrativeWriter
	Notifier Notifier
}

// Notifier fans a critical finding out to admins (in-app + push).
type Notifier interface {
	CreateNotification(ctx context.Context, userID uint, nType domain.NotificationType, title, message string) error
}

// Users resolves the admin fan-out list.
type Users interface {
	ListByRole(ctx context.Context, role domain.UserRole) ([]*domain.User, error)
}

// NarrativeWriter writes one finding's Vietnamese narrative from frozen facts.
type NarrativeWriter interface {
	WriteFindingNarrative(ctx context.Context, findingType string, facts map[string]any) (string, error)
}

// Service runs the daily digest.
type Service struct {
	deps      Deps
	users     Users
	collector Collector
	logger    *slog.Logger
}

// NewService constructs the digest service with its collector set.
func NewService(deps Deps, users Users, collector Collector) *Service {
	return &Service{deps: deps, users: users, collector: collector, logger: observability.GetLogger().With("component", "insights")}
}

// RunDailyDigest collects, filters, narrates and persists one date's digest.
// Idempotent per (date, fingerprint): re-running the same date adds nothing.
func (s *Service) RunDailyDigest(ctx context.Context) error {
	now := s.deps.Clk.Now()
	date := now.Format("2006-01-02")
	since := now.Add(-24 * time.Hour)

	candidates, err := s.collector.Collect(ctx, since)
	if err != nil {
		return fmt.Errorf("collect signals: %w", err)
	}

	seen, err := s.deps.Findings.RecentlySeenFingerprints(ctx, date, dedupWindowDays)
	if err != nil {
		return fmt.Errorf("dedup set: %w", err)
	}
	dismissedTypes, err := s.deps.Findings.DismissedTypes(ctx, date, dedupWindowDays)
	if err != nil {
		return fmt.Errorf("suppression set: %w", err)
	}

	// Hard filters: dedup within the window, suppressed families, cap of 5
	// keeping the most severe survivors.
	var survivors []Candidate
	for _, c := range candidates {
		if seen[c.Fingerprint] || dismissedTypes[c.Type] {
			continue
		}
		survivors = append(survivors, c)
	}
	sortBySeverity(survivors)
	if len(survivors) > maxPerDigest {
		survivors = survivors[:maxPerDigest]
	}

	for _, c := range survivors {
		factsJSON, merr := json.Marshal(c.Facts)
		if merr != nil {
			continue
		}
		finding := &domain.DailyFinding{
			Date:        date,
			Fingerprint: c.Fingerprint,
			Type:        c.Type,
			Severity:    c.Severity,
			Facts:       string(factsJSON),
		}
		// Narrative is best-effort: a provider failure stores the finding
		// with an empty narrative — the card falls back to a facts summary.
		if s.deps.LLM != nil {
			if narrative, werr := s.deps.LLM.WriteFindingNarrative(ctx, c.Type, c.Facts); werr == nil {
				finding.Narrative = narrative
			} else {
				s.logger.Warn("insights: narrative failed, storing facts only", "type", c.Type, "error", werr)
			}
		}
		if cerr := s.deps.Findings.Create(ctx, finding); cerr != nil {
			s.logger.Error("insights: persist finding failed", "type", c.Type, "error", cerr)
			continue
		}
		// Push ONLY for severity=critical — everything else lives on the card.
		if c.Severity == domain.DailySeverityCritical && s.deps.Notifier != nil && s.users != nil {
			s.notifyAdmins(ctx, finding)
		}
	}
	s.logger.Info("insights: daily digest complete", "date", date, "candidates", len(candidates), "stored", len(survivors))
	return nil
}

// GetDigest returns one date's findings, hidden when nothing survived.
func (s *Service) GetDigest(ctx context.Context, date string) ([]*domain.DailyFinding, error) {
	if date == "" {
		date = s.deps.Clk.Now().Format("2006-01-02")
	}
	return s.deps.Findings.ListByDate(ctx, date)
}

// Dismiss suppresses a finding family (7-day suppression window).
func (s *Service) Dismiss(ctx context.Context, id uint) error {
	return s.deps.Findings.Dismiss(ctx, id, s.deps.Clk.Now())
}

func (s *Service) notifyAdmins(ctx context.Context, finding *domain.DailyFinding) {
	admins, err := s.users.ListByRole(ctx, domain.RoleAdmin)
	if err != nil {
		s.logger.Error("insights: admin fan-out failed", "error", err)
		return
	}
	for _, admin := range admins {
		_ = s.deps.Notifier.CreateNotification(ctx, admin.ID, domain.NotificationTypeSystemAlert,
			"Phát hiện quan trọng trên Tổng Quan", fallbackSummary(finding))
	}
}

func sortBySeverity(list []Candidate) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && severityRank(list[j].Severity) < severityRank(list[j-1].Severity); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

// fallbackSummary renders one line from the frozen facts — used for the
// critical push and wherever the narrative is absent.
func fallbackSummary(f *domain.DailyFinding) string {
	var facts map[string]any
	if json.Unmarshal([]byte(f.Facts), &facts) == nil {
		if label, ok := facts["summary"].(string); ok && label != "" {
			return label
		}
	}
	return fmt.Sprintf("Phát hiện loại %s (mức %s) — xem chi tiết trên Tổng Quan.", f.Type, f.Severity)
}

// LLMNarrativeWriter adapts the llm service onto the narrative port.
type LLMNarrativeWriter struct{ Service *llmsvc.Service }

// WriteFindingNarrative writes ≤3 Vietnamese sentences from frozen facts.
func (w LLMNarrativeWriter) WriteFindingNarrative(ctx context.Context, findingType string, facts map[string]any) (string, error) {
	factsJSON, err := json.Marshal(facts)
	if err != nil {
		return "", err
	}
	system := "Bạn tóm tắt một phát hiện vận hành cho chủ hệ thống payroll bằng tiếng Việt, tối đa 3 câu. " +
		"CHỈ dùng nguyên văn các con số trong dữ kiện — tuyệt đối không tính toán hay bịa số. " +
		"Mỗi phát hiện phải có một câu 'vì sao đáng chú ý'. Không thêm tiêu đề hay định dạng markdown."
	user := fmt.Sprintf("Loại phát hiện: %s\nDữ kiện (nguyên văn): %s", findingType, string(factsJSON))
	resp, err := w.Service.Chat(ctx, 0, chatRequest(system, user, w.Service.ModelForRoute(ctx, "deliverable")))
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("mimo trả về kết quả rỗng")
	}
	return resp.Choices[0].Message.Content, nil
}

// chatRequest builds the minimal narrative request.
func chatRequest(system, user, model string) llmport.ChatRequest {
	if model == "" {
		model = narrativeModel
	}
	return llmport.ChatRequest{
		Model: model,
		Messages: []llmport.Message{
			{Role: "system", Content: []llmport.ContentPart{llmport.TextPart(system)}},
			{Role: "user", Content: []llmport.ContentPart{llmport.TextPart(user)}},
		},
		MaxTokens: 300,
	}
}
