package persistence

import (
	"context"
	"testing"

	"api-server/internal/domain"

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
		db.Exec("DELETE FROM advance_payment_requests WHERE employee_id IN (999901, 999902, 999903)")
		db.Exec("DELETE FROM advance_payments WHERE employee_id IN (999901, 999902, 999903)")
	}

	return repo, cleanup
}
