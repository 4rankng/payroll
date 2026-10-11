import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { ApprovalDecisionModal } from './ApprovalDecisionModal';
import type { ApprovalRequest } from '@/types/api/approval.types';

const request: ApprovalRequest = {
  id: 7,
  type: 'void_timesheet',
  type_label: 'đề xuất huỷ bảng công',
  status: 'pending',
  status_label: 'chờ duyệt',
  retry_count: 0,
  requester_id: 2,
  created_at: '2026-10-11T10:00:00+07:00',
};

const baseProps = {
  request,
  isPending: false,
  onClose: vi.fn(),
  onConfirm: vi.fn(),
};

describe('ApprovalDecisionModal', () => {
  it('reject mode requires guidance before confirming', () => {
    render(<ApprovalDecisionModal {...baseProps} open mode="reject" />);
    const confirm = screen.getByRole('button', { name: 'Từ chối' });
    expect(confirm).toBeDisabled();
    fireEvent.change(screen.getByLabelText(/Lý do \/ hướng dẫn/i), { target: { value: 'gửi mai, dữ liệu chưa đủ' } });
    expect(confirm).toBeEnabled();
    fireEvent.click(confirm);
    expect(baseProps.onConfirm).toHaveBeenCalledWith(7, 'gửi mai, dữ liệu chưa đủ');
  });

  it('approve mode confirms without a note', () => {
    render(<ApprovalDecisionModal {...baseProps} open mode="approve" />);
    fireEvent.click(screen.getByRole('button', { name: 'Duyệt' }));
    expect(baseProps.onConfirm).toHaveBeenCalledWith(7, '');
  });

  it('cancel mode warns the action is terminal', () => {
    render(<ApprovalDecisionModal {...baseProps} open mode="cancel" />);
    expect(screen.getByText(/không thử lại, không leo thang/i)).toBeInTheDocument();
  });

  it('renders nothing without a mode', () => {
    const { container } = render(<ApprovalDecisionModal {...baseProps} open mode={null} />);
    expect(container).toBeEmptyDOMElement();
  });
});
