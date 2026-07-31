package onepay

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccountInfoReturnsEmployeeAccountHolderName(t *testing.T) {
	t.Setenv("MOCK_ONEPAY_HOLDER_NAME", "")

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, statement := range []string{
		`CREATE TABLE banks (
			id INTEGER PRIMARY KEY,
			swift_code TEXT NOT NULL
		)`,
		`CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			fullname TEXT NOT NULL,
			bank_id INTEGER,
			bank_account_number TEXT NOT NULL,
			bank_account_name TEXT,
			deleted_at DATETIME
		)`,
		`INSERT INTO banks (id, swift_code) VALUES (1, 'VCBVVNVX')`,
		`INSERT INTO employees (id, fullname, bank_id, bank_account_number, bank_account_name)
		 VALUES (7, 'Nguyễn Việt Duy', 1, '0123456789', 'NGUYEN VIET DUY')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("prepare database: %v", err)
		}
	}

	server := &Server{
		cfg: Config{SkipVerify: true},
		recoverer: &recoverer{
			db: db,
		},
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/onepayout/api/v1/customers?request_id=empval1&account_number=0123456789&swift_code=VCBVVNVX",
		nil,
	)
	response := httptest.NewRecorder()

	server.getAccountInfo(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		HolderName string `json:"holder_name"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.HolderName != "NGUYEN VIET DUY" {
		t.Fatalf("holder_name = %q, want account holder name %q", body.HolderName, "NGUYEN VIET DUY")
	}
}

func TestAccountHolderNameUsesExplicitOverride(t *testing.T) {
	t.Setenv("MOCK_ONEPAY_HOLDER_NAME", "MISMATCH TEST NAME")

	name, err := (&Server{}).accountHolderName(t.Context(), "0123456789", "VCBVVNVX")

	if err != nil {
		t.Fatalf("accountHolderName returned error: %v", err)
	}
	if name != "MISMATCH TEST NAME" {
		t.Fatalf("holder name = %q, want explicit override", name)
	}
}

func TestAccountHolderNameReturnsEmptyWhenEmployeeIsNotFound(t *testing.T) {
	t.Setenv("MOCK_ONEPAY_HOLDER_NAME", "")

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`CREATE TABLE banks (id INTEGER PRIMARY KEY, swift_code TEXT NOT NULL)`,
		`CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			fullname TEXT NOT NULL,
			bank_id INTEGER,
			bank_account_number TEXT NOT NULL,
			bank_account_name TEXT,
			deleted_at DATETIME
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("prepare database: %v", err)
		}
	}

	name, err := (&Server{recoverer: &recoverer{db: db}}).
		accountHolderName(t.Context(), "not-found", "VCBVVNVX")

	if err != nil {
		t.Fatalf("accountHolderName returned error: %v", err)
	}
	if name != "" {
		t.Fatalf("holder name = %q, want empty fallback", name)
	}
}

func TestGetAccountInfoReturnsServerErrorWhenEmployeeLookupFails(t *testing.T) {
	t.Setenv("MOCK_ONEPAY_HOLDER_NAME", "")

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	server := &Server{
		cfg:       Config{SkipVerify: true},
		recoverer: &recoverer{db: db},
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/onepayout/api/v1/customers?request_id=empval1&account_number=0123456789&swift_code=VCBVVNVX",
		nil,
	)
	response := httptest.NewRecorder()

	server.getAccountInfo(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
