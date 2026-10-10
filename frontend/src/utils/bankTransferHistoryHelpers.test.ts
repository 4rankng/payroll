import { describe, expect, it } from 'vitest';

import {
  BANK_TRANSFER_CYCLE_LABELS,
  formatBankTransferDateTime,
  formatBankTransferPeriod,
} from './bankTransferHistoryHelpers';

describe('BANK_TRANSFER_CYCLE_LABELS', () => {
  it('pins the four cycle labels', () => {
    expect(BANK_TRANSFER_CYCLE_LABELS).toEqual({
      1: 'Kỳ 1 · ngày 1–7',
      2: 'Kỳ 2 · ngày 8–14',
      3: 'Kỳ 3 · ngày 15–21',
      4: 'Kỳ 4 · ngày 22–cuối tháng',
    });
  });
});

describe('formatBankTransferPeriod', () => {
  it('compacts a period within the same month', () => {
    expect(formatBankTransferPeriod('2026-07-08', '2026-07-14')).toBe('08–14/07/2026');
  });

  it('keeps both complete dates when a period crosses a month boundary', () => {
    expect(formatBankTransferPeriod('2026-07-29', '2026-08-04')).toBe('29/07/2026–04/08/2026');
  });
});

describe('formatBankTransferDateTime', () => {
  it('formats an instant in the Vietnam business timezone', () => {
    expect(formatBankTransferDateTime('2026-07-17T13:24:00Z')).toBe('17/07/2026 · 20:24');
  });

  it('returns a fallback for missing or invalid legacy values', () => {
    expect(formatBankTransferDateTime()).toBe('—');
    expect(formatBankTransferDateTime('not-a-date')).toBe('—');
  });
});
