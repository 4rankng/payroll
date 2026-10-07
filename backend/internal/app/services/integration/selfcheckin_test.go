package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
	"api-server/internal/pkg/clock"
)

// --- self check-in fakes -----------------------------------------------------

type fakeProjectRepo struct {
	byID   map[uint]*domain.Project
	byCode map[string]*domain.Project
}

func (r *fakeProjectRepo) GetByID(_ context.Context, id uint) (*domain.Project, error) {
	if p, ok := r.byID[id]; ok {
		return p, nil
	}
	return nil, domain.NewNotFoundError("project not found")
}

func (r *fakeProjectRepo) GetByCode(_ context.Context, code string) (*domain.Project, error) {
	if p, ok := r.byCode[code]; ok {
		return p, nil
	}
	return nil, domain.NewNotFoundError("project not found")
}

type fakeAssignmentRepo struct {
	byEmployee map[uint][]*domain.ProjectEmployee
}

func (r *fakeAssignmentRepo) GetByEmployee(_ context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	return r.byEmployee[employeeID], nil
}

type fakePayrateRepo struct {
	byProject map[uint][]*domain.Payrate
}

func (r *fakePayrateRepo) GetByProject(_ context.Context, projectID uint) ([]*domain.Payrate, error) {
	return r.byProject[projectID], nil
}

type fakeMutator struct {
	result        *project.SelfCheckinChatbotResult
	err           error
	calls         int
	lastProjectID uint
	lastEmpID     uint
	lastEnable    bool
	lastUpdatedBy uint
}

func (f *fakeMutator) SetSelfCheckinViaChatbot(_ context.Context, projectID, employeeID uint, enable bool, updatedBy uint) (*project.SelfCheckinChatbotResult, error) {
	f.calls++
	f.lastProjectID = projectID
	f.lastEmpID = employeeID
	f.lastEnable = enable
	f.lastUpdatedBy = updatedBy
	return f.result, f.err
}

// --- fixture -----------------------------------------------------------------

// lgdShiftWindow is the expected advisory window for the fixture payrate
// (08:00-17:00) at fixtureNow (10:00): check-in ±1h around 08:00, checkout
// [end-1h, end+4h].
var (
	lgdPayrateJSON = domain.PayrateConfiguration(`{"Lao công":{"weekday":{"08:00-17:00":30000}}}`)
	wantWindowName = "Ca ngày"
)

func newSelfCheckinFixture(t *testing.T, sender zalo.Sender, enabled EnabledChecker) (*SelfCheckinService, *fakeStore, *fakeMutator, *fakeAssignmentRepo) {
	t.Helper()
	store := newFakeStore()
	users := newFakeUserRepo()
	emp := &fakeEmployeeRepo{byMobile: map[string][]*domain.Employee{}}

	u := &domain.User{
		ID:       1,
		Username: "hoanvv",
		Fullname: "Vũ Văn Hoan",
		Role:     domain.RolePartner,
		Mobile:   ptr("0987654321"),
	}
	users.byMobile["0987654321"] = u
	users.byID[1] = u
	uid := u.ID
	emp.byMobile["0987654321"] = []*domain.Employee{{ID: 7, UserID: &uid, Fullname: "Vũ Văn Hoan"}}

	lgd := &domain.Project{
		ID:                   10,
		Name:                 "LGD",
		Code:                 "LGD",
		GeofenceRadiusMeters: 150,
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng 1", Lat: 10.1, Lng: 106.1}},
		ShiftNames:           []domain.ShiftName{{Range: "08:00-17:00", Name: wantWindowName}},
	}
	other := &domain.Project{ID: 20, Name: "Khác", Code: "ABC"}
	projects := &fakeProjectRepo{
		byID:   map[uint]*domain.Project{10: lgd, 20: other},
		byCode: map[string]*domain.Project{"LGD": lgd, "ABC": other},
	}

	assignments := &fakeAssignmentRepo{byEmployee: map[uint][]*domain.ProjectEmployee{
		7: {
			{ProjectID: 10, EmployeeID: 7, Position: "Lao công", PaymentSchedule: string(domain.PaymentScheduleFlexible), CheckInEnabled: true},
			{ProjectID: 20, EmployeeID: 7, Position: "Lao công", PaymentSchedule: string(domain.PaymentScheduleFlexible)},
		},
	}}

	payrates := &fakePayrateRepo{byProject: map[uint][]*domain.Payrate{
		10: {{ProjectID: 10, Payrate: lgdPayrateJSON}},
	}}

	mutator := &fakeMutator{}

	svc := NewSelfCheckinService(
		store, users, emp, projects, assignments, payrates, mutator, sender,
		"12345", enabled, []string{"LGD"}, clock.NewFake(fixtureNow), nil,
	)
	return svc, store, mutator, assignments
}

