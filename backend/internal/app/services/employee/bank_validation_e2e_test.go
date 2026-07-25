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

// TestManualEdit_NameMismatch_EndToEnd proves that a manual edit with a wrong
// account holder name is rejected before persistence.
//
// Chain exercised:
//
//	mock OnePay /customers (returns holder_name="PHAM THI THUY HANG")
//	→ onepay.Provider.CheckAccount (detects mismatch, returns Valid=false)
//	→ BankAccountValidator.Validate (maps to status=invalid + VN reason)
//	→ validateManualBankAccount (returns a typed validation error)
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
		ID:                1175,
		BankID:            &bank.ID,
		BankAccountNumber: "0984218060",
		BankAccountName:   "WRONG NAME HERE", // ← intentionally mismatched
		BankAccountStatus: domain.BankAccountStatusValid,
	}

	err = validateManualBankAccount(context.Background(), validator, employee)

	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, bankAccountInvalidErrorCode, domainErr.Code)
	assert.Contains(t, domainErr.Message, "không khớp")
	assert.Contains(t, domainErr.Message, "PHAM THI THUY HANG")
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
	err = validateManualBankAccount(context.Background(), validator, employee)

	require.NoError(t, err)
	assert.Equal(t, domain.BankAccountStatusValid, employee.BankAccountStatus,
		"matching name must allow the edit")
	assert.Nil(t, employee.BankAccountInvalidReason,
		"no reason when valid")
}
