import { describe, expect, it } from 'vitest';
import { formatSettlementBusinessDate, getLatestSettlementDate } from './transactionHelpers';

describe('transactionHelpers', () => {
  describe('getLatestSettlementDate', () => {
    it('returns the latest settlement business date regardless of API ordering', () => {
      expect(getLatestSettlementDate([
        { settlement_date: '2026-07-31' },
        { settlement_date: '2026-07-24' },
        { settlement_date: '2026-07-28' },
      ])).toBe('2026-07-31');
    });

    it('does not substitute the transaction creation date when settlement history is absent', () => {
      expect(getLatestSettlementDate()).toBeNull();
      expect(getLatestSettlementDate([])).toBeNull();
    });

    it('ignores invalid settlement dates', () => {
      expect(getLatestSettlementDate([
        { settlement_date: '' },
        { settlement_date: 'not-a-date' },
        { settlement_date: '2026-02-30' },
        { settlement_date: '2026-02-28' },
      ])).toBe('2026-02-28');
    });
  });

  it('formats the business date without inventing a time', () => {
    expect(formatSettlementBusinessDate('2026-07-31')).toBe('31 tháng 07 2026');
  });
});
