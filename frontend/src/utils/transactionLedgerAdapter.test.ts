import { describe, expect, it } from 'vitest';

import { transactionToLedgerEntryView } from './transactionLedgerAdapter';
import type { Transaction } from '@/services/api/transaction.service';

const transaction: Transaction = {
  id: 4242,
  description: 'Chi lương tuần',
  transaction_code: 'itest2026102842426746400000000000000000_vendor',
  transaction_type: 'payroll_payment',
  amount: 1_000_000,
  settled_amount: 400_000,
  remaining_amount: 600_000,
  party: 'itest2026102842426746400000000000000000_vendor',
  status: 'pending',
  settlement_method: 'payable',
  url: 'https://evidence.example/x.pdf',
  asset_id: 77,
  asset: undefined,
  created_by: 9,
  created_at: '2026-10-28T10:00:00Z',
  updated_at: '2026-10-28T11:00:00Z',
};

describe('transactionToLedgerEntryView (PAYR-20 adapter)', () => {
  const view = transactionToLedgerEntryView(transaction);

  it('giữ transaction id — không được hoán sang id khác (details/reversal dùng id này)', () => {
    expect(view.id).toBe(4242);
  });

  it('map các trường hiển thị sang card shape', () => {
    expect(view).toMatchObject({
      account: 'payable',
      party: 'itest2026102842426746400000000000000000_vendor',
      description: 'Chi lương tuần',
      debit: 1_000_000,
      credit: 1_000_000,
      net_amount: 1_000_000,
      balance: 600_000,
      date: '2026-10-28T10:00:00Z',
      url: 'https://evidence.example/x.pdf',
      asset_id: 77,
      reference: 'itest2026102842426746400000000000000000_vendor',
      created_by: 9,
      created_at: '2026-10-28T10:00:00Z',
      updated_at: '2026-10-28T11:00:00Z',
    });
    expect(view.asset).toBeUndefined();
  });

  it('ref dài ≥43 ký tự nằm trọn trong party/reference (card wrap ở layer render, không cắt trong adapter)', () => {
    expect(view.party.length).toBeGreaterThanOrEqual(43);
    expect(view.reference!.length).toBeGreaterThanOrEqual(43);
    expect(view.party).toBe(transaction.transaction_code);
  });
});
