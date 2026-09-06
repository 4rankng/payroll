import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { AdBanner } from '@/types/api/ad-banner.types';

import { EmployeeAdSheet } from './EmployeeAdSheet';

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: () => true,
}));

const banner: AdBanner = {
  id: 1,
  title: 'TING TING SOFTWARE SOLUTIONS xin thông báo',
  body: 'Công nhân dự án LG Display có thể chấm công tự động và ứng lương ngay.',
  bullets: [
    'Chấm công tự động, chính xác từng ca làm',
    'Ứng lương theo công đã làm, tối đa 70%',
    'Nhận tiền nhanh chóng, chủ động chi tiêu',
  ],
  ctas: [
    { label: 'Gọi hotline', type: 'phone', value: '0914827988' },
    { label: 'Zalo', type: 'zalo', value: 'https://zalo.me/g/example' },
  ],
  footer: 'Ting Ting Software Solutions — Đồng hành cùng người lao động.',
  targetProjectIds: [12],
  priority: 0,
  startsAt: '2026-09-06T00:00:00Z',
  endsAt: '2026-10-06T00:00:00Z',
  isActive: true,
  createdAt: '2026-09-06T00:00:00Z',
  updatedAt: '2026-09-06T00:00:00Z',
};

describe('EmployeeAdSheet', () => {
  it('renders the full campaign content', () => {
    render(<EmployeeAdSheet open onClose={vi.fn()} banner={banner} />);

    expect(screen.getByText(banner.title)).toBeInTheDocument();
    expect(screen.getByText(banner.body)).toBeInTheDocument();
    banner.bullets.forEach((bullet) => {
      expect(screen.getByText(bullet)).toBeInTheDocument();
    });
    expect(screen.getByRole('button', { name: 'Gọi hotline' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Zalo' })).toBeInTheDocument();
    expect(screen.getByText(banner.footer)).toBeInTheDocument();
  });

  it('invokes onClose when the employee dismisses', () => {
    const onClose = vi.fn();
    render(<EmployeeAdSheet open onClose={onClose} banner={banner} />);

    fireEvent.click(screen.getByRole('button', { name: 'Đóng' }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('forwards CTA taps with the CTA and its index', () => {
    const onCTAClick = vi.fn();
    render(<EmployeeAdSheet open onClose={vi.fn()} banner={banner} onCTAClick={onCTAClick} />);

    fireEvent.click(screen.getByRole('button', { name: 'Zalo' }));
    expect(onCTAClick).toHaveBeenCalledWith(banner.ctas[1], 1);
  });
});
