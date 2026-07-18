import { describe, expect, it } from 'vitest';

import { formatBankTransferDateTime, formatBankTransferPeriod } from './bankTransferHistoryHelpers';

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
