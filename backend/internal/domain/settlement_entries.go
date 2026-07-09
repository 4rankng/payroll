package domain

import "time"

// BuildDisbursementSettlementEntries builds the three double-entry rows that
// post a disbursement settlement to the ledger:
//
//	Cash       credit net          (party "Nhân viên")
//	Receivable debit  net+fee      (partner company)
//	Revenue    credit fee          (partner company)
//
// Debits (net+fee) equal credits (net+fee), so the entry is balanced. Worker and
// bulk-transfer (ProcessBankResult) share this builder so the journal shape can
// never drift between the two settlement paths. createdBy is the system user id;
// partnerCompany is the receivable/revenue party (each caller resolves its own).
func BuildDisbursementSettlementEntries(transactionID uint, net, fee int64, partnerCompany string, createdBy uint, at time.Time) []*LedgerEntry {
	tid := transactionID
	return []*LedgerEntry{
		{
			Date:          at,
			Account:       AccountCash,
			Party:         "Nhân viên",
			Debit:         0,
			Credit:        net,
			CreatedBy:     createdBy,
			TransactionID: &tid,
		},
		{
			Date:          at,
			Account:       AccountReceivable,
			Party:         partnerCompany,
			Debit:         net + fee,
			Credit:        0,
			CreatedBy:     createdBy,
			TransactionID: &tid,
		},
		{
			Date:          at,
			Account:       AccountRevenue,
			Party:         partnerCompany,
			Debit:         0,
			Credit:        fee,
			CreatedBy:     createdBy,
			TransactionID: &tid,
		},
	}
}
