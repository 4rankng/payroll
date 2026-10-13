import { fireEvent, render, screen } from '@testing-library/react';
import { Activity, WalletCards } from 'lucide-react';
import { describe, expect, it, vi } from 'vitest';

import {
  DashboardActivityPanel,
  DashboardMetricStrip,
  DashboardPriorityList,
} from './DashboardOverview';

describe('DashboardOverview density', () => {
  it('uses compact metric cells on desktop', () => {
    render(
      <DashboardMetricStrip
        items={[
          {
            label: 'Đã trả kỳ này',
            value: '1.026.336.457 ₫',
            context: 'Tổng đã trả',
            icon: WalletCards,
            tone: 'primary',
          },
        ]}
      />,
    );

    expect(screen.getByText('Đã trả kỳ này').parentElement?.parentElement).toHaveClass(
      'px-3',
      'py-3',
      'sm:px-4',
    );
  });

  it('keeps touch-safe actions while reducing desktop row height', () => {
    const { rerender } = render(
      <DashboardPriorityList
        items={[
          {
            title: 'Bảng công chờ duyệt',
            value: '3',
            detail: 'Cần xử lý',
            statusLabel: 'Cần duyệt',
            icon: Activity,
            tone: 'warning',
            onClick: vi.fn(),
          },
        ]}
        isRefreshing={false}
        onRefresh={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Làm mới số liệu' })).toHaveClass(
      'min-h-11',
      'sm:min-h-9',
    );
    expect(screen.getByRole('button', { name: /Bảng công chờ duyệt/ })).toHaveClass(
      'min-h-[68px]',
      'sm:min-h-[60px]',
    );

    rerender(
      <DashboardActivityPanel
        monthLabel="08/2026"
        totalActive={62}
        items={[{ label: 'Lương tuần', value: 25, onClick: vi.fn() }]}
      />,
    );

    expect(screen.getByRole('button', { name: /Lương tuần/ })).toHaveClass(
      'min-h-11',
      'sm:min-h-9',
    );
  });

  it('collapses per-project rows behind a toggle and hides the group when empty', () => {
    const { rerender } = render(
      <DashboardActivityPanel
        monthLabel="10/2026"
        totalActive={158}
        items={[{ label: 'Lương tuần', value: 73, onClick: vi.fn() }]}
        projectItems={[
          { label: 'Nha may VERICO', value: 120 },
          { label: 'San xuat xa phong', value: 38 },
        ]}
      />,
    );

    const toggle = screen.getByRole('button', { name: /Theo dự án/ });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('Nha may VERICO')).not.toBeInTheDocument();

    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('Nha may VERICO')).toBeInTheDocument();
    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText('38')).toBeInTheDocument();

    rerender(
      <DashboardActivityPanel
        monthLabel="10/2026"
        totalActive={158}
        items={[{ label: 'Lương tuần', value: 73, onClick: vi.fn() }]}
        projectItems={[]}
      />,
    );

    expect(screen.queryByRole('button', { name: /Theo dự án/ })).not.toBeInTheDocument();
  });
});
