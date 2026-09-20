package identity

import (
	"context"
	"errors"
	"testing"

	"api-server/internal/domain"
)

// --- fakes -----------------------------------------------------------------

type fakeUsers struct {
	byMobile map[string]*domain.User
	byID     map[uint]*domain.User
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byMobile: map[string]*domain.User{}, byID: map[uint]*domain.User{}}
}

func (f *fakeUsers) GetByMobile(_ context.Context, mobile string) (*domain.User, error) {
	user, ok := f.byMobile[mobile]
	if !ok {
		return nil, domain.NewNotFoundError("user not found")
	}
	return user, nil
}

func (f *fakeUsers) GetByID(_ context.Context, id uint) (*domain.User, error) {
	user, ok := f.byID[id]
	if !ok {
		return nil, domain.NewNotFoundError("user not found")
	}
	return user, nil
}

type fakeEmployees struct {
	byMobile map[string][]*domain.Employee
	err      error // when set, every lookup fails
}

func newFakeEmployees() *fakeEmployees {
	return &fakeEmployees{byMobile: map[string][]*domain.Employee{}}
}

func (f *fakeEmployees) ListByMobile(_ context.Context, mobile string) ([]*domain.Employee, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byMobile[mobile], nil
}

// userWithMobile registers an account reachable through users.mobile.
func userWithMobile(repo *fakeUsers, id uint, role domain.UserRole, mobile string) *domain.User {
	u := &domain.User{ID: id, Username: "u" + mobile, Role: role, Mobile: &mobile}
	repo.byMobile[mobile] = u
	repo.byID[id] = u
	return u
}

// employeeWithUser registers an employee record linked to an account.
func employeeWithUser(repo *fakeUsers, employees *fakeEmployees, userID uint, mobile string) *domain.Employee {
	u := &domain.User{ID: userID, Username: "emp" + mobile, Role: domain.RoleEmployee}
	repo.byID[userID] = u
	emp := &domain.Employee{ID: userID, UserID: &u.ID, Mobile: mobile}
	employees.byMobile[mobile] = append(employees.byMobile[mobile], emp)
	return emp
}

// --- MobileCandidates ------------------------------------------------------

func TestMobileCandidates(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"canonical form is the only candidate", "0909123456", []string{"0909123456"}},
		{"country code falls back to the raw value", "+84909123456", []string{"0909123456", "+84909123456"}},
		{"84 prefix falls back to the raw value", "84909123456", []string{"0909123456", "84909123456"}},
		{"spaced input is normalized first", "0912 345 678", []string{"0912345678", "0912 345 678"}},
		{"unparseable input is only itself", "0000000000", []string{"0000000000"}},
		{"empty input yields no candidate", "", nil},
		{"blank input yields no candidate", "   ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MobileCandidates(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("MobileCandidates(%q) = %v, want %v", tt.raw, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("MobileCandidates(%q) = %v, want %v", tt.raw, got, tt.want)
				}
			}
		})
	}
}

// --- ResolveUserByMobile ---------------------------------------------------

func TestResolveUserByMobile_UsersMobileBeatsEmployeeRecord(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	admin := userWithMobile(users, 3, domain.RoleAdmin, "0912345678")
	// Same number also on an employee record pointing at a different account.
	employeeWithUser(users, employees, 77, "0912345678")

	for _, input := range []string{"0912345678", "+84912345678", "0912 345 678", " 0912345678 "} {
		got, err := New(users, employees).ResolveUserByMobile(context.Background(), input)
		if err != nil {
			t.Fatalf("ResolveUserByMobile(%q) error = %v", input, err)
		}
		if got.ID != admin.ID {
			t.Errorf("ResolveUserByMobile(%q) resolved user %d, want the users.mobile admin %d", input, got.ID, admin.ID)
		}
	}
}