// selfCheckinVerifyHappy runs the OTP → verify steps and returns the minted
// action token.
func selfCheckinVerifyHappy(t *testing.T, svc *SelfCheckinService, sender *fakeSender) string {
	t.Helper()
	ctx := context.Background()
	res, err := svc.SelfCheckinOTP(ctx, "0987654321")
	if err != nil {
		t.Fatalf("SelfCheckinOTP: %v", err)
	}
	if !res.Found || !res.OTPSent {
		t.Fatalf("found=%v otp_sent=%v, want true/true", res.Found, res.OTPSent)
	}
	verify, err := svc.SelfCheckinVerify(ctx, res.SessionID, codeOf(t, sender))
	if err != nil {
		t.Fatalf("SelfCheckinVerify: %v", err)
	}
	if verify.ActionToken == "" || verify.ExpiresIn != 300 {
		t.Fatalf("verify = %+v", verify)
	}
	return verify.ActionToken
}

// --- tests -------------------------------------------------------------------

func TestSelfCheckinOTPHappyPathCarriesAssignments(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _, _ := newSelfCheckinFixture(t, sender, nil)
	ctx := context.Background()

	res, err := svc.SelfCheckinOTP(ctx, "0987654321")
	if err != nil {
		t.Fatalf("SelfCheckinOTP: %v", err)
	}
	if res.SessionID == "" || res.EmployeeName != "Vũ Văn Hoan" || res.ExpiresIn != 600 {
		t.Fatalf("session=%q name=%q expires=%d", res.SessionID, res.EmployeeName, res.ExpiresIn)
	}
	// Only the LGD assignment is eligible; the ABC one is filtered out.
	if len(res.Assignments) != 1 {
		t.Fatalf("assignments = %d, want 1 (LGD only)", len(res.Assignments))
	}
	a := res.Assignments[0]
	if a.ProjectID != 10 || a.ProjectName != "LGD" || !a.CheckInEnabled {
		t.Fatalf("assignment = %+v", a)
	}
	if a.PaymentSchedule != string(domain.PaymentScheduleFlexible) {
		t.Fatalf("payment_schedule = %q", a.PaymentSchedule)
	}
	if a.Target == nil || a.Target.RadiusMeters != 150 || len(a.Target.Gates) != 1 {
		t.Fatalf("target = %+v", a.Target)
	}
	if len(a.ShiftWindows) != 1 {
		t.Fatalf("shift_windows = %d, want 1", len(a.ShiftWindows))
	}
	w := a.ShiftWindows[0]
	if w.ShiftStart.Hour() != 8 || w.CheckInWindowStart.Hour() != 7 ||
		w.CheckInWindowEnd.Hour() != 9 || w.CheckOutWindowEnd.Hour() != 21 {
		t.Fatalf("window = %+v", w)
	}
	if w.Name != wantWindowName {
		t.Fatalf("shift name = %q, want %q", w.Name, wantWindowName)
	}
	// The template used is the self check-in one passed to the constructor;
	// the ZNS data payload carries only the OTP.
	if sender.lastData == nil || len(sender.lastData["otp"]) != 6 {
		t.Fatalf("sender data = %v", sender.lastData)
	}
}

func TestSelfCheckinOTPUnknownPhone(t *testing.T) {
	sender := &fakeSender{}
	svc, store, _, _ := newSelfCheckinFixture(t, sender, nil)

	res, err := svc.SelfCheckinOTP(context.Background(), "0900000000")
	if err != nil {
		t.Fatalf("unknown phone must not error: %v", err)
	}
	if res.Found || res.OTPSent || res.FailureReason != FailureAccountNotFound {
		t.Fatalf("res = %+v", res)
	}
	if sender.calls != 0 || len(store.sessions) != 0 {
		t.Fatalf("send calls=%d sessions=%d, want none", sender.calls, len(store.sessions))
	}
}

func TestSelfCheckinOTPDisabledToggle(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _, _ := newSelfCheckinFixture(t, sender, fakeEnabled{on: false})

	res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
	if err != nil {
		t.Fatalf("SelfCheckinOTP: %v", err)
	}
	if !res.Found || res.FailureReason != FailureZaloDisabled || sender.calls != 0 {
		t.Fatalf("res = %+v calls=%d", res, sender.calls)
	}
}

