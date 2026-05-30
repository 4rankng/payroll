package persistence

import (
	"context"
	"fmt"
	"sort"
	"time"

	"api-server/internal/domain"
)

func (r *LedgerEntryRepository) GetCashFlowSummary(ctx context.Context, start, end time.Time) (*domain.CashFlowSummary, error) {
	summary := &domain.CashFlowSummary{
		StartDate: start,
		EndDate:   end,
		ByAccount: make(map[domain.LedgerAccount]domain.AccountSummary),
	}

	// Query 1: period account summaries
	var periodResults []accountAgg
	if err := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Select("account, CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit").
		Where("date BETWEEN ? AND ? AND deleted_at IS NULL", start, end).
		Group("account").
		Scan(&periodResults).Error; err != nil {
		return nil, err
	}

	var revenueCredits, revenueDebits, cashDebits, cashCredits, expenseDebits int64
	for _, acc := range periodResults {
		switch acc.Account {
		case domain.AccountRevenue:
			revenueCredits, revenueDebits = acc.TotalCredit, acc.TotalDebit
		case domain.AccountCash:
			cashDebits, cashCredits = acc.TotalDebit, acc.TotalCredit
		case domain.AccountExpense:
			expenseDebits = acc.TotalDebit
		}
		summary.ByAccount[domain.LedgerAccount(acc.Account)] = domain.AccountSummary{
			Account:     domain.LedgerAccount(acc.Account),
			TotalDebit:  acc.TotalDebit,
			TotalCredit: acc.TotalCredit,
			NetAmount:   domain.GetUserBalance(domain.LedgerAccount(acc.Account), acc.TotalDebit, acc.TotalCredit),
		}
	}

	summary.TotalInflow = revenueCredits + cashDebits
	summary.TotalOutflow = revenueDebits + cashCredits + expenseDebits
	summary.NetCashFlow = summary.TotalInflow - summary.TotalOutflow

	// Query 2: opening balance (all transactions before start)
	openingBalance, err := calculateNetWorthBefore(r.DB.WithContext(ctx), start)
	if err != nil {
		return nil, err
	}
	summary.OpeningBalance = openingBalance
	summary.ClosingBalance = summary.OpeningBalance + summary.NetCashFlow

	return summary, nil
}

func (r *LedgerEntryRepository) GetTotalByAccountType(ctx context.Context, accountType string, startDate, endDate time.Time) (int64, error) {
	type totals struct {
		TotalDebit  int64
		TotalCredit int64
	}
	var res totals

	query := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Where("account = ? AND deleted_at IS NULL", accountType)
	if !startDate.IsZero() {
		query = query.Where("date BETWEEN ? AND ?", startDate, endDate)
	} else {
		query = query.Where("date <= ?", endDate)
	}

	if err := query.
		Select("CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit").
		Row().
		Scan(&res.TotalDebit, &res.TotalCredit); err != nil {
		return 0, err
	}

	return domain.GetUserBalance(domain.LedgerAccount(accountType), res.TotalDebit, res.TotalCredit), nil
}

// GetMonthlyFinancialsBatch retrieves revenue and expenses for all months in a date range in one query.
func (r *LedgerEntryRepository) GetMonthlyFinancialsBatch(ctx context.Context, startDate, endDate time.Time) (map[string]domain.MonthlyFinancials, error) {
	type monthlyResult struct {
		Month       string
		Account     string
		TotalDebit  int64
		TotalCredit int64
	}

	var results []monthlyResult
	if err := r.DB.WithContext(ctx).
		Model(&domain.LedgerEntry{}).
		Select("DATE_FORMAT(date, '%Y-%m') as month, account, CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit").
		Where("account IN (?, ?) AND date BETWEEN ? AND ? AND deleted_at IS NULL", domain.AccountRevenue, domain.AccountExpense, startDate, endDate).
		Group("DATE_FORMAT(date, '%Y-%m'), account").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	monthlyData := make(map[string]domain.MonthlyFinancials)
	for _, res := range results {
		monthly := monthlyData[res.Month]
		monthly.Month = res.Month
		switch res.Account {
		case domain.AccountRevenue:
			monthly.Revenue = res.TotalCredit - res.TotalDebit
		case domain.AccountExpense:
			monthly.Expenses = res.TotalDebit - res.TotalCredit
		}
		monthlyData[res.Month] = monthly
	}
	return monthlyData, nil
}

