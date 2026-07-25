package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestCreateWithBudgetCheck_RejectsOverBudget verifies that CreateWithBudgetCheck
// rejects a request when the employee's active requests already consume the full budget.
func TestCreateWithBudgetCheck_RejectsOverBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	// Create an employee and advance_payment record
	empID, forMonth := uint64(999901), "2026-05"

	// Insert advance_payment directly to set budget = 200_000
	advPay := &domain.AdvancePayment{
		EmployeeID:   uint(empID),
		ForMonth:     forMonth,
		MaxAdvAmount: 200000,
		ProjectID:    1,
	}
	if err := repo.DB.Create(advPay).Error; err != nil {
		t.Fatalf("setup: create advance_payment: %v", err)
	}

	// First request: 150k → should succeed
	req1 := &domain.AdvancePaymentRequest{
		AdvPayID:      advPay.ID,
		ProjectID:     advPay.ProjectID,
		EmployeeID:    uint(empID),
		RequestAmount: 150000,
		Fee:           3000,
		NetAmount:     147000,
		Status:        domain.AdvancePaymentStatusPending,
	}
	if err := repo.CreateWithBudgetCheck(ctx, req1, empID, forMonth); err != nil {
		t.Fatalf("first request should succeed: %v", err)
	}

	// Second request: 100k → should FAIL (150k + 100k = 250k > 200k budget)
	req2 := &domain.AdvancePaymentRequest{
		AdvPayID:      advPay.ID,
		ProjectID:     advPay.ProjectID,
		EmployeeID:    uint(empID),
		RequestAmount: 100000,
		Fee:           2000,
		NetAmount:     98000,
		Status:        domain.AdvancePaymentStatusPending,
	}
	err := repo.CreateWithBudgetCheck(ctx, req2, empID, forMonth)
	if err == nil {
		t.Fatal("second request should be rejected (over budget)")
	}
	t.Logf("Correctly rejected: %v", err)
}

// TestCreateWithBudgetCheck_Concurrent verifies that two concurrent requests
// that together exceed the budget cannot both succeed.
func TestCreateWithBudgetCheck_Concurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	empID, forMonth := uint64(999902), "2026-05"

	// Budget: 200k
	advPay := &domain.AdvancePayment{
		EmployeeID:   uint(empID),
		ForMonth:     forMonth,
		MaxAdvAmount: 200000,
		ProjectID:    1,
	}
	if err := repo.DB.Create(advPay).Error; err != nil {
		t.Fatalf("setup: create advance_payment: %v", err)
	}

	// Two goroutines each try to request 150k (total 300k > 200k budget)
	type result struct {
		err error
		id  uint
	}
	ch := make(chan result, 2)

	for i := 0; i < 2; i++ {
		go func() {
			req := &domain.AdvancePaymentRequest{
				AdvPayID:      advPay.ID,
				ProjectID:     advPay.ProjectID,
				EmployeeID:    uint(empID),
				RequestAmount: 150000,
				Fee:           3000,
				NetAmount:     147000,
				Status:        domain.AdvancePaymentStatusPending,
			}
			err := repo.CreateWithBudgetCheck(ctx, req, empID, forMonth)
			ch <- result{err: err, id: req.ID}
		}()
	}

	var successCount, failCount int
	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err == nil {
			successCount++
			t.Logf("Request succeeded: ID=%d", r.id)
		} else {
			failCount++
			t.Logf("Request rejected: %v", r.err)
		}
	}

	if successCount > 1 {
		t.Errorf("RACE CONDITION: %d requests succeeded but budget only allows 1 (2x 150k > 200k)", successCount)
	}

	if successCount != 1 {
		t.Errorf("expected exactly 1 success and 1 failure, got %d success, %d failure", successCount, failCount)
	}
}

// TestCreateWithBudgetCheck_NoBudgetRow tests that requests are rejected when
// no advance_payment row exists for the employee/month.
func TestCreateWithBudgetCheck_NoBudgetRow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	empID, forMonth := uint64(999903), "2026-05"

	req := &domain.AdvancePaymentRequest{
		EmployeeID:    uint(empID),
		RequestAmount: 100000,
		Status:        domain.AdvancePaymentStatusPending,
	}
	err := repo.CreateWithBudgetCheck(ctx, req, empID, forMonth)
	if err == nil {
		t.Fatal("should reject when no advance_payment row exists")
	}
	t.Logf("Correctly rejected: %v", err)
}

