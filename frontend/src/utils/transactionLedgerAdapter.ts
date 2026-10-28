import type { Transaction } from '@/services/api/transaction.service';
import type { LedgerEntry } from '@/types/api/financial.types';

/**
 * View-only adapter (PAYR-20): the /admin/ledger list is served by the
 * Transaction API on BOTH desktop and mobile — one model, one source of truth.
 * The LedgerEntry shape here is purely the card contract consumed by
 * LedgerMobileList; `id` remains a TRANSACTION id, so details/reversal go
 * through transaction endpoints (openTransactionDetails), never
 * ledger-entry endpoints.
 */
export function transactionToLedgerEntryView(transaction: Transaction): LedgerEntry {
  return {
    id: transaction.id,
    account: transaction.settlement_method,
    party: transaction.party,
    description: transaction.description,
    debit: transaction.amount,
    credit: transaction.amount,
    balance: transaction.remaining_amount ?? 0,
    net_amount: transaction.amount,
    date: transaction.created_at,
    url: transaction.url,
    asset_id: transaction.asset_id,
    asset: transaction.asset ?? undefined,
    reference: transaction.transaction_code,
    created_by: transaction.created_by,
    created_at: transaction.created_at,
    updated_at: transaction.updated_at,
  };
}
