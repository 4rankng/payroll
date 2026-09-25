package advance_payment

import (
	"testing"

	"api-server/internal/domain"
)

func seededEmployee() *domain.Employee {
	oldBankID := uint(4499)
	return &domain.Employee{
		BankID:                   &oldBankID,
		BankAccountNumber:        "0000000000",
		BankAccountName:          "NGUYEN VAN NGUYEN",
		BankAccountStatus:        domain.BankAccountStatusValid,
		BankAccountInvalidReason: nil,
	}
}

// TestApplyImportBankUpdate_NoChangeSkipsProvider pins the fast path: a row
// identical to the stored record must not call the verifier at all (most
// rows of a monthly re-upload are unchanged) and must not restamp statuses.
func TestApplyImportBankUpdate_NoChangeSkipsProvider(t *testing.T) {
	emp := seededEmployee()
	bankID := emp.BankID
	calls := 0

	updated := applyImportBankUpdate(emp, bankID, "0000000000", "NGUYEN VAN NGUYEN",
		func(*uint, string, string) (string, *string, bool) {
			calls++
			return domain.BankAccountStatusValid, nil, true
		})

	if calls != 0 {
		t.Fatalf("provider called %d times for an unchanged tuple", calls)
	}
	if updated {
		t.Fatal("unchanged tuple must not mark the employee updated")
	}
	if emp.BankAccountValidatedAt != nil {
		t.Fatal("unchanged tuple must not restamp validated_at")
	}
}

// TestApplyImportBankUpdate_InvalidVerdictKeepsStoredTuple is the core
// contract: when OnePay confirms the workbook's bank details are wrong, the
// stored working tuple stays untouched and the invalid verdict is stamped so
// the row surfaces in the warning list.
func TestApplyImportBankUpdate_InvalidVerdictKeepsStoredTuple(t *testing.T) {
	emp := seededEmployee()
	newBankID := uint(4488)
	reason := "Số tài khoản không hợp lệ"

	updated := applyImportBankUpdate(emp, &newBankID, "0347921581", "NGUYEN VAN NGUYEN",
		func(*uint, string, string) (string, *string, bool) {
			return domain.BankAccountStatusInvalid, &reason, false
		})

	if !updated {
		t.Fatal("the invalid verdict must still count as an update (name/mobile/status flow)")
	}
	if emp.BankAccountNumber != "0000000000" || emp.BankAccountName != "NGUYEN VAN NGUYEN" {
		t.Fatalf("stored tuple was overwritten: %q / %q", emp.BankAccountNumber, emp.BankAccountName)
	}
	if emp.BankID == nil || *emp.BankID != 4499 {
		t.Fatalf("stored bank was overwritten: %v", emp.BankID)
	}
	if emp.BankAccountStatus != domain.BankAccountStatusInvalid {
		t.Fatalf("status = %q, want invalid", emp.BankAccountStatus)
	}
	if emp.BankAccountInvalidReason == nil || *emp.BankAccountInvalidReason != reason {
		t.Fatalf("invalid reason not stamped: %v", emp.BankAccountInvalidReason)
	}
	if emp.BankAccountValidatedAt == nil {
		t.Fatal("validated_at not stamped")
	}
}

// TestApplyImportBankUpdate_ValidVerdictWritesTuple covers the happy path.
func TestApplyImportBankUpdate_ValidVerdictWritesTuple(t *testing.T) {
	emp := seededEmployee()
	newBankID := uint(4488)

	updated := applyImportBankUpdate(emp, &newBankID, "0347921581", "NGUYEN VAN NGUYEN",
		func(*uint, string, string) (string, *string, bool) {
			return domain.BankAccountStatusValid, nil, true
		})

	if !updated {
		t.Fatal("changed tuple must mark the employee updated")
	}
	if emp.BankAccountNumber != "0347921581" || emp.BankAccountName != "NGUYEN VAN NGUYEN" {
		t.Fatalf("tuple not written: %q / %q", emp.BankAccountNumber, emp.BankAccountName)
	}
	if emp.BankID == nil || *emp.BankID != 4488 {
		t.Fatalf("bank not written: %v", emp.BankID)
	}
	if emp.BankAccountStatus != domain.BankAccountStatusValid {
		t.Fatalf("status = %q, want valid", emp.BankAccountStatus)
	}
}

