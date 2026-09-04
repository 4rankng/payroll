import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { usePartnerImportHistory } from '@/hooks/timesheet/usePartnerImportHistory';
import { UploadHistorySheet } from './UploadHistorySheet';

vi.mock('@/hooks/timesheet/usePartnerImportHistory', () => ({
  usePartnerImportHistory: vi.fn(),
}));

describe('UploadHistorySheet', () => {
  beforeEach(() => {
    vi.mocked(usePartnerImportHistory).mockReturnValue({
      data: {
        status: 'success',
        message: '',
        data: [{
          id: 91,
          project_id: 12,
          uploaded_by: 7,
          original_name: 'bang-cham-cong.xlsx',
          for_month: '2026-07',
          status: 'completed',
          total_rows: 2,
          created_count: 1,
          skipped_count: 0,
          error_count: 1,
          error_detail: JSON.stringify([{
            row: 2,
            employee: 'employee_id=91',
            reason: 'OnePay provider failed: timeout',
          }]),
          processed_at: '2026-07-25T06:00:00Z',
          created_at: '2026-07-25T05:59:00Z',
        }],
        pagination: {
          page: 1,
          pageSize: 10,
          totalPages: 1,
          totalRecords: 1,
        },
      },
      isLoading: false,
    } as ReturnType<typeof usePartnerImportHistory>);
  });

  it('marks completed imports with row errors as partial and hides technical details', () => {
    render(
      <UploadHistorySheet
        open
        onClose={vi.fn()}
        projects={[{ id: 12, name: 'Dự án A' }]}
      />,
    );

    expect(screen.getByText('Hoàn tất một phần')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Xem 1 lỗi' }));
    // Grouped rendering: one line per (employee, reason) with the affected
    // rows folded into a detail suffix instead of "Dòng N:" prefixes.
    expect(screen.getByText('Không thể xử lý dòng dữ liệu này')).toBeInTheDocument();
    expect(screen.getByText(/\(dòng 2\)/)).toBeInTheDocument();
    expect(screen.queryByText(/OnePay|provider|failed|employee_id/i)).not.toBeInTheDocument();
  });
});
