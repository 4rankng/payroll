package domain

// CompareUsers compares auditable fields of two users and returns changed fields.
func CompareUsers(original, updated *User) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.Username != updated.Username {
		changes["username"] = FieldChange{Before: original.Username, After: updated.Username}
	}
	if original.Fullname != updated.Fullname {
		changes["fullname"] = FieldChange{Before: original.Fullname, After: updated.Fullname}
	}
	if !equalStringPtr(original.Email, updated.Email) {
		changes["email"] = FieldChange{Before: strPtrToStr(original.Email), After: strPtrToStr(updated.Email)}
	}
	if !equalStringPtr(original.Mobile, updated.Mobile) {
		changes["mobile"] = FieldChange{Before: strPtrToStr(original.Mobile), After: strPtrToStr(updated.Mobile)}
	}
	if original.Role != updated.Role {
		changes["role"] = FieldChange{Before: string(original.Role), After: string(updated.Role)}
	}
	return changes
}

// CompareProjects compares auditable fields of two projects and returns changed fields.
func CompareProjects(original, updated *Project) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.Name != updated.Name {
		changes["name"] = FieldChange{Before: original.Name, After: updated.Name}
	}
	if original.Code != updated.Code {
		changes["code"] = FieldChange{Before: original.Code, After: updated.Code}
	}
	if original.ClientName != updated.ClientName {
		changes["client_name"] = FieldChange{Before: original.ClientName, After: updated.ClientName}
	}
	if original.Description != updated.Description {
		changes["description"] = FieldChange{Before: original.Description, After: updated.Description}
	}
	if !equalTimePtr(original.StartDate, updated.StartDate) {
		changes["start_date"] = FieldChange{Before: timePtrToStr(original.StartDate), After: timePtrToStr(updated.StartDate)}
	}
	if !equalTimePtr(original.EndDate, updated.EndDate) {
		changes["end_date"] = FieldChange{Before: timePtrToStr(original.EndDate), After: timePtrToStr(updated.EndDate)}
	}
	if original.ProjectStatus != updated.ProjectStatus {
		changes["project_status"] = FieldChange{Before: string(original.ProjectStatus), After: string(updated.ProjectStatus)}
	}
	if original.SalaryPeriodFrom != updated.SalaryPeriodFrom {
		changes["salary_period_from"] = FieldChange{Before: original.SalaryPeriodFrom, After: updated.SalaryPeriodFrom}
	}
	if original.SalaryPeriodTo != updated.SalaryPeriodTo {
		changes["salary_period_to"] = FieldChange{Before: original.SalaryPeriodTo, After: updated.SalaryPeriodTo}
	}
	if original.OffDays != updated.OffDays {
		changes["off_days"] = FieldChange{Before: original.OffDays, After: updated.OffDays}
	}
	return changes
}

// CompareBanks compares auditable fields of two banks and returns changed fields.
func CompareBanks(original, updated *Bank) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.BranchName != updated.BranchName {
		changes["branch_name"] = FieldChange{Before: original.BranchName, After: updated.BranchName}
	}
	if original.Bin != updated.Bin {
		changes["bin"] = FieldChange{Before: original.Bin, After: updated.Bin}
	}
	if original.BankCode != updated.BankCode {
		changes["bank_code"] = FieldChange{Before: original.BankCode, After: updated.BankCode}
	}
	return changes
}

// CompareLoans compares auditable fields of two loans and returns changed fields.
func CompareLoans(original, updated *Loan) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.LoanCode != updated.LoanCode {
		changes["loan_code"] = FieldChange{Before: original.LoanCode, After: updated.LoanCode}
	}
	if original.PrincipalAmount != updated.PrincipalAmount {
		changes["principal_amount"] = FieldChange{Before: original.PrincipalAmount, After: updated.PrincipalAmount}
	}
	if original.InterestRateBps != updated.InterestRateBps {
		changes["interest_rate_bps"] = FieldChange{Before: original.InterestRateBps, After: updated.InterestRateBps}
	}
	if original.TermMonths != updated.TermMonths {
		changes["term_months"] = FieldChange{Before: original.TermMonths, After: updated.TermMonths}
	}
	if original.StartDate != updated.StartDate {
		changes["start_date"] = FieldChange{Before: original.StartDate.Format("2006-01-02"), After: updated.StartDate.Format("2006-01-02")}
	}
	if original.EndDate != updated.EndDate {
		changes["end_date"] = FieldChange{Before: original.EndDate.Format("2006-01-02"), After: updated.EndDate.Format("2006-01-02")}
	}
	if original.PaymentDayOfMonth != updated.PaymentDayOfMonth {
		changes["payment_day_of_month"] = FieldChange{Before: original.PaymentDayOfMonth, After: updated.PaymentDayOfMonth}
	}
	if original.Status != updated.Status {
		changes["status"] = FieldChange{Before: string(original.Status), After: string(updated.Status)}
	}
	if original.LenderID != updated.LenderID {
		changes["lender_id"] = FieldChange{Before: original.LenderID, After: updated.LenderID}
	}
	if !equalStringPtr(original.Description, updated.Description) {
		changes["description"] = FieldChange{Before: strPtrToStr(original.Description), After: strPtrToStr(updated.Description)}
	}
	if original.OutstandingPrincipal != updated.OutstandingPrincipal {
		changes["outstanding_principal"] = FieldChange{Before: original.OutstandingPrincipal, After: updated.OutstandingPrincipal}
	}
	if original.TotalInterestPaid != updated.TotalInterestPaid {
		changes["total_interest_paid"] = FieldChange{Before: original.TotalInterestPaid, After: updated.TotalInterestPaid}
	}
	return changes
}

