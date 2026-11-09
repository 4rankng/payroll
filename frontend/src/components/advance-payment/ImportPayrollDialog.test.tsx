import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ImportPayrollDialog, getMonthOptions } from './ImportPayrollDialog';

const mutate = vi.fn();

vi.mock('@/hooks/api/useAdvancePayments', () => ({
  useImportFlexTemplate: () => ({
    mutate,
    isPending: false,
  }),
}));

vi.mock('@/hooks/api/useSettings', () => ({
  // No settings row in these tests → the dialog falls back to stop day 9.
  useSettingByKey: () => ({ data: undefined }),
}));

vi.mock('./FileDropZone', () => ({
  FileDropZone: ({ onFileChange }: { onFileChange: (file: File) => void }) => (
    <button
      type="button"
      onClick={() => onFileChange(new File(['UL'], 'lgd.xlsx', {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      }))}
    >
      Chọn tệp thử nghiệm
    </button>
  ),
}));

describe('ImportPayrollDialog', () => {
  it('sends force_reprocess when the admin chooses to process the file again', async () => {
    mutate.mockClear();
    render(<ImportPayrollDialog open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByRole('button', { name: 'Chọn tệp thử nghiệm' }));
    fireEvent.click(screen.getByLabelText('Xử lý lại tệp'));
    fireEvent.click(screen.getByRole('button', { name: 'Nhập' }));

    await waitFor(() => expect(mutate).toHaveBeenCalledOnce());
    const formData = mutate.mock.calls[0][0] as FormData;
    expect(formData.get('force_reprocess')).toBe('true');
    // The month selector defaults to the payroll period (YYYY-MM) and sends it
    // explicitly; the server only derives it when the field is absent. During
    // the request tail (day < stop day, default 9 without a settings row) the
    // payroll period is still the previous month — payroll for a month is
    // imported early in the following month — so the default tracks the same
    // stop-day rule as the selector.
    const now = new Date();
    const period = now.getDate() < 9
      ? new Date(now.getFullYear(), now.getMonth() - 1, 1)
      : now;
    const expectedMonth = `${period.getFullYear()}-${String(period.getMonth() + 1).padStart(2, '0')}`;
    expect(formData.get('forMonth')).toBe(expectedMonth);
  });

  it('shows the salary period selector defaulting to the payroll month', () => {
    render(<ImportPayrollDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText('Kỳ lương')).toBeInTheDocument();
    // Months render as a Select dropdown, not standalone MM/2026 buttons.
    expect(screen.queryByRole('button', { name: /\/2026/ })).not.toBeInTheDocument();
  });
});

describe('getMonthOptions', () => {
  it('offers previous + current while the default tail is open', () => {
    // Nov 8, stop day 9: requests for October are still open.
    const options = getMonthOptions(9, new Date(2026, 10, 8, 12, 0, 0));
    expect(options.map((opt) => opt.label)).toEqual(['Tháng 10/2026', 'Tháng 11/2026']);
    expect(options[0].value).toBe('2026-10');
    expect(options.find((opt) => opt.isCurrent)?.value).toBe('2026-11');
  });

  it('extends the tail to the configured stop day', () => {
    // Nov 10 with stop day 12: October stays importable, December never shows.
    const options = getMonthOptions(12, new Date(2026, 10, 10, 12, 0, 0));
    expect(options.map((opt) => opt.label)).toEqual(['Tháng 10/2026', 'Tháng 11/2026']);
    expect(options.some((opt) => opt.label.includes('/2027'))).toBe(false);
    expect(options.every((opt) => opt.value !== '2026-12')).toBe(true);
  });

  it('offers only the current month from the stop day onward', () => {
    // Nov 9 with the default stop day 9: the tail just closed.
    const options = getMonthOptions(9, new Date(2026, 10, 9, 12, 0, 0));
    expect(options.map((opt) => opt.label)).toEqual(['Tháng 11/2026']);
    expect(options[0].isCurrent).toBe(true);
    // Boundary: the stop day itself closes the tail too.
    expect(getMonthOptions(10, new Date(2026, 10, 10, 12, 0, 0)).map((opt) => opt.label)).toEqual([
      'Tháng 11/2026',
    ]);
  });
});
