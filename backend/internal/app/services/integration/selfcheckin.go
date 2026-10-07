package integration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"api-server/internal/app/services/attendance"
	"api-server/internal/app/services/identity"
	"api-server/internal/app/services/otp"
	"api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/cache"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/phone"
)

// SelfCheckinMutator is the subset of *project.ProjectEmployeeService the
// update step needs. Routing the bot's enable/disable through the same
// row-locked service method as the admin toggle means the verdict (immediate
// vs queued vs cancelled-pending) always reflects the true assignment state.
type SelfCheckinMutator interface {
	SetSelfCheckinViaChatbot(ctx context.Context, projectID, employeeID uint, enable bool, updatedBy uint) (*project.SelfCheckinChatbotResult, error)
}

// projectRepo is the narrow subset of domain.ProjectRepository the service
// needs: GetByCode resolves the configured supported codes to project IDs
// (the rollout gate), GetByID fetches the geofence and display data for an
// assignment's project.
type projectRepo interface {
	GetByID(ctx context.Context, id uint) (*domain.Project, error)
	GetByCode(ctx context.Context, code string) (*domain.Project, error)
}

// assignmentRepo is the narrow subset of domain.ProjectEmployeeRepository the
// OTP response needs: the employee's assignments (filtered to active rows).
type assignmentRepo interface {
	GetByEmployee(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error)
}

// payrateRepo is the narrow subset of domain.PayrateRepository needed to
// resolve the advisory shift windows (same source as the profile endpoint).
type payrateRepo interface {
	GetByProject(ctx context.Context, projectID uint) ([]*domain.Payrate, error)
}

// SelfCheckinPendingChange summarizes an assignment's queued check-in change.
// Type is "enable" or "disable"; EffectiveFrom is day 1 of the month it lands.
type SelfCheckinPendingChange struct {
	Type          string
	EffectiveFrom time.Time
}

// SelfCheckinTarget carries the project's attendance geofence so the bot can
// answer "where do I check in" without a second call.
type SelfCheckinTarget struct {
	RadiusMeters uint
	Gates        []domain.GeofenceGate
}

// SelfCheckinAssignment is one active assignment of the verified employee,
// restricted to projects whose code is enabled for the chatbot flow.
type SelfCheckinAssignment struct {
	ProjectID        uint
	ProjectName      string
	CheckInEnabled   bool
	PendingChange    *SelfCheckinPendingChange // nil when no change is queued
	CheckInStartDate *time.Time
	PaymentSchedule  string
	Target           *SelfCheckinTarget
	ShiftWindows     []attendance.ShiftWindow
}

// SelfCheckinRequestResult is the explicit outcome of a self check-in OTP
// request. On success it carries everything one round-trip needs: the OTP
// session fields plus the employee's eligible assignments.
type SelfCheckinRequestResult struct {
	Found             bool
	OTPSent           bool
	SessionID         string
	ExpiresIn         int
	EmployeeName      string
	FailureReason     string // "" when OTPSent; else a Failure* constant
	DeliveryErrorCode int    // Zalo business code when FailureDeliveryFailed
	Assignments       []SelfCheckinAssignment
}

// SelfCheckinVerifyResult is the outcome of a successful OTP verification.
type SelfCheckinVerifyResult struct {
	ActionToken string
	ExpiresIn   int
}

// SelfCheckinUpdateResult is the verdict of an enable/disable request, shaped
// so the bot can phrase day-1-8 vs day-9 wording without extra calls.
type SelfCheckinUpdateResult struct {
	Kind                   string // "enable" or "disable"
	Immediate              bool
	EffectiveFrom          time.Time // zero when a queued enable was cancelled
	CancelledPendingEnable bool
}

// SelfCheckinService implements the chatbot's self check-in flow:
// OTP → verify → update. It mirrors the password-reset trio but its store
// instance MUST be built with cache.NewZaloResetStoreWithPrefix(..., "zsc"):
// sessions and verified action tokens then live under "zsc:"/"zsc-ok:", so a
// self-checkin action token can never be replayed at /password-reset/reset
// (and vice versa).
type SelfCheckinService struct {
	store          ResetStore
	userRepo       userRepo
	employeeRepo   employeeRepo
	projectRepo    projectRepo
	assignmentRepo assignmentRepo
	payrateRepo    payrateRepo // may be nil — shift windows then stay empty
	mutator        SelfCheckinMutator
	zalo           zalo.Sender
	templateID     string
	enabled        EnabledChecker
	supportedCodes []string
	clk            clock.Clock
	logger         *slog.Logger
}

