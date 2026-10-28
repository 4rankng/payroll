import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import ZnsTrackingPage from './index';
import type { BackendZnsMessagesResponse, ZnsMessagesQueryParams } from '@/types/api/zns.types';

const mocks = vi.hoisted(() => ({
  capturedParams: [] as ZnsMessagesQueryParams[],
  response: {
    status: 'success',
    message: '',
    data: [],
    pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 0 },
  } as BackendZnsMessagesResponse,
  isError: false,
  isLoading: false,
  refetch: vi.fn(),
}));

vi.mock('@/hooks/api/useZnsMessages', () => ({
  useZnsMessages: (params: ZnsMessagesQueryParams) => {
    mocks.capturedParams.push(params);
    return {
      data: mocks.response,
      isLoading: mocks.isLoading,
      isError: mocks.isError,
      refetch: mocks.refetch,
    };
  },
}));

vi.mock('@/contexts', () => ({
  useAppState: () => ({ setPageTitle: vi.fn() }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.capturedParams = [];
  mocks.isError = false;
  mocks.isLoading = false;
  mocks.response = {
    status: 'success',
    message: '',
    data: [
      {
        id: 1,
        created_at: '2026-10-28T09:30:00+07:00',
        feature: 'salary_notification',
        template_id: 'tpl-salary-01',
        phone: '0912345678',
        status: 'sent',
        provider_msg_id: 'zns-msg-001',
        provider_code: 0,
        error_msg: '',
        duration_ms: 220,
      },
      {
        id: 2,
        created_at: '2026-10-28T08:00:00+07:00',
        feature: 'password_reset',
        template_id: 'tpl-reset-02',
        phone: '0987654321',
        status: 'failed',
        provider_msg_id: '',
        provider_code: -118,
        error_msg: 'Phone has no linked Zalo account',
        duration_ms: 90,
      },
    ],
    pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 2 },
  };
});

describe('ZnsTrackingPage', () => {
  it('renders every tracking row without truncating the error', () => {
    const { container } = render(<ZnsTrackingPage />);

    expect(screen.getByText('0912345678')).toBeInTheDocument();
    expect(screen.getByText('0987654321')).toBeInTheDocument();
    expect(screen.getByText('zns-msg-001')).toBeInTheDocument();
    // Full error text visible (doctrine: no truncate).
    expect(screen.getByText(/Phone has no linked Zalo account/)).toBeInTheDocument();
    // Status + feature labels appear in the table body (not the filter chips).
    const table = container.querySelector('tbody')!;
    expect(within(table).getByText('Đã gửi')).toBeInTheDocument();
    expect(within(table).getByText('Lỗi')).toBeInTheDocument();
    expect(within(table).getByText('Thông báo lương')).toBeInTheDocument();
    expect(within(table).getByText('Đặt lại mật khẩu')).toBeInTheDocument();
  });

  it('sends the status filter when a status chip is toggled', () => {
    render(<ZnsTrackingPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Lỗi' }));

    const last = mocks.capturedParams[mocks.capturedParams.length - 1];
    expect(last.status).toEqual(['failed']);
    expect(last.page).toBe(1);
  });

  it('sends the feature filter from the dropdown', () => {
    render(<ZnsTrackingPage />);

    fireEvent.click(screen.getByRole('combobox', { name: 'Lọc tính năng' }));
    fireEvent.click(within(document.body).getByRole('option', { name: /Thông báo lương/ }));

    const last = mocks.capturedParams[mocks.capturedParams.length - 1];
    expect(last.feature).toBe('salary_notification');
  });

  it('sends the free-text query and date bounds', () => {
    render(<ZnsTrackingPage />);

    fireEvent.change(screen.getByRole('searchbox', { name: 'Tìm SĐT hoặc mã ZNS' }), {
      target: { value: '09123' },
    });
    fireEvent.change(screen.getByLabelText('Từ ngày'), { target: { value: '2026-10-01' } });
    fireEvent.change(screen.getByLabelText('Đến ngày'), { target: { value: '2026-10-28' } });

    const last = mocks.capturedParams[mocks.capturedParams.length - 1];
    expect(last.q).toBe('09123');
    expect(last.fromDate).toBe('2026-10-01');
    expect(last.toDate).toBe('2026-10-28');
  });

  it('combines status and feature filters', () => {
    render(<ZnsTrackingPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Đã gửi' }));
    fireEvent.click(screen.getByRole('combobox', { name: 'Lọc tính năng' }));
    fireEvent.click(within(document.body).getByRole('option', { name: /Đặt lại mật khẩu/ }));

    const last = mocks.capturedParams[mocks.capturedParams.length - 1];
    expect(last.status).toEqual(['sent']);
    expect(last.feature).toBe('password_reset');
  });
});