func TestResolveUserByMobile_RolesWithUserRowMobile(t *testing.T) {
	for _, role := range []domain.UserRole{domain.RoleAdmin, domain.RolePartner, domain.RoleAdvPartner} {
		t.Run(string(role), func(t *testing.T) {
			users, employees := newFakeUsers(), newFakeEmployees()
			want := userWithMobile(users, 11, role, "0901234567")

			got, err := New(users, employees).ResolveUserByMobile(context.Background(), "+84901234567")
			if err != nil {
				t.Fatalf("ResolveUserByMobile error = %v", err)
			}
			if got.ID != want.ID {
				t.Errorf("resolved user %d, want %d", got.ID, want.ID)
			}
		})
	}
}

func TestResolveUserByMobile_EmployeeViaEmployeesMobile(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 5, "0987654321")

	for _, input := range []string{"0987654321", "+84987654321"} {
		got, err := New(users, employees).ResolveUserByMobile(context.Background(), input)
		if err != nil {
			t.Fatalf("ResolveUserByMobile(%q) error = %v", input, err)
		}
		if got.ID != 5 || got.Role != domain.RoleEmployee {
			t.Errorf("ResolveUserByMobile(%q) = user %d role %q, want employee 5", input, got.ID, got.Role)
		}
	}
}

func TestResolveUserByMobile_EmployeeNotInUsersMobile(t *testing.T) {
	// The employee number must be found through employees.mobile alone: the
	// users store holds no row for it (the documented rule).
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 9, "0900000001")
	if len(users.byMobile) != 0 {
		t.Fatal("test setup: employee number must not be in users.byMobile")
	}

	got, err := New(users, employees).ResolveUserByMobile(context.Background(), "0900000001")
	if err != nil {
		t.Fatalf("ResolveUserByMobile error = %v", err)
	}
	if got.ID != 9 {
		t.Errorf("resolved user %d, want 9", got.ID)
	}
}

func TestResolveUserByMobile_SharedByTwoEmployees_IsConflict(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 41, "0374990377")
	employeeWithUser(users, employees, 42, "0374990377")

	got, err := New(users, employees).ResolveUserByMobile(context.Background(), "0374990377")
	if err == nil {
		t.Fatalf("ResolveUserByMobile returned user %v, want a conflict error", got)
	}
	if !domain.IsConflictError(err) {
		t.Fatalf("error = %v, want a conflict error", err)
	}
	if got != nil {
		t.Errorf("user = %v, want nil alongside the conflict", got)
	}
}

func TestResolveUserByMobile_SharedNumberDoesNotFallBackToRawCandidate(t *testing.T) {
	// Canonical form carries two employees; only the raw form carries one. The
	// ambiguity must not be dodged by trying the next candidate.
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 41, "0987654321")
	employeeWithUser(users, employees, 42, "0987654321")
	employeeWithUser(users, employees, 43, "84987654321")

	_, err := New(users, employees).ResolveUserByMobile(context.Background(), "84987654321")
	if err == nil || !domain.IsConflictError(err) {
		t.Fatalf("error = %v, want a conflict error", err)
	}
}

func TestResolveUserByMobile_UnknownNumber_NotFound(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 5, "0987654321")

	for _, input := range []string{"0900000000", "+84900000000", "khong-phai-so"} {
		got, err := New(users, employees).ResolveUserByMobile(context.Background(), input)
		if err == nil {
			t.Fatalf("ResolveUserByMobile(%q) = %v, want not-found", input, got)
		}
		if !domain.IsNotFoundError(err) {
			t.Errorf("ResolveUserByMobile(%q) error = %v, want not-found", input, err)
		}
	}
}

func TestResolveUserByMobile_EmptyInput_NotFound(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	// A row with an empty mobile must never be reachable from an empty input.
	employees.byMobile[""] = []*domain.Employee{{ID: 8, Mobile: ""}}

	for _, input := range []string{"", "   "} {
		got, err := New(users, employees).ResolveUserByMobile(context.Background(), input)
		if err == nil || !domain.IsNotFoundError(err) {
			t.Fatalf("ResolveUserByMobile(%q) = %v, %v; want not-found", input, got, err)
		}
	}
}