// NewSelfCheckinService constructs the service. enabled may be nil
// (always-on, dev). supportedCodes gates which project codes the flow may
// touch; templateID is the ZNS template (the bootstrap layer falls back to the
// reset template when SELF_CHECKIN_TEMPLATE_ID is unset).
func NewSelfCheckinService(
	store ResetStore,
	userRepo userRepo,
	employeeRepo employeeRepo,
	projectRepo projectRepo,
	assignmentRepo assignmentRepo,
	payrateRepo payrateRepo,
	mutator SelfCheckinMutator,
	zaloSender zalo.Sender,
	templateID string,
	enabled EnabledChecker,
	supportedCodes []string,
	clk clock.Clock,
	logger *slog.Logger,
) *SelfCheckinService {
	if clk == nil {
		clk = clock.New()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SelfCheckinService{
		store:          store,
		userRepo:       userRepo,
		employeeRepo:   employeeRepo,
		projectRepo:    projectRepo,
		assignmentRepo: assignmentRepo,
		payrateRepo:    payrateRepo,
		mutator:        mutator,
		zalo:           zaloSender,
		templateID:     templateID,
		enabled:        enabled,
		supportedCodes: supportedCodes,
		clk:            clk,
		logger:         logger,
	}
}

// SelfCheckinOTP resolves the phone to an account and synchronously dispatches
// a Zalo ZNS OTP, reporting explicitly whether the account existed and whether
// delivery succeeded. On success the response also carries the employee's
// active assignments on supported projects (check-in state, pending change,
// geofence target, shift windows) so the conversation needs one round-trip.
func (s *SelfCheckinService) SelfCheckinOTP(ctx context.Context, rawPhone string) (*SelfCheckinRequestResult, error) {
	recipient, err := phone.NormalizeVietnameseMobile(rawPhone)
	if err != nil {
		return nil, domain.NewValidationError(constants.MsgIntegrationPhoneInvalidVN)
	}

	// Hot-check the admin toggle after the phone check.
	if s.enabled != nil {
		if on, _ := s.enabled.IsEnabled(ctx); !on {
			return &SelfCheckinRequestResult{Found: true, FailureReason: FailureZaloDisabled}, nil
		}
	}

	u, emp, err := resolveAccountFor(ctx, s.userRepo, s.employeeRepo, rawPhone)
	if err != nil {
		// Not found, or the number is carried by several employees (conflict).
		return &SelfCheckinRequestResult{Found: false, FailureReason: FailureAccountNotFound}, nil
	}

	code, err := otp.GenerateCode()
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgIntegrationOTPSendFailedVN, err)
	}

	trackingID := fmt.Sprintf("integ_selfcheckin_%d_%d", u.ID, s.clk.Now().UnixNano())
	res, sendErr := s.zalo.Send(ctx, recipient, s.templateID, trackingID, map[string]string{"otp": code})
	if sendErr != nil {
		s.logger.Error("self check-in: zns transport error", "error", sendErr, "user_id", u.ID)
		return &SelfCheckinRequestResult{Found: true, FailureReason: FailureDeliveryFailed}, nil
	}
	if res.ErrorCode != 0 {
		s.logger.Info("self check-in: zns business error",
			"user_id", u.ID, "error_code", res.ErrorCode, "error_msg", res.ErrorMsg)
		return &SelfCheckinRequestResult{
			Found:             true,
			FailureReason:     FailureDeliveryFailed,
			DeliveryErrorCode: res.ErrorCode,
		}, nil
	}

	// Deliverable code → create the session so the employee can verify it.
	sessionID, err := s.store.Create(ctx, u.ID, hashHex(otp.HashCode(code)))
	if err != nil {
		s.logger.Error("self check-in: store create failed", "error", err, "user_id", u.ID)
		return nil, domain.NewInternalError(constants.MsgIntegrationOTPSendFailedVN, err)
	}

	return &SelfCheckinRequestResult{
		Found:        true,
		OTPSent:      true,
		SessionID:    sessionID,
		ExpiresIn:    int(s.store.TTL().Seconds()),
		EmployeeName: displayName(u, emp),
		Assignments:  s.assignmentsForEmployee(ctx, u.ID),
	}, nil
}

