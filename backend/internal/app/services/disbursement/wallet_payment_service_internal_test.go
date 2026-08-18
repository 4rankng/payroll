package disbursement

import "testing"

func TestAccountCheckPatch_WaivesFeeOnlyWhenTransferWasNotAttempted(t *testing.T) {
	service := &WalletPaymentService{}

	rejected := service.accountCheckPatch(AccountCheckOutcome{Verified: false, FeeWaived: true})
	if rejected.Fee == nil || *rejected.Fee != 0 {
		t.Fatalf("rejected account-check fee = %v, want 0", rejected.Fee)
	}

	verified := service.accountCheckPatch(AccountCheckOutcome{Verified: true, AccountName: "NGUYEN VAN A"})
	if verified.Fee != nil {
		t.Fatalf("verified account check unexpectedly changed fee to %d", *verified.Fee)
	}
	if verified.RecipientName == nil || *verified.RecipientName != "NGUYEN VAN A" {
		t.Fatalf("verified account name = %v, want persisted bank-confirmed name", verified.RecipientName)
	}
}
