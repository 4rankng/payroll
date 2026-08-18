package disbursement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	disbursementservice "api-server/internal/app/services/disbursement"
	"api-server/internal/domain"
	"api-server/internal/domain/ports/infrastructure"

	"github.com/gin-gonic/gin"
)

type employeeAccountLookupRepoStub struct {
	employee *domain.Employee
	err      error
}

func (s employeeAccountLookupRepoStub) GetByID(context.Context, uint) (*domain.Employee, error) {
	return s.employee, s.err
}

type employeeAccountVerifierStub struct {
	checkResult   *infrastructure.AccountCheckResult
	checkErr      error
	checkCalls    int
	transferCalls int
	lastRequest   infrastructure.AccountCheckRequest
}

type employeeAccountProviderWithoutVerifier struct{}

func (employeeAccountProviderWithoutVerifier) Name() string { return "unsupported" }
func (employeeAccountProviderWithoutVerifier) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	return nil, nil
}
func (employeeAccountProviderWithoutVerifier) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}

func (s *employeeAccountVerifierStub) Name() string { return "onepay" }

func (s *employeeAccountVerifierStub) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	s.transferCalls++
	return nil, nil
}

func (s *employeeAccountVerifierStub) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return nil, nil
}

func (s *employeeAccountVerifierStub) CheckAccount(_ context.Context, req infrastructure.AccountCheckRequest) (*infrastructure.AccountCheckResult, error) {
	s.checkCalls++
	s.lastRequest = req
	return s.checkResult, s.checkErr
}

func completeLookupEmployee() *domain.Employee {
	bankID := uint(9)
	return &domain.Employee{
		ID:                42,
		Fullname:          "Nguyễn Văn A",
		BankID:            &bankID,
		BankAccountNumber: "0123456789",
		BankAccountName:   "NGUYEN VAN A",
		Bank: &domain.Bank{
			ID:         bankID,
			BranchName: "Ngân hàng kiểm thử",
			BankCode:   "TEST",
			SwiftCode:  "TESTVNVX",
		},
	}
}

func performEmployeeAccountCheck(t *testing.T, handler *ManualDisbursementHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/employee-account-check", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.CheckEmployeeAccount(ctx)
	return recorder
}

func performCustomAccountCheck(t *testing.T, handler *ManualDisbursementHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/check-account", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.CheckAccount(ctx)
	return recorder
}