// CompareLenders compares auditable fields of two lenders and returns changed fields.
func CompareLenders(original, updated *Lender) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.Name != updated.Name {
		changes["name"] = FieldChange{Before: original.Name, After: updated.Name}
	}
	if !equalStringPtr(original.CCCD, updated.CCCD) {
		changes["cccd"] = FieldChange{Before: strPtrToStr(original.CCCD), After: strPtrToStr(updated.CCCD)}
	}
	if !equalStringPtr(original.Email, updated.Email) {
		changes["email"] = FieldChange{Before: strPtrToStr(original.Email), After: strPtrToStr(updated.Email)}
	}
	if !equalStringPtr(original.Mobile, updated.Mobile) {
		changes["mobile"] = FieldChange{Before: strPtrToStr(original.Mobile), After: strPtrToStr(updated.Mobile)}
	}
	if !equalStringPtr(original.Notes, updated.Notes) {
		changes["notes"] = FieldChange{Before: strPtrToStr(original.Notes), After: strPtrToStr(updated.Notes)}
	}
	if !equalUintPtr(original.BankID, updated.BankID) {
		changes["bank_id"] = FieldChange{Before: uintPtrToVal(original.BankID), After: uintPtrToVal(updated.BankID)}
	}
	if !equalStringPtr(original.BankAccountNumber, updated.BankAccountNumber) {
		changes["bank_account_number"] = FieldChange{Before: strPtrToStr(original.BankAccountNumber), After: strPtrToStr(updated.BankAccountNumber)}
	}
	if !equalStringPtr(original.BankAccountName, updated.BankAccountName) {
		changes["bank_account_name"] = FieldChange{Before: strPtrToStr(original.BankAccountName), After: strPtrToStr(updated.BankAccountName)}
	}
	return changes
}

// CompareTimesheets compares auditable fields of two timesheets and returns changed fields.
func CompareTimesheets(original, updated *Timesheet) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.HoursWorked != updated.HoursWorked {
		changes["hours_worked"] = FieldChange{Before: original.HoursWorked, After: updated.HoursWorked}
	}
	if original.Amount != updated.Amount {
		changes["amount"] = FieldChange{Before: original.Amount, After: updated.Amount}
	}
	if original.PayRate != updated.PayRate {
		changes["pay_rate"] = FieldChange{Before: original.PayRate, After: updated.PayRate}
	}
	if original.Status != updated.Status {
		changes["status"] = FieldChange{Before: string(original.Status), After: string(updated.Status)}
	}
	if original.PaymentStatus != updated.PaymentStatus {
		changes["payment_status"] = FieldChange{Before: string(original.PaymentStatus), After: string(updated.PaymentStatus)}
	}
	if original.Date != updated.Date {
		changes["date"] = FieldChange{Before: original.Date.Format("2006-01-02"), After: updated.Date.Format("2006-01-02")}
	}
	return changes
}

// CompareTransactions compares auditable fields of two transactions and returns changed fields.
func CompareTransactions(original, updated *Transaction) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}
	changes := make(map[string]FieldChange)
	if original.Description != updated.Description {
		changes["description"] = FieldChange{Before: original.Description, After: updated.Description}
	}
	if original.Amount != updated.Amount {
		changes["amount"] = FieldChange{Before: original.Amount, After: updated.Amount}
	}
	if original.Party != updated.Party {
		changes["party"] = FieldChange{Before: original.Party, After: updated.Party}
	}
	if original.Status != updated.Status {
		changes["status"] = FieldChange{Before: string(original.Status), After: string(updated.Status)}
	}
	if original.TransactionType != updated.TransactionType {
		changes["transaction_type"] = FieldChange{Before: string(original.TransactionType), After: string(updated.TransactionType)}
	}
	if original.URL != updated.URL {
		changes["url"] = FieldChange{Before: original.URL, After: updated.URL}
	}
	if !equalUintPtr(original.AssetID, updated.AssetID) {
		changes["asset_id"] = FieldChange{Before: uintPtrToVal(original.AssetID), After: uintPtrToVal(updated.AssetID)}
	}
	return changes
}