// TestGetByEmployee_DateRange verifies the optional fromDate/toDate filter on the
// employee history list. Seeds one request stamped to the current month and one
// to the previous month, then asserts that the date-range filter returns only the
// matching subset, that the total count reflects the filter, and that nil bounds
// return all rows (backward compatibility).
func TestGetByEmployee_DateRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	// The employees/projects tables have many NOT NULL columns, so we can't
	// trivially seed throwaway rows. Discover real IDs from the test DB instead,
	// and skip if either is missing (e.g. a fresh migration with no seed data).
	var empID uint64
	if err := repo.DB.Model(&domain.Employee{}).Limit(1).
		Select("id").Scan(&empID).Error; err != nil || empID == 0 {
		t.Skipf("no employee row available to satisfy FK; skipping: %v", err)
	}
	var projectID uint64
	if err := repo.DB.Model(&domain.Project{}).Limit(1).
		Select("id").Scan(&projectID).Error; err != nil || projectID == 0 {
		t.Skipf("no project row available to satisfy FK; skipping: %v", err)
	}
	repo.DB.Unscoped().Where("employee_id = ? AND for_month = ?", empID, "2020-01").
		Delete(&domain.AdvancePayment{})

	advPay := &domain.AdvancePayment{
		EmployeeID:   uint(empID),
		ForMonth:     "2020-01",
		MaxAdvAmount: 1_000_000,
		ProjectID:    uint(projectID),
	}
	if err := repo.DB.Create(advPay).Error; err != nil {
		t.Fatalf("setup: create advance_payment: %v", err)
	}

	thisMonthTime := startOfMonth(time.Now())
	lastMonthTime := startOfMonth(time.Now().AddDate(0, -1, 0))

	reqThisMonth := &domain.AdvancePaymentRequest{
		AdvPayID:      advPay.ID,
		ProjectID:     advPay.ProjectID,
		EmployeeID:    uint(empID),
		RequestAmount: 100000,
		Fee:           2000,
		NetAmount:     98000,
		Status:        domain.AdvancePaymentStatusPending,
		CreatedAt:     thisMonthTime,
	}
	if err := repo.DB.Create(reqThisMonth).Error; err != nil {
		t.Fatalf("create this-month request: %v", err)
	}

	reqLastMonth := &domain.AdvancePaymentRequest{
		AdvPayID:      advPay.ID,
		ProjectID:     advPay.ProjectID,
		EmployeeID:    uint(empID),
		RequestAmount: 200000,
		Fee:           4000,
		NetAmount:     196000,
		Status:        domain.AdvancePaymentStatusPending,
		CreatedAt:     lastMonthTime,
	}
	if err := repo.DB.Create(reqLastMonth).Error; err != nil {
		t.Fatalf("create last-month request: %v", err)
	}

	// No filter → both rows (backward compatibility).
	all, totalAll, err := repo.GetByEmployee(ctx, empID, 100, 0, nil, nil, nil)
	if err != nil {
		t.Fatalf("GetByEmployee (no filter): %v", err)
	}
	if totalAll < 2 || len(all) < 2 {
		t.Fatalf("expected >=2 rows without filter, got total=%d len=%d", totalAll, len(all))
	}
	for _, request := range all {
		if request.ID != reqThisMonth.ID && request.ID != reqLastMonth.ID {
			continue
		}
		if request.AdvancePayment == nil || request.AdvancePayment.ForMonth != "2020-01" {
			t.Fatalf("request id=%d must preload payroll month 2020-01, got %#v", request.ID, request.AdvancePayment)
		}
	}

	// Filter to the previous month only → reqLastMonth, not reqThisMonth.
	prevEnd := endOfMonth(lastMonthTime)
	filtered, totalFiltered, err := repo.GetByEmployee(ctx, empID, 100, 0, &lastMonthTime, &prevEnd, nil)
	if err != nil {
		t.Fatalf("GetByEmployee (prev month): %v", err)
	}
	if totalFiltered != 1 || len(filtered) != 1 {
		t.Fatalf("expected 1 row in previous month, got total=%d len=%d", totalFiltered, len(filtered))
	}
	if filtered[0].ID != reqLastMonth.ID {
		t.Errorf("expected last-month request id=%d, got id=%d", reqLastMonth.ID, filtered[0].ID)
	}

	// Filter to the current month only → reqThisMonth, not reqLastMonth.
	thisEnd := endOfMonth(thisMonthTime)
	thisMonthRows, totalThis, err := repo.GetByEmployee(ctx, empID, 100, 0, &thisMonthTime, &thisEnd, nil)
	if err != nil {
		t.Fatalf("GetByEmployee (this month): %v", err)
	}
	if totalThis != 1 || len(thisMonthRows) != 1 {
		t.Fatalf("expected 1 row in current month, got total=%d len=%d", totalThis, len(thisMonthRows))
	}
	if thisMonthRows[0].ID != reqThisMonth.ID {
		t.Errorf("expected this-month request id=%d, got id=%d", reqThisMonth.ID, thisMonthRows[0].ID)
	}

	// Salary-period (for_month) filter: both requests are charged to 2020-01 even
	// though reqLastMonth was created a month earlier, so the filter must surface
	// both. This is the employee-history guarantee — group by salary period, not
	// by the calendar month the request was submitted in.
	janFM := "2020-01"
	byMonth, _, err := repo.GetByEmployee(ctx, empID, 100, 0, nil, nil, &janFM)
	if err != nil {
		t.Fatalf("GetByEmployee (for_month=2020-01): %v", err)
	}
	seen := make(map[uint]bool, len(byMonth))
	for _, r := range byMonth {
		seen[r.ID] = true
	}
	if !seen[reqThisMonth.ID] || !seen[reqLastMonth.ID] {
		t.Errorf("for_month=2020-01 must include both requests regardless of created_at month; got %v", seen)
	}

	// A different salary period returns neither of our 2020-01 requests.
	otherFM := "2020-02"
	otherRows, _, err := repo.GetByEmployee(ctx, empID, 100, 0, nil, nil, &otherFM)
	if err != nil {
		t.Fatalf("GetByEmployee (for_month=2020-02): %v", err)
	}
	for _, r := range otherRows {
		if r.ID == reqThisMonth.ID || r.ID == reqLastMonth.ID {
			t.Errorf("for_month=2020-02 must not return 2020-01 requests; got id=%d", r.ID)
		}
	}

	// Cleanup the rows this test created (setupTestRepo only wipes 999901's
	// advance_payment_requests for the budget-check month; ours are on 2020-01).
	repo.DB.Where("id IN (?, ?)", reqThisMonth.ID, reqLastMonth.ID).
		Delete(&domain.AdvancePaymentRequest{})
	repo.DB.Where("id = ?", advPay.ID).Delete(&domain.AdvancePayment{})
}

