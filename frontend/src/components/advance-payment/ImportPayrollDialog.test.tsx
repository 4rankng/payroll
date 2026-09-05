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
    // The salary month is derived server-side from the upload date — the
    // client no longer sends it.
    expect(formData.get('forMonth')).toBeNull();
  });

  it('shows the derived salary period instead of a month selector', () => {
    render(<ImportPayrollDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText('Kỳ lương')).toBeInTheDocument();
    // No month chips to select.
    expect(screen.queryByRole('button', { name: /\/2026/ })).not.toBeInTheDocument();
  });
});
