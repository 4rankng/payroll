import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ImportPayrollDialog } from './ImportPayrollDialog';

const mutate = vi.fn();

vi.mock('@/hooks/api/useAdvancePayments', () => ({
  useImportFlexTemplate: () => ({
    mutate,
    isPending: false,
  }),
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
    // explicitly; the server only derives it when the field is absent. Before
    // day 9 the payroll period is still the previous month — payroll for a
    // month is imported early in the following month — so the default tracks
    // the same day-9 rule as the selector.
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
