import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, vi } from 'vitest';
import { RejectUnpaidTimesheetsDialog } from './RejectUnpaidTimesheetsDialog';

const mutateAsync = vi.fn();

vi.mock('@/hooks/api/useProjects', () => ({
  useAllProjects: () => ({
    data: [{ id: 7, name: 'Dự án Yusen', code: 'YV001' }],
    isLoading: false,
  }),
}));

vi.mock('@/hooks/api/useTimesheets', () => ({
  useRejectUnpaidTimesheets: () => ({ mutateAsync, isPending: false }),
}));

describe('RejectUnpaidTimesheetsDialog', () => {
  beforeEach(() => {
    mutateAsync.mockReset();
    mutateAsync.mockResolvedValue({ rejected_count: 3 });
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.setSystemTime(new Date('2026-07-31T12:00:00+07:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('requires a reason and submits the confirmed inclusive project range', async () => {
    const onOpenChange = vi.fn();
    render(
      <RejectUnpaidTimesheetsDialog
        open
        onOpenChange={onOpenChange}
        initialProjectId={7}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Kiểm tra phạm vi' }));
    expect(screen.getByRole('alert')).toHaveTextContent('Vui lòng nhập lý do loại.');

    fireEvent.change(screen.getByLabelText('Lý do loại'), {
      target: { value: '  Sai dữ liệu chấm công  ' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Kiểm tra phạm vi' }));

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Xác nhận phạm vi loại' })).toHaveFocus();
    });
    expect(screen.getByText('Dự án Yusen')).toBeInTheDocument();
    expect(screen.getByText('2026-07-01 – 2026-07-31')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Xác nhận loại' }));

    await waitFor(() => {
      expect(mutateAsync).toHaveBeenCalledWith({
        project_id: 7,
        from_date: '2026-07-01',
        to_date: '2026-07-31',
        rejection_reason: 'Sai dữ liệu chấm công',
      });
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it('opens a calendar when either displayed date is clicked', () => {
    render(
      <RejectUnpaidTimesheetsDialog
        open
        onOpenChange={vi.fn()}
        initialProjectId={7}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /Mở lịch chọn ngày bắt đầu/ }));
    expect(screen.getByRole('grid')).toBeInTheDocument();

    fireEvent.keyDown(document.activeElement ?? document.body, { key: 'Escape' });
    fireEvent.click(screen.getByRole('button', { name: /Mở lịch chọn ngày kết thúc/ }));
    expect(screen.getByRole('grid')).toBeInTheDocument();
  });

  it('restores focus to the reason field when returning from confirmation', async () => {
    render(
      <RejectUnpaidTimesheetsDialog
        open
        onOpenChange={vi.fn()}
        initialProjectId={7}
      />,
    );

    const reason = screen.getByLabelText('Lý do loại');
    fireEvent.change(reason, { target: { value: 'Sai dữ liệu chấm công' } });
    fireEvent.click(screen.getByRole('button', { name: 'Kiểm tra phạm vi' }));
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Xác nhận phạm vi loại' })).toHaveFocus();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Quay lại' }));

    await waitFor(() => expect(screen.getByLabelText('Lý do loại')).toHaveFocus());
  });

  it('retains the confirmed form values when submission fails', async () => {
    mutateAsync.mockRejectedValueOnce(new Error('request failed'));
    const onOpenChange = vi.fn();
    render(
      <RejectUnpaidTimesheetsDialog
        open
        onOpenChange={onOpenChange}
        initialProjectId={7}
      />,
    );

    fireEvent.change(screen.getByLabelText('Lý do loại'), {
      target: { value: 'Sai dữ liệu chấm công' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Kiểm tra phạm vi' }));
    fireEvent.click(screen.getByRole('button', { name: 'Xác nhận loại' }));

    await waitFor(() => expect(mutateAsync).toHaveBeenCalledOnce());
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByText('Sai dữ liệu chấm công')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Xác nhận loại' })).toBeInTheDocument();
  });
});