func TestResolveUserByMobile_StoredValueWithCountryCode(t *testing.T) {
	// Some rows predate normalization: the raw candidate must still reach them.
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 6, "84909123456")

	got, err := New(users, employees).ResolveUserByMobile(context.Background(), "84909123456")
	if err != nil {
		t.Fatalf("ResolveUserByMobile error = %v", err)
	}
	if got.ID != 6 {
		t.Errorf("resolved user %d, want 6", got.ID)
	}
}

func TestResolveUserByMobile_EmployeeWithoutLinkedAccount_NotFound(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	employees.byMobile["0900000002"] = []*domain.Employee{{ID: 7, Mobile: "0900000002"}}

	got, err := New(users, employees).ResolveUserByMobile(context.Background(), "0900000002")
	if err == nil || !domain.IsNotFoundError(err) {
		t.Fatalf("ResolveUserByMobile = %v, %v; want not-found", got, err)
	}
}

func TestResolveUserByMobile_UserRowWithEmployeeRoleIsSkipped(t *testing.T) {
	// A legacy copy of an employee number inside users.mobile must not shadow
	// the employee record that owns it.
	users, employees := newFakeUsers(), newFakeEmployees()
	userWithMobile(users, 50, domain.RoleEmployee, "0900000003")
	employeeWithUser(users, employees, 51, "0900000003")

	got, err := New(users, employees).ResolveUserByMobile(context.Background(), "0900000003")
	if err != nil {
		t.Fatalf("ResolveUserByMobile error = %v", err)
	}
	if got.ID != 51 {
		t.Errorf("resolved user %d, want the employee record's account 51", got.ID)
	}
}

func TestResolveUserByMobile_EmployeeLookupErrorIsNotAMatch(t *testing.T) {
	users, employees := newFakeUsers(), newFakeEmployees()
	employeeWithUser(users, employees, 5, "0987654321")
	employees.err = errors.New("db down")

	got, err := New(users, employees).ResolveUserByMobile(context.Background(), "0987654321")
	if err == nil || !domain.IsNotFoundError(err) {
		t.Fatalf("ResolveUserByMobile = %v, %v; want not-found (fail closed)", got, err)
	}
}

func TestResolveUserByMobile_NilEmployeeStore(t *testing.T) {
	users := newFakeUsers()
	admin := userWithMobile(users, 3, domain.RoleAdmin, "0912345678")

	got, err := New(users, nil).ResolveUserByMobile(context.Background(), "0912345678")
	if err != nil || got.ID != admin.ID {
		t.Fatalf("ResolveUserByMobile = %v, %v; want the admin account", got, err)
	}
	if _, err := New(users, nil).ResolveUserByMobile(context.Background(), "0987654321"); err == nil {
		t.Error("an employee-only number must not resolve without the employees store")
	}
}

// --- EmployeesByMobile -----------------------------------------------------

func TestEmployeesByMobile_ReturnsEveryMatchOfTheFirstHit(t *testing.T) {
	employees := newFakeEmployees()
	employees.byMobile["0374990377"] = []*domain.Employee{{ID: 1}, {ID: 2}}

	got := EmployeesByMobile(context.Background(), employees, "+84374990377")
	if len(got) != 2 {
		t.Fatalf("EmployeesByMobile returned %d matches, want 2 (duplicates are kept)", len(got))
	}

	if got := EmployeesByMobile(context.Background(), employees, "0900000000"); got != nil {
		t.Errorf("EmployeesByMobile(unknown) = %v, want none", got)
	}
	if got := EmployeesByMobile(context.Background(), nil, "0374990377"); got != nil {
		t.Errorf("EmployeesByMobile(nil repo) = %v, want none", got)
	}
}