// GetLedgerSummary calculates a comprehensive ledger summary for a date range.
//
// ACCOUNTING FORMULA:
//
//	Net Worth = Assets - Liabilities = Cash + Receivable - Payable
func (r *LedgerEntryRepository) GetLedgerSummary(ctx context.Context, fromDate, toDate time.Time) (*domain.LedgerSummaryData, error) {
	summary := &domain.LedgerSummaryData{
		AccountSummaries: make(map[string]domain.AccountSummaryData),
	}

	// Opening balance: all transactions before fromDate
	openingBalance, err := calculateNetWorthBefore(r.DB.WithContext(ctx), fromDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate opening balance: %w", err)
	}
	summary.OpeningBalance = openingBalance

	// Period account summaries
	var periodRows []accountAgg
	if err := r.DB.WithContext(ctx).Model(&domain.LedgerEntry{}).
		Select("account, CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit, CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit").
		Where("date BETWEEN ? AND ? AND deleted_at IS NULL", fromDate, toDate).
		Group("account").
		Scan(&periodRows).Error; err != nil {
		return nil, fmt.Errorf("failed to get account summaries: %w", err)
	}

	for _, acc := range periodRows {
		summary.AccountSummaries[acc.Account] = domain.AccountSummaryData{
			TotalDebit:  acc.TotalDebit,
			TotalCredit: acc.TotalCredit,
			NetAmount:   domain.GetUserBalance(domain.LedgerAccount(acc.Account), acc.TotalDebit, acc.TotalCredit),
		}
	}

	// Closing balance + equity — single query up to toDate
	type financialSummary struct {
		Account     string `gorm:"column:account"`
		CreatedBy   *uint  `gorm:"column:created_by"`
		TotalDebit  int64  `gorm:"column:total_debit"`
		TotalCredit int64  `gorm:"column:total_credit"`
	}
	var financialData []financialSummary
	if err := r.DB.WithContext(ctx).Model(&domain.LedgerEntry{}).
		Select(`account, created_by,
			CAST(COALESCE(SUM(debit), 0) AS SIGNED) as total_debit,
			CAST(COALESCE(SUM(credit), 0) AS SIGNED) as total_credit`).
		Where("date <= ? AND deleted_at IS NULL", toDate).
		Group("account, created_by").
		Scan(&financialData).Error; err != nil {
		return nil, fmt.Errorf("failed to get financial summary: %w", err)
	}

	var closingBalance int64
	contributionMap := make(map[uint]int64)
	var totalEquity int64
	for _, d := range financialData {
		switch d.Account {
		case domain.AccountCash, domain.AccountReceivable:
			closingBalance += d.TotalDebit - d.TotalCredit
		case domain.AccountPayable, domain.AccountLoan:
			closingBalance -= d.TotalCredit - d.TotalDebit
		case domain.AccountEquity:
			if d.CreatedBy != nil {
				contrib := d.TotalCredit - d.TotalDebit
				contributionMap[*d.CreatedBy] += contrib
				totalEquity += contrib
			}
		}
	}
	summary.ClosingBalance = closingBalance

	// Owner contributions
	var adminUsers []domain.User
	if err := r.DB.WithContext(ctx).
		Select("id, fullname").
		Where("role = ? AND deleted_at IS NULL", domain.RoleAdmin).
		Find(&adminUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to get admin users: %w", err)
	}

	ownerContributions := make([]domain.OwnerContributionData, 0, len(adminUsers))
	for _, admin := range adminUsers {
		contrib := contributionMap[admin.ID]
		pct := 0.0
		if totalEquity > 0 {
			pct = float64(contrib) / float64(totalEquity) * 100
		}
		ownerContributions = append(ownerContributions, domain.OwnerContributionData{
			Owner:             admin.Fullname,
			TotalContribution: contrib,
			Percentage:        pct,
		})
	}
	summary.OwnerContributions = ownerContributions

	return summary, nil
}

