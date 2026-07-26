import { describe, expect, it } from 'vitest';

import { getLedgerReconciliationCardState } from './SettlementSimulationDialog';

describe('getLedgerReconciliationCardState', () => {
  it('shows partial transaction scope as amber instead of a false match or zero delta', () => {
    const state = getLedgerReconciliationCardState(
      {
        exported_total: 510,
        ledger_receivable: 510,
        delta: 0,
        reconciled: false,
      },
      [{
        code: 'LEDGER_PARTIAL_TRANSACTION_SCOPE',
        message: 'Có 1 giao dịch sổ cái chỉ được bao phủ một phần.',
      }],
    );

    expect(state).toEqual({
      value: 'Một phần',
      sub: 'Xem cảnh báo bên dưới',
      tone: 'amber',
    });
  });

  it.each([
    'LEDGER_RECONCILIATION_FAILED',
    'LEDGER_TRANSACTION_LINK_MISSING',
  ])('keeps %s red when a partial-scope warning is also present', (hardFailureCode) => {
    const state = getLedgerReconciliationCardState(
      {
        exported_total: 510,
        ledger_receivable: 0,
        delta: -510,
        reconciled: false,
      },
      [
        {
          code: 'LEDGER_PARTIAL_TRANSACTION_SCOPE',
          message: 'Có 1 giao dịch sổ cái chỉ được bao phủ một phần.',
        },
        {
          code: hardFailureCode,
          message: 'Dữ liệu sổ cái cần được kiểm tra.',
        },
      ],
    );

    expect(state.value).toBe('Lệch');
    expect(state.tone).toBe('red');
  });
});
