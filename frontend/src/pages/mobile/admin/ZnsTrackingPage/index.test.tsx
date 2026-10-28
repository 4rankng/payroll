import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import ZnsTrackingPageMobile from './index';
import type { BackendZnsMessagesResponse } from '@/types/api/zns.types';

const mocks = vi.hoisted(() => ({
  response: {
    status: 'success',
    message: '',
    data: [],
    pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 0 },
  } as BackendZnsMessagesResponse,
}));

vi.mock('@/hooks/api/useZnsMessages', () => ({
  useZnsMessages: () => ({ data: mocks.response, isLoading: false, isError: false, refetch: vi.fn() }),
}));

vi.mock('@/contexts', () => ({
  useAppState: () => ({ setPageTitle: vi.fn() }),
}));

beforeEach(() => {
  mocks.response = {
    status: 'success',
    message: '',
    data: [
      {
        id: 1,
        created_at: '2026-10-28T09:30:00+07:00',
        feature: 'self_checkin',
        template_id: 'tpl-checkin-01',
        phone: '0912345678',
        status: 'failed',
        provider_msg_id: 'zns-msg-009',
        provider_code: -115,
        error_msg: 'Quota exceeded for today',
        duration_ms: 120,
      },
    ],
    pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 1 },
  };
});

describe('ZnsTrackingPageMobile', () => {
  it('renders each ZNS as a card with full visible fields', () => {
    const { container } = render(<ZnsTrackingPageMobile />);

    // One card per ZNS — no table markup on mobile.
    expect(container.querySelectorAll('article')).toHaveLength(1);
    expect(container.querySelector('table')).toBeNull();

    expect(screen.getByText('0912345678')).toBeInTheDocument();
    expect(screen.getByText('zns-msg-009')).toBeInTheDocument();
    expect(screen.getByText('Tự chấm công')).toBeInTheDocument();
    expect(screen.getByText('Lỗi')).toBeInTheDocument();
    // Full error text visible (doctrine: no truncate).
    expect(screen.getByText(/Quota exceeded for today/)).toBeInTheDocument();
  });
});