// TestGetOrphanedApproved_WalletPaymentStatusFiltering is a table-driven
// regression test for the double-payment-safety boundary in
// GetOrphanedApproved. An APPROVED advance request must only be returned as
// an orphan (and thus re-enqueued by the poller) when it has NO
// wallet_payment in a non-failed/non-reversed state.
//
//   - only-failed            → returned (retry needed)
//   - only-reversed          → returned (retry needed)
//   - only-completed         → excluded (money sent — never retry)
//   - only-pending           → excluded (in-flight)
//   - only-verified          → excluded (in-flight)
//   - only-authorised        → excluded (in-flight)
//   - failed + completed     → excluded (at least one success — never retry)
//   - no wallet_payment      → returned (never attempted)
func TestGetOrphanedApproved_WalletPaymentStatusFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repo, cleanup := setupTestRepo(t)
	defer cleanup()
	ctx := context.Background()

	// The advance_payment_requests table has FK constraints on employee_id
	// and project_id, so we discover real IDs from the test DB.
	var empID uint64
	if err := repo.DB.Model(&domain.Employee{}).Limit(1).
		Select("id").Scan(&empID).Error; err != nil || empID == 0 {
		t.Skipf("no employee row available to satisfy FK; skipping: %v", err)
	}
	var projectID uint64
	if err := repo.DB.Model(&domain.Project{}).Limit(1).
		Select("id").Scan(&projectID).Error; err != nil || projectID == 0 {
		t.Skipf("no project row available to satisfy FK; skipping: %v", err)
	}

	const provider = "1pay"
	// Each case uses a distinct YYYY-MM (column is varchar(7)).
	testMonths := []string{"2018-01", "2018-02", "2018-03", "2018-04", "2018-05", "2018-06", "2018-07", "2018-08"}

	cases := []struct {
		name       string
		wpStatuses []domaintx.State // wallet_payment statuses to seed for this request
		wantOrphan bool             // should GetOrphanedApproved return this request?
	}{
		{"only_failed", []domaintx.State{domaintx.StateFailed}, true},
		{"only_reversed", []domaintx.State{domaintx.StateReversed}, true},
		{"only_completed", []domaintx.State{domaintx.StateCompleted}, false},
		{"only_pending", []domaintx.State{domaintx.StatePending}, false},
		{"only_verified", []domaintx.State{domaintx.StateVerified}, false},
		{"only_authorised", []domaintx.State{domaintx.StateAuthorised}, false},
		{"failed_and_completed", []domaintx.State{domaintx.StateFailed, domaintx.StateCompleted}, false},
		{"no_wallet_payment", nil, true},
	}

	type seed struct {
		reqID    uint
		wpTxnIDs []uuid.UUID
	}
	seeds := make([]seed, 0, len(cases))

	for i, tc := range cases {
		advPay := &domain.AdvancePayment{
			EmployeeID:   uint(empID),
			ForMonth:     testMonths[i],
			MaxAdvAmount: 1_000_000,
			ProjectID:    uint(projectID),
		}
		if err := repo.DB.Create(advPay).Error; err != nil {
			t.Fatalf("setup [%s]: create advance_payment: %v", tc.name, err)
		}

		req := &domain.AdvancePaymentRequest{
			AdvPayID:      advPay.ID,
			ProjectID:     advPay.ProjectID,
			EmployeeID:    uint(empID),
			RequestAmount: 50000,
			Fee:           1000,
			NetAmount:     49000,
			Status:        domain.AdvancePaymentStatusApproved,
		}
		if err := repo.DB.Create(req).Error; err != nil {
			t.Fatalf("setup [%s]: create request: %v", tc.name, err)
		}

		// Force updated_at stale using MySQL's own clock so the orphan
		// window (updated_at < NOW() - 5 min) is deterministic regardless of
		// the Go/MySQL timezone offset.
		if err := repo.DB.Exec(
			"UPDATE advance_payment_requests SET updated_at = DATE_SUB(NOW(), INTERVAL 10 MINUTE) WHERE id = ?",
			req.ID,
		).Error; err != nil {
			t.Fatalf("setup [%s]: set stale updated_at: %v", tc.name, err)
		}

		txnIDs := make([]uuid.UUID, 0, len(tc.wpStatuses))
		for _, st := range tc.wpStatuses {
			txnID := uuid.New()
			wpReqID := fmt.Sprintf("tt-test-%s-%s", tc.name, txnID.String()[:8])
			wp := &domaintx.WalletPayment{
				TxnID:              txnID,
				RequestID:          wpReqID,
				Provider:           provider,
				RequestedAmount:    49000,
				Fee:                1000,
				RecipientName:      "Test Recipient",
				RecipientAccountNo: "9999999999",
				RecipientBank:      "TESTBANK",
				Status:             st,
				EntityID:           uint64Ptr(req.ID),
				Version:            0,
			}
			if err := repo.DB.Create(wp).Error; err != nil {
				t.Fatalf("setup [%s]: create wallet_payment (%s): %v", tc.name, st, err)
			}
			txnIDs = append(txnIDs, txnID)
		}

		seeds = append(seeds, seed{reqID: req.ID, wpTxnIDs: txnIDs})
	}

	orphans, err := repo.GetOrphanedApproved(ctx, 100, provider)
	if err != nil {
		t.Fatalf("GetOrphanedApproved: %v", err)
	}

	returned := make(map[uint]bool, len(orphans))
	for _, o := range orphans {
		returned[o.ID] = true
	}

	for i, tc := range cases {
		reqID := seeds[i].reqID
		if got := returned[reqID]; got != tc.wantOrphan {
			t.Errorf("case %q (request id=%d): want orphan=%v, got orphan=%v", tc.name, reqID, tc.wantOrphan, got)
		}
	}

	// Cleanup.
	for _, s := range seeds {
		if len(s.wpTxnIDs) > 0 {
			repo.DB.Where("txn_id IN ?", s.wpTxnIDs).Delete(&domaintx.WalletPayment{})
		}
		repo.DB.Where("id = ?", s.reqID).Delete(&domain.AdvancePaymentRequest{})
	}
	repo.DB.Unscoped().Where("employee_id = ? AND for_month IN ?", empID, testMonths).
		Delete(&domain.AdvancePayment{})
}

// uint64Ptr is a small helper to take the address of a uint value as *uint64.
func uint64Ptr(v uint) *uint64 {
	u := uint64(v)
	return &u
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func endOfMonth(t time.Time) time.Time {
	return startOfMonth(t).AddDate(0, 1, -1)
}

// getTestDB connects to the local test database.
func getTestDB() (*gorm.DB, error) {
	dsn := "root:rootpassword@tcp(localhost:3306)/payroll_db?parseTime=true&loc=Local"
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// setupTestRepo creates a test database connection and cleans up test data.
// It uses the same DB config as the running server (localhost:3306/payroll_db).
func setupTestRepo(t *testing.T) (*AdvancePaymentRequestRepository, func()) {
	t.Helper()

	db, err := getTestDB()
	if err != nil {
		t.Skipf("cannot connect to test DB: %v", err)
	}

	repo := &AdvancePaymentRequestRepository{
		BaseRepository: &BaseRepository{DB: db},
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM advance_payment_requests WHERE employee_id IN (999901, 999902, 999903, 999904)")
		db.Exec("DELETE FROM advance_payments WHERE employee_id IN (999901, 999902, 999903, 999904)")
	}

	return repo, cleanup
}