func TestSelfCheckinOTPDeliveryFailure(t *testing.T) {
	t.Run("transport error", func(t *testing.T) {
		sender := &fakeSender{err: context.DeadlineExceeded}
		svc, store, _, _ := newSelfCheckinFixture(t, sender, nil)

		res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
		if err != nil {
			t.Fatalf("SelfCheckinOTP: %v", err)
		}
		if !res.Found || res.FailureReason != FailureDeliveryFailed || len(store.sessions) != 0 {
			t.Fatalf("res = %+v sessions=%d", res, len(store.sessions))
		}
	})
	t.Run("zns business error reports the code", func(t *testing.T) {
		sender := &fakeSender{result: zalo.SendResult{ErrorCode: -118, ErrorMsg: "phone not linked"}}
		svc, store, _, _ := newSelfCheckinFixture(t, sender, nil)

		res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
		if err != nil {
			t.Fatalf("SelfCheckinOTP: %v", err)
		}
		if res.FailureReason != FailureDeliveryFailed || res.DeliveryErrorCode != -118 || len(store.sessions) != 0 {
			t.Fatalf("res = %+v", res)
		}
	})
}

func TestSelfCheckinOTPFiltersAssignmentsBySupportedCodes(t *testing.T) {
	sender := &fakeSender{}
	svc, _, _, assignments := newSelfCheckinFixture(t, sender, nil)
	ended := fixtureNow.AddDate(0, 0, -30)
	assignments.byEmployee[7] = append(assignments.byEmployee[7],
		&domain.ProjectEmployee{ProjectID: 20, EmployeeID: 7, LastDate: &ended},
	)

	res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
	if err != nil {
		t.Fatalf("SelfCheckinOTP: %v", err)
	}
	for _, a := range res.Assignments {
		if a.ProjectID == 20 {
			t.Errorf("unsupported project 20 leaked into assignments: %+v", a)
		}
	}
	if len(res.Assignments) != 1 {
		t.Fatalf("assignments = %d, want 1", len(res.Assignments))
	}
}

func TestSelfCheckinOTPPendingChangeRendering(t *testing.T) {
	effective := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	yes, no := true, false

	t.Run("pending enable", func(t *testing.T) {
		sender := &fakeSender{}
		svc, _, _, assignments := newSelfCheckinFixture(t, sender, nil)
		assignments.byEmployee[7] = []*domain.ProjectEmployee{{
			ProjectID: 10, EmployeeID: 7, Position: "Lao công",
			PendingCheckInEnabled: &yes, CheckInEffectiveFrom: &effective,
		}}

		res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
		if err != nil {
			t.Fatalf("SelfCheckinOTP: %v", err)
		}
		if len(res.Assignments) != 1 || res.Assignments[0].PendingChange == nil {
			t.Fatalf("assignments = %+v", res.Assignments)
		}
		pc := res.Assignments[0].PendingChange
		if pc.Type != "enable" || !pc.EffectiveFrom.Equal(effective) {
			t.Fatalf("pending change = %+v", pc)
		}
	})
	t.Run("pending disable", func(t *testing.T) {
		sender := &fakeSender{}
		svc, _, _, assignments := newSelfCheckinFixture(t, sender, nil)
		assignments.byEmployee[7] = []*domain.ProjectEmployee{{
			ProjectID: 10, EmployeeID: 7, Position: "Lao công", CheckInEnabled: true,
			PendingCheckInEnabled: &no, CheckInEffectiveFrom: &effective,
		}}

		res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
		if err != nil {
			t.Fatalf("SelfCheckinOTP: %v", err)
		}
		if len(res.Assignments) != 1 || res.Assignments[0].PendingChange == nil {
			t.Fatalf("assignments = %+v", res.Assignments)
		}
		pc := res.Assignments[0].PendingChange
		if pc.Type != "disable" || !pc.EffectiveFrom.Equal(effective) {
			t.Fatalf("pending change = %+v", pc)
		}
	})
	t.Run("start date passthrough", func(t *testing.T) {
		sender := &fakeSender{}
		svc, _, _, assignments := newSelfCheckinFixture(t, sender, nil)
		start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		assignments.byEmployee[7] = []*domain.ProjectEmployee{{
			ProjectID: 10, EmployeeID: 7, Position: "Lao công", CheckInEnabled: true,
			CheckInStartDate: &start,
		}}

		res, err := svc.SelfCheckinOTP(context.Background(), "0987654321")
		if err != nil {
			t.Fatalf("SelfCheckinOTP: %v", err)
		}
		if len(res.Assignments) != 1 || res.Assignments[0].CheckInStartDate == nil ||
			!res.Assignments[0].CheckInStartDate.Equal(start) {
			t.Fatalf("check_in_start_date = %+v", res.Assignments[0].CheckInStartDate)
		}
	})
}

