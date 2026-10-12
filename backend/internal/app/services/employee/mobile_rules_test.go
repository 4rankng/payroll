package employee

import (
	"context"
	"testing"

	userservice "api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// mobileRulesRepo layers a controllable mobile lookup on the shared in-memory
// stub so the create/update mobile rules can be exercised in isolation.
type mobileRulesRepo struct {
	stubEmployeeRepo
	holdersByMobile map[string][]*domain.Employee
}

func (r *mobileRulesRepo) ListByMobile(_ context.Context, mobile string) ([]*domain.Employee, error) {
	return r.holdersByMobile[mobile], nil
}

// newMobileRulesService wires the same minimal harness the create tests use
// (create_after_commit_test.go): direct transactions, no-op events, in-memory
// user repo. required is the policy toggle; nil means off, matching runtime
// wiring where an unset settings row disables the policy.
func newMobileRulesService(repo *mobileRulesRepo, required func(context.Context) bool) *EmployeeService {
	users := &createEmployeeTestUserRepo{}
	events := createEmployeeTestEventBus{}
	return &EmployeeService{
		EmployeeRepo:             repo,
		UserRepo:                 users,
		UserService:              userservice.NewUserService(users, nil, events, "test-secret", "test-salt", nil),
		TransactionManager:       directTransactionManager{},
		events:                   events,
		isEmployeeMobileRequired: required,
	}
}

func mobileRulesValidEmployee() *domain.Employee {
	return &domain.Employee{Fullname: "Nguyen Van A", CCCD: "012345678901"}
}

func TestCreateEmployee_MobileRequiredToggle(t *testing.T) {
	t.Run("policy on rejects a missing mobile on manual and import paths", func(t *testing.T) {
		service := newMobileRulesService(&mobileRulesRepo{}, func(context.Context) bool { return true })

		_, err := service.CreateEmployee(context.Background(), mobileRulesValidEmployee(), 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileRequiredVN)

		// Imports share createEmployeeCore, so the toggle gates them too.
		_, err = service.CreateEmployeeFromImport(context.Background(), mobileRulesValidEmployee(), 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileRequiredVN)
	})

	t.Run("policy on accepts a create with a mobile", func(t *testing.T) {
		service := newMobileRulesService(&mobileRulesRepo{}, func(context.Context) bool { return true })

		emp := mobileRulesValidEmployee()
		emp.Mobile = "0912345678"
		created, err := service.CreateEmployee(context.Background(), emp, 9)
		require.NoError(t, err)
		require.Equal(t, "0912345678", created.Mobile)
	})

	t.Run("policy off accepts a missing mobile", func(t *testing.T) {
		service := newMobileRulesService(&mobileRulesRepo{}, nil)
		_, err := service.CreateEmployee(context.Background(), mobileRulesValidEmployee(), 9)
		require.NoError(t, err)
	})
}

func TestCreateEmployee_RejectsDuplicateMobile(t *testing.T) {
	repo := &mobileRulesRepo{holdersByMobile: map[string][]*domain.Employee{
		"0912345678": {{ID: 7, Fullname: "Existing Holder", CCCD: "001179010398"}},
	}}
	// nil toggle: the uniqueness rule is always on.
	service := newMobileRulesService(repo, nil)

	t.Run("explicit collision is a conflict regardless of the toggle", func(t *testing.T) {
		emp := &domain.Employee{Fullname: "Nguyen Van B", CCCD: "012345678902", Mobile: "0912345678"}
		_, err := service.CreateEmployee(context.Background(), emp, 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileTakenVN)
	})

	t.Run("outer spaces do not evade the lookup", func(t *testing.T) {
		emp := &domain.Employee{Fullname: "Nguyen Van B", CCCD: "012345678902", Mobile: " 0912345678 "}
		_, err := service.CreateEmployee(context.Background(), emp, 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileTakenVN)
	})

	t.Run("a free number creates", func(t *testing.T) {
		emp := &domain.Employee{Fullname: "Nguyen Van B", CCCD: "012345678902", Mobile: "0987654321"}
		_, err := service.CreateEmployee(context.Background(), emp, 9)
		require.NoError(t, err)
	})
}

func TestUpdateEmployee_MobileDuplicateSelfExclusion(t *testing.T) {
	base := func() *domain.Employee {
		return &domain.Employee{ID: 1, Fullname: "Nguyen Van A", CCCD: "012345678901", Mobile: "0911111111"}
	}

	t.Run("moving onto a number someone else holds is rejected", func(t *testing.T) {
		repo := &mobileRulesRepo{
			stubEmployeeRepo: stubEmployeeRepo{byID: map[uint]*domain.Employee{1: base()}},
			holdersByMobile: map[string][]*domain.Employee{
				"0922222222": {{ID: 2, Fullname: "Other", CCCD: "012345678902"}},
			},
		}
		service := newMobileRulesService(repo, nil)

		emp := base()
		emp.Mobile = "0922222222"
		err := service.UpdateEmployee(context.Background(), emp, 9)
		require.ErrorContains(t, err, constants.MsgEmployeeMobileTakenVN)
	})

	t.Run("re-saving the same number passes even when a legacy twin holds it", func(t *testing.T) {
		repo := &mobileRulesRepo{
			stubEmployeeRepo: stubEmployeeRepo{byID: map[uint]*domain.Employee{1: base()}},
			holdersByMobile: map[string][]*domain.Employee{
				"0911111111": {
					{ID: 1, Fullname: "Nguyen Van A", CCCD: "012345678901"},
					{ID: 3, Fullname: "Legacy Twin", CCCD: "012345678903"},
				},
			},
		}
		service := newMobileRulesService(repo, nil)

		require.NoError(t, service.UpdateEmployee(context.Background(), base(), 9))
		require.Equal(t, 1, repo.updateCalls)
	})

	t.Run("a number held only by the employee itself passes", func(t *testing.T) {
		repo := &mobileRulesRepo{
			stubEmployeeRepo: stubEmployeeRepo{byID: map[uint]*domain.Employee{1: base()}},
			holdersByMobile: map[string][]*domain.Employee{
				"0922222222": {{ID: 1, Fullname: "Nguyen Van A", CCCD: "012345678901"}},
			},
		}
		service := newMobileRulesService(repo, nil)

		emp := base()
		emp.Mobile = "0922222222"
		require.NoError(t, service.UpdateEmployee(context.Background(), emp, 9))
	})
}

// duplicateGroupsRepo serves the two reads the duplicate-mobile report needs.
type duplicateGroupsRepo struct {
	stubEmployeeRepo
	groups  []*domain.DuplicateIdentityGroup
	holders map[string][]*domain.Employee
}

func (r *duplicateGroupsRepo) GetDuplicateIdentities(context.Context) ([]*domain.DuplicateIdentityGroup, error) {
	return r.groups, nil
}

func (r *duplicateGroupsRepo) ListByMobiles(_ context.Context, mobiles []string) ([]*domain.Employee, error) {
	var out []*domain.Employee
	for _, mobile := range mobiles {
		out = append(out, r.holders[mobile]...)
	}
	return out, nil
}

func TestGetDuplicateMobileGroups_ScopesVisibleMembers(t *testing.T) {
	repo := &duplicateGroupsRepo{
		groups: []*domain.DuplicateIdentityGroup{
			{Field: "mobile", Value: "0912345678", EmployeeIDs: []uint{1, 2}, Count: 2},
			// A CCCD collision is not a mobile group — the nudge only reports numbers.
			{Field: "cccd", Value: "012345678901", EmployeeIDs: []uint{1, 3}, Count: 2},
		},
		holders: map[string][]*domain.Employee{
			"0912345678": {
				{ID: 1, Fullname: "Visible Holder", Mobile: "0912345678"},
				{ID: 2, Fullname: "Hidden Holder", Mobile: "0912345678"},
			},
		},
	}
	service := &EmployeeService{EmployeeRepo: repo}

	t.Run("unscoped sees every holder", func(t *testing.T) {
		groups, err := service.GetDuplicateMobileGroups(context.Background(), nil)
		require.NoError(t, err)
		require.Len(t, groups, 1)
		require.Equal(t, "0912345678", groups[0].Mobile)
		require.Len(t, groups[0].Employees, 2)
		require.Zero(t, groups[0].HiddenCount)
	})

	t.Run("scoped hides out-of-scope holders but keeps the group", func(t *testing.T) {
		groups, err := service.GetDuplicateMobileGroups(context.Background(), []uint{1})
		require.NoError(t, err)
		require.Len(t, groups, 1)
		require.Equal(t, []DuplicateMobileMember{{ID: 1, Fullname: "Visible Holder"}}, groups[0].Employees)
		require.Equal(t, 1, groups[0].HiddenCount)
	})

	t.Run("a group with no visible member is dropped", func(t *testing.T) {
		groups, err := service.GetDuplicateMobileGroups(context.Background(), []uint{99})
		require.NoError(t, err)
		require.Empty(t, groups)
	})
}