// TestApplyImportBankUpdate_BlankNameCellPreservesStoredName combines the
// no-wipe rule with the verdict flow: an empty holder-name cell proposes the
// stored name, so a valid verdict updates number+bank while the name stays.
func TestApplyImportBankUpdate_BlankNameCellPreservesStoredName(t *testing.T) {
	emp := seededEmployee()
	newBankID := uint(4488)

	updated := applyImportBankUpdate(emp, &newBankID, "0347921581", "",
		func(bankID *uint, no, name string) (string, *string, bool) {
			if name != "NGUYEN VAN NGUYEN" {
				t.Fatalf("verifier should see the stored name for a blank cell, got %q", name)
			}
			return domain.BankAccountStatusValid, nil, true
		})

	if !updated {
		t.Fatal("expected update")
	}
	if emp.BankAccountName != "NGUYEN VAN NGUYEN" {
		t.Fatalf("stored holder name wiped: %q", emp.BankAccountName)
	}
	if emp.BankAccountNumber != "0347921581" {
		t.Fatalf("account number not updated: %q", emp.BankAccountNumber)
	}
}

// TestApplyImportBankUpdate_InvalidVerdictOnBlankNameKeepsEverything covers
// the combined case: wrong account number AND blank holder name — nothing
// stored may change except the invalid stamp.
func TestApplyImportBankUpdate_InvalidVerdictOnBlankNameKeepsEverything(t *testing.T) {
	emp := seededEmployee()

	updated := applyImportBankUpdate(emp, nil, "9999999999", "",
		func(*uint, string, string) (string, *string, bool) {
			return domain.BankAccountStatusInvalid, nil, false
		})

	if !updated {
		t.Fatal("expected the verdict stamp to count as an update")
	}
	if emp.BankAccountNumber != "0000000000" || emp.BankAccountName != "NGUYEN VAN NGUYEN" || emp.BankID == nil || *emp.BankID != 4499 {
		t.Fatalf("stored tuple changed: %+v", emp)
	}
	if emp.BankAccountStatus != domain.BankAccountStatusInvalid {
		t.Fatalf("status = %q, want invalid", emp.BankAccountStatus)
	}
}

// TestApplyImportBankUpdate_DisabledValidationWritesWithoutStamp covers the
// nil-validator deployment: CheckBankUpdate short-circuits (no provider call,
// empty status), tuples flow, and the status fields are left untouched.
func TestApplyImportBankUpdate_DisabledValidationWritesWithoutStamp(t *testing.T) {
	emp := seededEmployee()
	newBankID := uint(4488)

	// Mirrors EmployeeService.CheckBankUpdate with a nil validator: the
	// provider is never reached and the verdict carries no status.
	disabledCheck := func(*uint, string, string) (string, *string, bool) {
		return "", nil, true
	}

	updated := applyImportBankUpdate(emp, &newBankID, "0347921581", "NGUYEN VAN NGUYEN", disabledCheck)

	if !updated {
		t.Fatal("expected update")
	}
	if emp.BankAccountNumber != "0347921581" {
		t.Fatalf("tuple not written: %q", emp.BankAccountNumber)
	}
	if emp.BankAccountStatus != domain.BankAccountStatusValid {
		t.Fatalf("existing status must be untouched, got %q", emp.BankAccountStatus)
	}
	if emp.BankAccountValidatedAt != nil {
		t.Fatal("validated_at must not be stamped when validation is disabled")
	}
}

// TestApplyImportBankUpdate_UnverifiedFailsOpen covers the provider-outage
// path: the tuple is written and the unverified verdict is stamped (fail-open
// matches UpdateBankInfo semantics — a stale verdict is worse than none).
func TestApplyImportBankUpdate_UnverifiedFailsOpen(t *testing.T) {
	emp := seededEmployee()

	updated := applyImportBankUpdate(emp, emp.BankID, "0347921581", "NGUYEN VAN NGUYEN",
		func(*uint, string, string) (string, *string, bool) {
			return domain.BankAccountStatusUnverified, nil, true
		})

	if !updated {
		t.Fatal("expected update")
	}
	if emp.BankAccountNumber != "0347921581" {
		t.Fatalf("tuple not written: %q", emp.BankAccountNumber)
	}
	if emp.BankAccountStatus != domain.BankAccountStatusUnverified {
		t.Fatalf("status = %q, want unverified", emp.BankAccountStatus)
	}
}

// TestBankTupleChanged covers the comparison helper's nil/pointer edges.
func TestBankTupleChanged(t *testing.T) {
	emp := seededEmployee()
	newBankID := uint(4488)

	if bankTupleChanged(emp, emp.BankID, "0000000000", "NGUYEN VAN NGUYEN") {
		t.Fatal("identical tuple reported as changed")
	}
	if !bankTupleChanged(emp, &newBankID, "0000000000", "NGUYEN VAN NGUYEN") {
		t.Fatal("bank change not detected")
	}
	if !bankTupleChanged(emp, nil, "0000000000", "NGUYEN VAN NGUYEN") {
		t.Fatal("bank cleared not detected")
	}
	if !bankTupleChanged(emp, emp.BankID, "1111111111", "NGUYEN VAN NGUYEN") {
		t.Fatal("number change not detected")
	}
	if !bankTupleChanged(emp, emp.BankID, "0000000000", "OTHER NAME") {
		t.Fatal("name change not detected")
	}
}
