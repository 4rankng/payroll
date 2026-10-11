// Package approval implements the approval-request engine: durable request
// lifecycle (pending → approved → executed | rejected → rescheduled →
// escalated | expired | cancelled) with deterministic per-type preconditions,
// server-derived payloads, and a reject-with-note retry loop that re-runs at
// 08:00 ICT (max 3 attempts, then escalation).
//
// Boundary rules enforced here (red-team constraints):
//   - request payloads are SERVER-DERIVED: filers supply identifying params
//     only (cycle date, timesheet id, …) — never recipients or message text;
//   - the filer role chooses from a per-type allowlist (partners may file none
//     of the send_* types);
//   - decisions (approve/reject/cancel) are admin-only, re-checked here even
//     though Casbin already denies other roles at the path level;
//   - unattended runs (the retry worker) may only re-file (rescheduled →
//     pending) — they never approve and never send anything.
package approval

import (
	"context"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

// CycleReader reads the deterministic data behind one statement cycle (the
// "sao kê" period at a report date). Implemented by
// *payroll.PayrollReportByProjectService.
type CycleReader interface {
	GetProjectsForPayrollReport(ctx context.Context, atDate time.Time) ([]*domainServices.ProjectReportData, error)
}

// UserDirectory resolves users for recipient derivation and notification fan
// out. Implemented by *user.UserService's repository (domain.UserRepository).
type UserDirectory interface {
	GetByID(ctx context.Context, id uint) (*domain.User, error)
	GetByIDs(ctx context.Context, ids []uint) (map[uint]*domain.User, error)
	ListByRole(ctx context.Context, role domain.UserRole) ([]*domain.User, error)
}

// ProjectUserDirectory resolves the users attached to a project — the source
// of server-derived recipients. Implemented by domain.ProjectUserRepository.
type ProjectUserDirectory interface {
	GetProjectUsers(ctx context.Context, projectID uint) ([]*domain.ProjectUser, error)
}

// EmployeeDirectory resolves employee display names for payload snapshots.
type EmployeeDirectory interface {
	GetByID(ctx context.Context, id uint) (*domain.Employee, error)
}

// ProjectDirectory resolves project display names for payload snapshots.
type ProjectDirectory interface {
	GetByID(ctx context.Context, id uint) (*domain.Project, error)
}

// TimesheetReader reads timesheet facts for void requests and preconditions.
// Implemented by domain.TimesheetRepository.
type TimesheetReader interface {
	GetByID(ctx context.Context, id uint) (*domain.Timesheet, error)
}

// EmailDispatch hands a payroll-statement email to the EXISTING async email
// pipeline. The approval engine never sends mail itself. Implemented by
// *asynq.Client.EnqueuePayrollReportEmail.
type EmailDispatch interface {
	EnqueuePayrollReportEmail(req dto.SendPayrollReportEmailRequest, initiatedBy uint, requestKey string) (string, bool, error)
}

// Drafter writes the LLM email draft for statement-email requests from the
// frozen payload facts. Unattended-safe: it only produces text, never sends.
// A drafting failure NEVER blocks filing — the request degrades to the
// deterministic payload and the reviewer still decides.
type Drafter interface {
	DraftStatementEmail(ctx context.Context, facts StatementDraftFacts, guidance string) (subject, body string, err error)
}

// StatementDraftFacts is the frozen fact sheet for one statement-email draft.
type StatementDraftFacts struct {
	ReportAtDate   string
	PeriodLabel    string
	TotalAmount    string
	TimesheetCount int
	RecipientCount int
}

// Notifier delivers in-app/push notifications (escalation fan out).
// Implemented by *notification.NotificationService.
type Notifier interface {
	CreateNotification(ctx context.Context, userID uint, notificationType domain.NotificationType, title, message string) error
}

// DrafterFunc adapts a drafting function onto the Drafter port.
type DrafterFunc func(ctx context.Context, facts StatementDraftFacts, guidance string) (string, string, error)

// DraftStatementEmail implements Drafter.
func (f DrafterFunc) DraftStatementEmail(ctx context.Context, facts StatementDraftFacts, guidance string) (string, string, error) {
	return f(ctx, facts, guidance)
}