// SelfCheckinVerify consumes the OTP session (single use) and, on success,
// mints a single-use action token bound to the account. A wrong code leaves
// the session intact for retry within its TTL.
func (s *SelfCheckinService) SelfCheckinVerify(ctx context.Context, sessionID, code string) (*SelfCheckinVerifyResult, error) {
	userID, err := s.store.Consume(ctx, sessionID, hashHex(otp.HashCode(code)))
	if err != nil {
		return nil, s.mapStoreError(err)
	}

	token, err := s.store.CreateVerified(ctx, userID)
	if err != nil {
		return nil, domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	}

	return &SelfCheckinVerifyResult{
		ActionToken: token,
		ExpiresIn:   int(cache.DefaultZaloResetVerifiedTTL.Seconds()),
	}, nil
}

// SelfCheckinUpdate consumes a verified action token (single use) and applies
// the enable/disable through the shared, row-locked chatbot service method.
// The supported-project gate runs before the token is consumed so an expected
// caller mistake (unsupported project) doesn't burn the verification.
func (s *SelfCheckinService) SelfCheckinUpdate(ctx context.Context, actionToken string, projectID uint, enable bool, actorID uint) (*SelfCheckinUpdateResult, error) {
	p, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil || p == nil || !s.isSupportedCode(p.Code) {
		if err != nil {
			s.logger.Warn("self check-in: project lookup failed", "project_id", projectID, "error", err)
		}
		return nil, domain.NewValidationError(constants.MsgSelfCheckinProjectNotSupportedVN)
	}

	userID, err := s.store.ConsumeVerified(ctx, actionToken)
	if err != nil {
		return nil, s.mapStoreError(err)
	}

	emp, err := s.employeeRepo.GetByUserID(ctx, userID)
	if err != nil || emp == nil {
		// The token pointed at an account whose employee record is gone —
		// treat it like the deleted-account case in ResetPassword: invalid.
		return nil, domain.NewUnauthorizedError(constants.MsgSelfCheckinTokenInvalidVN)
	}

	r, err := s.mutator.SetSelfCheckinViaChatbot(ctx, projectID, emp.ID, enable, actorID)
	if err != nil {
		// Validation errors (already enabled, payrate missing, quota conflict)
		// surface as-is so the bot can relay the exact Vietnamese reason.
		return nil, err
	}

	return &SelfCheckinUpdateResult{
		Kind:                   r.Kind,
		Immediate:              r.Immediate,
		EffectiveFrom:          r.EffectiveFrom,
		CancelledPendingEnable: r.CancelledPendingEnable,
	}, nil
}

// assignmentsForEmployee lists the employee's active assignments restricted to
// projects whose code is enabled for the chatbot flow. Errors are logged and
// degrade to an empty list: the assignments are advisory context for the bot,
// never a reason to fail an OTP that was already delivered.
func (s *SelfCheckinService) assignmentsForEmployee(ctx context.Context, userID uint) []SelfCheckinAssignment {
	supported := s.supportedProjectIDs(ctx)
	if len(supported) == 0 {
		return nil
	}

	emp, err := s.employeeRepo.GetByUserID(ctx, userID)
	if err != nil || emp == nil {
		// A resolved account without an employee row has no assignments.
		return nil
	}
	rows, err := s.assignmentRepo.GetByEmployee(ctx, emp.ID)
	if err != nil {
		s.logger.Warn("self check-in: list assignments failed", "employee_id", emp.ID, "error", err)
		return nil
	}

	now := s.clk.Now()
	out := make([]SelfCheckinAssignment, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.LastDate != nil {
			continue // ended assignment — not active
		}
		if _, ok := supported[row.ProjectID]; !ok {
			continue // project code not enabled for the chatbot flow
		}
		item := SelfCheckinAssignment{
			ProjectID:        row.ProjectID,
			CheckInEnabled:   row.CheckInEnabled,
			CheckInStartDate: row.CheckInStartDate,
			PaymentSchedule:  row.PaymentSchedule,
		}
		if row.HasPendingCheckInChange() && row.CheckInEffectiveFrom != nil {
			item.PendingChange = &SelfCheckinPendingChange{
				Type:          "disable",
				EffectiveFrom: *row.CheckInEffectiveFrom,
			}
			if row.HasPendingCheckInEnable() {
				item.PendingChange.Type = "enable"
			}
		}
		p, err := s.projectRepo.GetByID(ctx, row.ProjectID)
		if err != nil || p == nil {
			s.logger.Warn("self check-in: project lookup failed", "project_id", row.ProjectID, "error", err)
			continue
		}
		item.ProjectName = p.Name
		item.Target = &SelfCheckinTarget{
			RadiusMeters: p.GeofenceRadiusMeters,
			Gates:        p.GeofenceGates,
		}
		item.ShiftWindows = s.shiftWindows(ctx, p, row.Position, now)
		out = append(out, item)
	}
	return out
}