func TestCheckEmployeeAccountUsesPersistedBankDataWithoutTransfer(t *testing.T) {
	provider := &employeeAccountVerifierStub{
		checkResult: &infrastructure.AccountCheckResult{
			Valid:        true,
			AccountName:  "NGUYEN VAN A",
			RawErrorCode: "00",
			RawMessage:   "approved",
		},
	}
	handler := NewManualDisbursementHandler(
		disbursementservice.NewRegistry(provider),
		nil,
		employeeAccountLookupRepoStub{employee: completeLookupEmployee()},
		nil, nil, nil, slog.Default(), nil,
	)

	recorder := performEmployeeAccountCheck(t, handler, `{"employee_id":42}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if provider.checkCalls != 1 {
		t.Fatalf("check calls = %d, want 1", provider.checkCalls)
	}
	if provider.transferCalls != 0 {
		t.Fatalf("transfer calls = %d, want 0", provider.transferCalls)
	}
	if provider.lastRequest.AccountNo != "0123456789" || provider.lastRequest.AccountName != "NGUYEN VAN A" || provider.lastRequest.SwiftCode != "TESTVNVX" {
		t.Fatalf("provider request did not use persisted bank tuple: %+v", provider.lastRequest)
	}
	if provider.lastRequest.AccountType != infrastructure.AccountTypeBankAccount || provider.lastRequest.Amount != 0 {
		t.Fatalf("provider request has unexpected lookup fields: %+v", provider.lastRequest)
	}
	if len(provider.lastRequest.RequestID) != 20 || !strings.HasPrefix(provider.lastRequest.RequestID, "emplook") {
		t.Fatalf("lookup request ID = %q, want 20 characters with emplook prefix", provider.lastRequest.RequestID)
	}

	var responseBody struct {
		Data employeeAccountCheckResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Data.Outcome != employeeAccountOutcomeValid {
		t.Fatalf("outcome = %q, want valid", responseBody.Data.Outcome)
	}
	if responseBody.Data.StoredBank.AccountNumber != "0123456789" {
		t.Fatalf("stored account = %q", responseBody.Data.StoredBank.AccountNumber)
	}
}

func TestCheckEmployeeAccountRejectsIncompleteDataBeforeProviderCall(t *testing.T) {
	employee := completeLookupEmployee()
	employee.Bank.SwiftCode = ""
	provider := &employeeAccountVerifierStub{checkResult: &infrastructure.AccountCheckResult{Valid: true}}
	handler := NewManualDisbursementHandler(
		disbursementservice.NewRegistry(provider), nil,
		employeeAccountLookupRepoStub{employee: employee},
		nil, nil, nil, slog.Default(), nil,
	)

	recorder := performEmployeeAccountCheck(t, handler, `{"employee_id":42}`)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body = %s", recorder.Code, recorder.Body.String())
	}
	if provider.checkCalls != 0 {
		t.Fatalf("check calls = %d, want 0", provider.checkCalls)
	}
	if !strings.Contains(recorder.Body.String(), "employee_bank_data_incomplete") {
		t.Fatalf("missing stable error code: %s", recorder.Body.String())
	}
}

func TestCheckEmployeeAccountClassifiesProviderAnswers(t *testing.T) {
	tests := []struct {
		name   string
		result *infrastructure.AccountCheckResult
		want   string
	}{
		{name: "valid", result: &infrastructure.AccountCheckResult{Valid: true}, want: employeeAccountOutcomeValid},
		{name: "name mismatch", result: &infrastructure.AccountCheckResult{RawErrorCode: "name_mismatch"}, want: employeeAccountOutcomeNameMismatch},
		{name: "bank configuration", result: &infrastructure.AccountCheckResult{RawErrorCode: "12"}, want: employeeAccountOutcomeUnverified},
		{name: "account configuration", result: &infrastructure.AccountCheckResult{RawErrorCode: "13"}, want: employeeAccountOutcomeUnverified},
		{name: "bank unavailable", result: &infrastructure.AccountCheckResult{RawErrorCode: "19"}, want: employeeAccountOutcomeUnverified},
		{name: "invalid account", result: &infrastructure.AccountCheckResult{RawErrorCode: "14"}, want: employeeAccountOutcomeInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyEmployeeAccountCheck(test.result); got != test.want {
				t.Fatalf("outcome = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCheckEmployeeAccountDistinguishesUnavailableAndMissingResults(t *testing.T) {
	tests := []struct {
		name       string
		repository employeeAccountLookupRepository
		provider   infrastructure.DisbursementProvider
		wantStatus int
		wantCode   string
	}{
		{
			name:       "employee not found",
			repository: employeeAccountLookupRepoStub{err: domain.NewNotFoundError("missing")},
			provider:   &employeeAccountVerifierStub{},
			wantStatus: http.StatusNotFound,
			wantCode:   "employee_not_found",
		},
		{
			name:       "provider lacks verifier",
			repository: employeeAccountLookupRepoStub{employee: completeLookupEmployee()},
			provider:   employeeAccountProviderWithoutVerifier{},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "account_verifier_unavailable",
		},
		{
			name:       "provider returns empty result",
			repository: employeeAccountLookupRepoStub{employee: completeLookupEmployee()},
			provider:   &employeeAccountVerifierStub{},
			wantStatus: http.StatusBadGateway,
			wantCode:   "account_verification_empty_result",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewManualDisbursementHandler(
				disbursementservice.NewRegistry(test.provider), nil, test.repository,
				nil, nil, nil, slog.Default(), nil,
			)
			recorder := performEmployeeAccountCheck(t, handler, `{"employee_id":42}`)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), test.wantCode) {
				t.Fatalf("missing error code %q: %s", test.wantCode, recorder.Body.String())
			}
		})
	}
}

func TestCheckEmployeeAccountProviderFailureDoesNotLogAccountNumber(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	provider := &employeeAccountVerifierStub{checkErr: errors.New("provider unavailable")}
	handler := NewManualDisbursementHandler(
		disbursementservice.NewRegistry(provider), nil,
		employeeAccountLookupRepoStub{employee: completeLookupEmployee()},
		nil, nil, nil, logger, nil,
	)

	recorder := performEmployeeAccountCheck(t, handler, `{"employee_id":42}`)
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(logs.String(), "0123456789") {
		t.Fatalf("logs leaked full account number: %s", logs.String())
	}
	if !strings.Contains(recorder.Body.String(), "account_verification_provider_error") {
		t.Fatalf("missing provider error code: %s", recorder.Body.String())
	}
}

func TestCheckAccountProviderFailureDoesNotExposeCustomAccountData(t *testing.T) {
	const sentinel = "CUSTOM-ACCOUNT-SENTINEL-0123456789"
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	provider := &employeeAccountVerifierStub{checkErr: errors.New(sentinel)}
	handler := NewManualDisbursementHandler(
		disbursementservice.NewRegistry(provider), nil, nil,
		nil, nil, nil, logger, nil,
	)

	recorder := performCustomAccountCheck(t, handler, `{"bank_code":"TESTVNVX","account_no":"0123456789","account_name":"NGUYEN VAN B","account_type":"0"}`)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(logs.String(), sentinel) || strings.Contains(recorder.Body.String(), sentinel) {
		t.Fatalf("custom lookup exposed provider error data; logs=%s body=%s", logs.String(), recorder.Body.String())
	}
}

func TestCheckAccountUsesTwentyCharacterRequestID(t *testing.T) {
	provider := &employeeAccountVerifierStub{checkResult: &infrastructure.AccountCheckResult{Valid: true}}
	handler := NewManualDisbursementHandler(
		disbursementservice.NewRegistry(provider), nil, nil,
		nil, nil, nil, slog.Default(), nil,
	)

	recorder := performCustomAccountCheck(t, handler, `{"bank_code":"TESTVNVX","account_no":"0123456789","account_name":"NGUYEN VAN B","account_type":"0"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	if len(provider.lastRequest.RequestID) != 20 || !strings.HasPrefix(provider.lastRequest.RequestID, "mdcheck") {
		t.Fatalf("request ID = %q, want 20 characters with mdcheck prefix", provider.lastRequest.RequestID)
	}
}

func TestCheckAccountRejectsMalformedCustomValuesBeforeProviderCall(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "short account", body: `{"bank_code":"TESTVNVX","account_no":"12345","account_name":"NGUYEN VAN B","account_type":"0"}`},
		{name: "malformed swift", body: `{"bank_code":"BAD","account_no":"123456","account_name":"NGUYEN VAN B","account_type":"0"}`},
		{name: "short holder name", body: `{"bank_code":"TESTVNVX","account_no":"123456","account_name":"A","account_type":"0"}`},
		{name: "invalid account type", body: `{"bank_code":"TESTVNVX","account_no":"123456","account_name":"NGUYEN VAN B","account_type":"9"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &employeeAccountVerifierStub{checkResult: &infrastructure.AccountCheckResult{Valid: true}}
			handler := NewManualDisbursementHandler(
				disbursementservice.NewRegistry(provider), nil, nil,
				nil, nil, nil, slog.Default(), nil,
			)
			recorder := performCustomAccountCheck(t, handler, test.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
			}
			if provider.checkCalls != 0 {
				t.Fatalf("provider calls = %d, want 0", provider.checkCalls)
			}
		})
	}
}
