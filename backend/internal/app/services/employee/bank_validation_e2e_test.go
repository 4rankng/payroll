package employee

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	infrastructure "api-server/internal/domain/ports/infrastructure"
	"api-server/internal/infra/disbursement/onepay"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManualEdit_NameMismatch_EndToEnd proves the full chain that fires
// when an admin manually edits an employee's bank account holder name to a
// WRONG value against a (mocked) OnePay that returns a different holder.
//
// This is the scenario the user tested in prod and saw no warning for. The
// test pins down what the BACKEND returns so we can confirm the frontend
// warning condition (bank_account_status === "invalid") is met.
//
// Chain exercised:
//   mock OnePay /customers (returns holder_name="PHAM THI THUY HANG")
//   → onepay.Provider.CheckAccount (detects mismatch, returns Valid=false)
//   → BankAccountValidator.Validate (maps to status=invalid + VN reason)
//   → applyBankAccountValidation (writes the 3 fields on the entity)
//
// The mock returns a CONFLICTING holder name (not empty) so the provider's
// name-matching branch fires — unlike the local sandbox which returns "" and
// skips matching.
func TestManualEdit_NameMismatch_EndToEnd(t *testing.T) {
	// Mock OnePay server: returns a fixed holder name that will mismatch
	// whatever the employee sends.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a SPECIFIC holder name — this is what triggers the mismatch
		// path in Provider.CheckAccount (empty holder_name skips matching).
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response_code":  "00",
			"message":        "SUCCESSFUL",
			"state":          "approved",
			"account_number": "0984218060",
			"holder_name":    "PHAM THI THUY HANG", // ← the "real" holder
			"swift_code":     "MSCBVNVX",
		})
	}))
	defer srv.Close()

	// Build a real OnePay provider pointed at the mock.
	opClient, err := onepay.NewClient(onepay.Config{
		PartnerID: "TESTPARTNER", PartnerKey: "TESTPARTNERKEY", AccountID: "ACC01",
		Endpoint: srv.URL,
	}, nil)
	require.NoError(t, err)
	provider := onepay.NewProvider(opClient, nil)

	// Register it in a disbursement registry so the validator can resolve it.
	registry := disbursement.NewRegistry(provider)
	active, err := registry.Active(context.Background())
	require.NoError(t, err)
	_, isVerifier := active.(infrastructure.AccountVerifier)
	require.True(t, isVerifier, "provider must implement AccountVerifier")

	// Bank with a SWIFT code (required for the validator to proceed).
	bank := &domain.Bank{ID: 1, BranchName: "MB Bank", SwiftCode: "MSCBVNVX"}
	bankRepo := &stubBankRepo{byID: map[uint]*domain.Bank{1: bank}}

	// Validator with NO cache (isolates the OnePay path).
	validator := NewBankAccountValidator(registry, bankRepo, nil, nil)

	// Simulate the manual edit: admin sets a WRONG holder name.
	employee := &domain.Employee{
		ID:                  1175,
		BankID:              &bank.ID,
		BankAccountNumber:   "0984218060",
		BankAccountName:     "WRONG NAME HERE", // ← intentionally mismatched
		BankAccountStatus:   domain.BankAccountStatusValid,
	}

	// This is exactly what the UpdateEmployee service path calls.
	applyBankAccountValidation(context.Background(), validator, employee)

	// ── Assertions: what the backend persists + returns to the frontend ──

	// 1. Status must be "invalid" (the frontend warning condition).
	assert.Equal(t, domain.BankAccountStatusInvalid, employee.BankAccountStatus,
		"mismatched name must flag the account invalid")

	// 2. Reason must be present, Vietnamese, and mention the mismatch.
	require.NotNil(t, employee.BankAccountInvalidReason, "reason must be set")
	t.Logf("reason returned: %q", *employee.BankAccountInvalidReason)
	assert.Contains(t, *employee.BankAccountInvalidReason, "không khớp",
		"reason must mention the mismatch in Vietnamese")

	// 3. Validated-at timestamp must be set.
	require.NotNil(t, employee.BankAccountValidatedAt, "validated_at must be set")

	// ── Simulate the JSON the frontend would receive ──
	// (This is what buildEmployeeResponse serializes.)
	dto := struct {
		BankAccountStatus        string  `json:"bank_account_status"`
		BankAccountInvalidReason *string `json:"bank_account_invalid_reason,omitempty"`
	}{
		BankAccountStatus:        employee.BankAccountStatus,
		BankAccountInvalidReason: employee.BankAccountInvalidReason,
	}
	payload, err := json.Marshal(dto)
	require.NoError(t, err)
	t.Logf("frontend would receive: %s", payload)

	// The frontend warning condition: bank_account_status === "invalid".
	assert.Contains(t, string(payload), `"bank_account_status":"invalid"`,
		"the JSON response must carry bank_account_status=invalid for the frontend warning to fire")
}

// TestManualEdit_NameMatches_EndToEnd is the control: when the admin enters
// the CORRECT name (matching what OnePay returns), the account stays valid
// and no warning fires.
func TestManualEdit_NameMatches_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response_code":  "00",
			"state":          "approved",
			"account_number": "0984218060",
			"holder_name":    "PHAM THI THUY HANG",
			"swift_code":     "MSCBVNVX",
		})
	}))
	defer srv.Close()

	opClient, err := onepay.NewClient(onepay.Config{
		PartnerID: "TESTPARTNER", PartnerKey: "TESTPARTNERKEY", AccountID: "ACC01",
		Endpoint: srv.URL,
	}, nil)
	require.NoError(t, err)
	provider := onepay.NewProvider(opClient, nil)
	registry := disbursement.NewRegistry(provider)
	bank := &domain.Bank{ID: 1, BranchName: "MB Bank", SwiftCode: "MSCBVNVX"}
	bankRepo := &stubBankRepo{byID: map[uint]*domain.Bank{1: bank}}
	validator := NewBankAccountValidator(registry, bankRepo, nil, nil)

	// Correct name: "PHAM THI THUY HANG" matches after normalization.
	employee := &domain.Employee{
		ID:                1175,
		BankID:            &bank.ID,
		BankAccountNumber: "0984218060",
		BankAccountName:   "Pham Thi Thuy Hang",
	}
	applyBankAccountValidation(context.Background(), validator, employee)

	assert.Equal(t, domain.BankAccountStatusValid, employee.BankAccountStatus,
		"matching name must keep the account valid")
	assert.Nil(t, employee.BankAccountInvalidReason,
		"no reason when valid")
}