func TestSelfCheckinVerifyUpdateLifecycle(t *testing.T) {
	sender := &fakeSender{}
	svc, store, mutator, _ := newSelfCheckinFixture(t, sender, nil)
	mutator.result = &project.SelfCheckinChatbotResult{
		Kind:          "enable",
		Immediate:     true,
		EffectiveFrom: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	token := selfCheckinVerifyHappy(t, svc, sender)
	if len(store.sessions) != 0 {
		t.Fatalf("session not consumed by verify")
	}

	res, err := svc.SelfCheckinUpdate(context.Background(), token, 10, true, 99)
	if err != nil {
		t.Fatalf("SelfCheckinUpdate: %v", err)
	}
	if !res.Immediate || res.Kind != "enable" || res.EffectiveFrom.IsZero() || res.CancelledPendingEnable {
		t.Fatalf("res = %+v", res)
	}
	if mutator.calls != 1 || mutator.lastProjectID != 10 || mutator.lastEmpID != 7 ||
		!mutator.lastEnable || mutator.lastUpdatedBy != 99 {
		t.Fatalf("mutator = %+v", mutator)
	}
}

func TestSelfCheckinUpdateUnsupportedProjectKeepsToken(t *testing.T) {
	sender := &fakeSender{}
	svc, store, mutator, _ := newSelfCheckinFixture(t, sender, nil)
	token := selfCheckinVerifyHappy(t, svc, sender)

	_, err := svc.SelfCheckinUpdate(context.Background(), token, 20, true, 99)
	if err == nil || err.Error() != constants.MsgSelfCheckinProjectNotSupportedVN {
		t.Fatalf("err = %v, want %q", err, constants.MsgSelfCheckinProjectNotSupportedVN)
	}
	if mutator.calls != 0 {
		t.Fatalf("mutator called %d times, want 0", mutator.calls)
	}
	// The gate runs before the consume: the verification is not burned on an
	// expected caller mistake, so the same token works for a supported project.
	if _, ok := store.verified[token]; !ok {
		t.Fatal("token was consumed by a rejected update")
	}
	svc.mutator = &fakeMutator{result: &project.SelfCheckinChatbotResult{Kind: "enable", Immediate: true, EffectiveFrom: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}}
	if _, err := svc.SelfCheckinUpdate(context.Background(), token, 10, true, 99); err != nil {
		t.Fatalf("update after rejected attempt: %v", err)
	}
}

func TestSelfCheckinUpdateUnknownProjectSameRejection(t *testing.T) {
	sender := &fakeSender{}
	svc, _, mutator, _ := newSelfCheckinFixture(t, sender, nil)
	token := selfCheckinVerifyHappy(t, svc, sender)

	// A non-existent project id must get the same rejection: no existence leak.
	_, err := svc.SelfCheckinUpdate(context.Background(), token, 999, true, 99)
	if err == nil || err.Error() != constants.MsgSelfCheckinProjectNotSupportedVN {
		t.Fatalf("err = %v, want %q", err, constants.MsgSelfCheckinProjectNotSupportedVN)
	}
	if mutator.calls != 0 {
		t.Fatalf("mutator called %d times, want 0", mutator.calls)
	}
}

func TestSelfCheckinUpdateValidationSurfaces(t *testing.T) {
	sender := &fakeSender{}
	svc, _, mutator, _ := newSelfCheckinFixture(t, sender, nil)
	wantErr := domain.NewValidationError(constants.MsgSelfCheckinAlreadyEnabledVN)
	mutator.result = &project.SelfCheckinChatbotResult{}
	mutator.err = wantErr
	token := selfCheckinVerifyHappy(t, svc, sender)

	_, err := svc.SelfCheckinUpdate(context.Background(), token, 10, true, 99)
	if err == nil || err.Error() != constants.MsgSelfCheckinAlreadyEnabledVN {
		t.Fatalf("err = %v, want validation %q", err, constants.MsgSelfCheckinAlreadyEnabledVN)
	}
	if !domain.IsValidationError(err) {
		t.Fatalf("err type = %T, want validation error", err)
	}
}

func TestSelfCheckinUpdateTokenSingleUse(t *testing.T) {
	sender := &fakeSender{}
	svc, _, mutator, _ := newSelfCheckinFixture(t, sender, nil)
	mutator.result = &project.SelfCheckinChatbotResult{Kind: "disable"}
	token := selfCheckinVerifyHappy(t, svc, sender)

	if _, err := svc.SelfCheckinUpdate(context.Background(), token, 10, false, 99); err != nil {
		t.Fatalf("first update: %v", err)
	}
	_, err := svc.SelfCheckinUpdate(context.Background(), token, 10, false, 99)
	if err == nil || !domain.IsUnauthorizedError(err) || !strings.Contains(err.Error(), constants.MsgSelfCheckinTokenInvalidVN) {
		t.Fatalf("replay err = %v, want unauthorized %q", err, constants.MsgSelfCheckinTokenInvalidVN)
	}
	if mutator.calls != 1 {
		t.Fatalf("mutator calls = %d, want 1", mutator.calls)
	}
}