// supportedProjectIDs resolves the configured project codes to IDs once per
// call. A code that no longer resolves (renamed/deleted) is skipped with a
// warning rather than failing every OTP.
func (s *SelfCheckinService) supportedProjectIDs(ctx context.Context) map[uint]struct{} {
	ids := make(map[uint]struct{}, len(s.supportedCodes))
	for _, code := range s.supportedCodes {
		p, err := s.projectRepo.GetByCode(ctx, code)
		if err != nil || p == nil {
			s.logger.Warn("self check-in: supported project code not resolvable", "code", code, "error", err)
			continue
		}
		ids[p.ID] = struct{}{}
	}
	return ids
}

// isSupportedCode reports whether a project code is enabled for the flow.
func (s *SelfCheckinService) isSupportedCode(code string) bool {
	for _, c := range s.supportedCodes {
		if strings.EqualFold(strings.TrimSpace(c), strings.TrimSpace(code)) {
			return true
		}
	}
	return false
}

// shiftWindows resolves the advisory shift windows for the assignment's
// position, mirroring the profile endpoint's populateShiftWindow: first
// (most recent) payrate, flattened config, admin display names from the
// project. Informational only — empty on any absence, never an error.
func (s *SelfCheckinService) shiftWindows(ctx context.Context, p *domain.Project, position string, now time.Time) []attendance.ShiftWindow {
	if s.payrateRepo == nil || position == "" {
		return nil
	}
	payrates, err := s.payrateRepo.GetByProject(ctx, p.ID)
	if err != nil || len(payrates) == 0 {
		return nil
	}
	flattened, err := payrates[0].Payrate.Flatten()
	if err != nil || len(flattened) == 0 {
		return nil
	}
	names := make(map[string]string, len(p.ShiftNames))
	for _, n := range p.ShiftNames {
		names[n.Range] = n.Name
	}
	return attendance.ResolveAllShiftWindows(flattened, position, now, names)
}

// resolveAccountFor maps a typed number to exactly one account via the shared
// identity resolver — the same lookup order as login and the self-service
// reset (users.mobile for admin/partner roles, then employees.mobile through
// employees.user_id). When the number is carried by a single employee record
// linked to that account, the employee row is returned too: it is the
// authoritative source for that person's name, so every endpoint reports the
// same identity.
func resolveAccountFor(ctx context.Context, users userRepo, employees employeeRepo, raw string) (*domain.User, *domain.Employee, error) {
	u, err := identity.New(users, employees).ResolveUserByMobile(ctx, raw)
	if err != nil {
		return nil, nil, err
	}
	if matches := identity.EmployeesByMobile(ctx, employees, raw); len(matches) == 1 {
		if emp := matches[0]; emp.UserID != nil && *emp.UserID == u.ID {
			return u, emp, nil
		}
	}
	return u, nil, nil
}

// mapStoreError translates OTP-session and action-token store errors into
// client-facing domain errors: a Redis outage is a 500 "try again"; a missing,
// expired, or consumed session/token is the same 401 (so a wrong code never
// reveals a valid session id).
func (s *SelfCheckinService) mapStoreError(err error) error {
	switch {
	case errors.Is(err, cache.ErrZaloResetStoreUnavailable):
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	case errors.Is(err, cache.ErrZaloResetSessionNotFound),
		errors.Is(err, cache.ErrZaloResetInvalidCode),
		errors.Is(err, cache.ErrZaloResetVerifiedNotFound):
		return domain.NewUnauthorizedError(constants.MsgSelfCheckinTokenInvalidVN)
	default:
		return domain.NewInternalError(constants.MsgZaloResetStoreDownVN, err)
	}
}
