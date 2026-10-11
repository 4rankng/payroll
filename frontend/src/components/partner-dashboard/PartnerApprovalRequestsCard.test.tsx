import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

const listState = vi.hoisted(() => ({ items: [] as unknown[], isLoading: true }));
vi.mock('@/hooks/api/useApprovals', () => ({
  useApprovalsList: () => ({
    data: { items: listState.items, pagination: null },
    isLoading: listState.isLoading,
  }),
}));

import { PartnerApprovalRequestsCard } from './PartnerApprovalRequestsCard';

describe('PartnerApprovalRequestsCard', () => {
  it('renders nothing while loading (no dead card)', () => {
    listState.isLoading = true;
    listState.items = [];
    const { container } = render(<PartnerApprovalRequestsCard />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders nothing when the partner has no requests (non-zero row law)', () => {
    listState.isLoading = false;
    listState.items = [];
    const { container } = render(<PartnerApprovalRequestsCard />);
    expect(container).toBeEmptyDOMElement();
  });

  it('lists own requests with status and decision note', () => {
    listState.isLoading = false;
    listState.items = [
      {
        id: 3,
        type: 'void_timesheet',
        type_label: 'đề xuất huỷ bảng công',
        status: 'rescheduled',
        status_label: 'đã lên lịch thử lại',
        retry_count: 1,
        decision_note: 'gửi mai, dữ liệu chưa đủ',
        requester_id: 9,
        created_at: '2026-10-11T09:00:00+07:00',
      },
    ];
    render(<PartnerApprovalRequestsCard />);
    expect(screen.getByText(/#3 · đề xuất huỷ bảng công/)).toBeInTheDocument();
    expect(screen.getByText('đã lên lịch thử lại')).toBeInTheDocument();
    expect(screen.getByText(/gửi mai, dữ liệu chưa đủ/)).toBeInTheDocument();
  });
});
