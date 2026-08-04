package domain

import "testing"

func TestCompareUsersIncludesMobile(t *testing.T) {
	beforeMobile := "0901234567"
	afterMobile := "0912345678"
	original := &User{Mobile: &beforeMobile}
	updated := *original
	updated.Mobile = &afterMobile

	changes := CompareUsers(original, &updated)
	change, ok := changes["mobile"]
	if !ok {
		t.Fatalf("expected mobile in user audit changes: %#v", changes)
	}
	if change.Before != beforeMobile || change.After != afterMobile {
		t.Fatalf("unexpected mobile audit change: %#v", change)
	}
}

func TestCompareLoans_IncludesMutableFinancialAndDescriptiveFields(t *testing.T) {
	beforeDescription := "Khoản vay ban đầu"
	afterDescription := "Khoản vay đã cập nhật"
	original := &Loan{
		Description:          &beforeDescription,
		OutstandingPrincipal: 10_000_000,
		TotalInterestPaid:    100_000,
	}
	updated := *original
	updated.Description = &afterDescription
	updated.OutstandingPrincipal = 8_000_000
	updated.TotalInterestPaid = 300_000

	changes := CompareLoans(original, &updated)
	for _, field := range []string{"description", "outstanding_principal", "total_interest_paid"} {
		if _, ok := changes[field]; !ok {
			t.Errorf("expected %q in loan audit changes: %#v", field, changes)
		}
	}
}

func TestCompareTransactions_IncludesEvidenceFields(t *testing.T) {
	beforeAssetID := uint(7)
	afterAssetID := uint(8)
	original := &Transaction{URL: "before.pdf", AssetID: &beforeAssetID}
	updated := *original
	updated.URL = "after.pdf"
	updated.AssetID = &afterAssetID

	changes := CompareTransactions(original, &updated)
	for _, field := range []string{"url", "asset_id"} {
		if _, ok := changes[field]; !ok {
			t.Errorf("expected %q in transaction audit changes: %#v", field, changes)
		}
	}
}