// periodData holds per-period debit/credit accumulators for chart data.
type periodData struct {
	Date             string
	CashDebit        int64
	CashCredit       int64
	ReceivableDebit  int64
	ReceivableCredit int64
	PayableDebit     int64
	PayableCredit    int64
	RevenueDebit     int64
	RevenueCredit    int64
	ExpenseDebit     int64
	ExpenseCredit    int64
}

func (r *LedgerEntryRepository) GetFinancialChartData(ctx context.Context, period string, startDate, endDate time.Time) ([]domain.FinancialChartDataPoint, error) {
	var entries []domain.LedgerEntry
	if err := r.DB.WithContext(ctx).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Order("date ASC").
		Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}

	periodMap := make(map[string]*periodData)
	for _, entry := range entries {
		key := r.queryBuilder.GetPeriodKey(entry.Date, period)
		if periodMap[key] == nil {
			periodMap[key] = &periodData{Date: key}
		}
		pd := periodMap[key]
		switch entry.Account {
		case domain.AccountCash:
			pd.CashDebit += entry.Debit
			pd.CashCredit += entry.Credit
		case domain.AccountReceivable:
			pd.ReceivableDebit += entry.Debit
			pd.ReceivableCredit += entry.Credit
		case domain.AccountPayable:
			pd.PayableDebit += entry.Debit
			pd.PayableCredit += entry.Credit
		case domain.AccountRevenue:
			pd.RevenueDebit += entry.Debit
			pd.RevenueCredit += entry.Credit
		case domain.AccountExpense:
			pd.ExpenseDebit += entry.Debit
			pd.ExpenseCredit += entry.Credit
		}
	}

	// Ensure all periods in range are present (even with zero values)
	for _, key := range r.queryBuilder.GeneratePeriodKeys(startDate, endDate, period) {
		if periodMap[key] == nil {
			periodMap[key] = &periodData{Date: key}
		}
	}

	keys := make([]string, 0, len(periodMap))
	for k := range periodMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	dataPoints := make([]domain.FinancialChartDataPoint, 0, len(keys))
	var (
		cumCashD, cumCashC int64
		cumRecvD, cumRecvC int64
		cumPayD, cumPayC   int64
		cumRevD, cumRevC   int64
		cumExpD, cumExpC   int64
	)
	for _, key := range keys {
		pd := periodMap[key]
		cumCashD += pd.CashDebit
		cumCashC += pd.CashCredit
		cumRecvD += pd.ReceivableDebit
		cumRecvC += pd.ReceivableCredit
		cumPayD += pd.PayableDebit
		cumPayC += pd.PayableCredit
		cumRevD += pd.RevenueDebit
		cumRevC += pd.RevenueCredit
		cumExpD += pd.ExpenseDebit
		cumExpC += pd.ExpenseCredit

		rev := domain.GetUserBalance(domain.AccountRevenue, cumRevD, cumRevC)
		exp := domain.GetUserBalance(domain.AccountExpense, cumExpD, cumExpC)
		dataPoints = append(dataPoints, domain.FinancialChartDataPoint{
			Date:            pd.Date,
			CashVND:         domain.GetUserBalance(domain.AccountCash, cumCashD, cumCashC),
			ReceivableVND:   domain.GetUserBalance(domain.AccountReceivable, cumRecvD, cumRecvC),
			PayableVND:      domain.GetUserBalance(domain.AccountPayable, cumPayD, cumPayC),
			RevenueVND:      rev,
			ExpensesVND:     exp,
			ProfitVND:       rev - exp,
			ActiveEmployees: 0, // populated by dashboard service
		})
	}
	return dataPoints, nil
}
