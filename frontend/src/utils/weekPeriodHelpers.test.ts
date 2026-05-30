
import { describe, it, expect, vi, afterEach } from 'vitest';
import { getCustomDateRanges, isEarlyMonth, isTodayAfter28th } from './weekPeriodHelpers';

describe('weekPeriodHelpers', () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    describe('Early Month (<= 10th)', () => {
        it('should identify early month correctly', () => {
            const date = new Date('2026-01-05');
            expect(isEarlyMonth(date)).toBe(true);
            expect(isTodayAfter28th(date)).toBe(false);
        });

        it('should return correct ranges for early month', () => {
            // Mock date to Jan 1st
            const date = new Date('2026-01-01');
            const ranges = getCustomDateRanges(date);

            expect(ranges).toHaveLength(2);

            // Range 1: 22-28 Previous Month (Dec 22-28)
            expect(ranges[0].label).toBe('22 - 28 Tháng 12');
            expect(ranges[0].from).toContain('2025-12-22');
            expect(ranges[0].to).toContain('2025-12-28');

            // Range 2: 01-07 Current Month (Jan 01-07)
            expect(ranges[1].label).toBe('01 - 07 Tháng 01');
            expect(ranges[1].from).toContain('2026-01-01');
            expect(ranges[1].to).toContain('2026-01-07');
        });

         it('should return correct ranges for early month (February)', () => {
            // Mock date to Feb 5th
            const date = new Date('2026-02-05');
            const ranges = getCustomDateRanges(date);

            expect(ranges).toHaveLength(2);

            // Range 1: 22-28 Previous Month (Jan 22-28)
            expect(ranges[0].label).toBe('22 - 28 Tháng 01');

            // Range 2: 01-07 Current Month (Feb 01-07)
            expect(ranges[1].label).toBe('01 - 07 Tháng 02');
        });
    });

    describe('Late Month (>= 28th)', () => {
        it('should identify late month correctly', () => {
             const date = new Date('2026-01-28');
             expect(isTodayAfter28th(date)).toBe(true);
             expect(isEarlyMonth(date)).toBe(false);
        });

        it('should return correct ranges for late month', () => {
            // Mock date to Jan 28th
            const date = new Date('2026-01-28');
            const ranges = getCustomDateRanges(date);

            expect(ranges).toHaveLength(2);

            // Range 1: 15-21 Current Month
            expect(ranges[0].label).toBe('15 - 21 Tháng 01');

            // Range 2: 22-28 Current Month
            expect(ranges[1].label).toBe('22 - 28 Tháng 01');
        });
    });

    describe('Mid Month (11th - 27th)', () => {
        it('should return empty array for mid month', () => {
            const date = new Date('2026-01-15');
             expect(isTodayAfter28th(date)).toBe(false);
             expect(isEarlyMonth(date)).toBe(false);
             expect(getCustomDateRanges(date)).toHaveLength(0);
        });
    });
});
