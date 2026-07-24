import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { BulkTransferExportDialog } from './BulkTransferExportDialog';

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: () => false,
}));

vi.mock('@/hooks/api/useProjects', () => ({
  useProjects: () => ({
    data: {
      data: [
        { id: 42, code: 'DA-42', name: 'Dự án 42' },
        { id: 84, code: 'DA-84', name: 'Dự án 84' },
      ],
    },
  }),
}));

vi.mock('@/hooks/api/useEmployees', () => ({
  useEmployees: () => ({ data: { data: [] } }),
}));

vi.mock('@/hooks/api/usePayrolls', () => ({
  useAutoBulkTransferStatus: () => ({ data: undefined }),
}));

describe('BulkTransferExportDialog OnePay mode', () => {
  it('submits a weekly range without exposing the monthly cohort', () => {
    const handleExport = vi.fn();

    render(
      <BulkTransferExportDialog
        open
        onOpenChange={vi.fn()}
        onExport={handleExport}
        mode="onepay"
        preselectedProjectIds={[42]}
      />,
    );

    expect(screen.getAllByText('Chọn kỳ lương tuần để tạo file chuyển tiền OnePay')).toHaveLength(2);
    expect(screen.queryByText('Lương tháng')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Tạo file OnePay' }));

    expect(handleExport).toHaveBeenCalledTimes(1);
    expect(handleExport).toHaveBeenCalledWith(expect.objectContaining({
      fromDate: expect.any(String),
      toDate: expect.any(String),
      project_ids: [42],
    }));
    expect(handleExport.mock.calls[0][0]).not.toHaveProperty('for_month');
  });

  it('locks period controls while the OnePay file is being created', () => {
    render(
      <BulkTransferExportDialog
        open
        onOpenChange={vi.fn()}
        onExport={vi.fn()}
        isLoading
        mode="onepay"
      />,
    );

    expect(screen.getByRole('button', { name: 'Đang tạo file...' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Tùy chỉnh' })).toBeDisabled();
  });
});
