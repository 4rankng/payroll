import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { GroupedStatCard } from './GroupedStatCard';

vi.mock('@/hooks/useCountUp', () => ({
  useCountUp: (value: number) => value,
}));

describe('GroupedStatCard', () => {
  it('collapses the strip behind the title toggle when defaultCollapsed', () => {
    render(
      <GroupedStatCard
        title="Hoạt động theo dự án"
        stats={[
          { label: 'LGD', value: 86 },
          { label: 'Samsung SDS', value: 16 },
        ]}
        defaultCollapsed
      />,
    );

    const toggle = screen.getByRole('button', { name: /Hoạt động theo dự án/ });
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('LGD')).not.toBeInTheDocument();

    fireEvent.click(toggle);

    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('LGD')).toBeInTheDocument();
    expect(screen.getByText('Samsung SDS')).toBeInTheDocument();
  });

  it('renders an interactive stat as an accessible toggle button', () => {
    const onClick = vi.fn();

    render(
      <GroupedStatCard
        title="Cần xử lý"
        stats={[
          {
            label: 'Đã duyệt',
            value: 3,
            variant: 'accent',
            isPressed: true,
            onClick,
          },
        ]}
      />,
    );

    const approvedFilter = screen.getByRole('button', { name: /Đã duyệt/ });
    expect(approvedFilter).toHaveAttribute('aria-pressed', 'true');

    fireEvent.click(approvedFilter);
    expect(onClick).toHaveBeenCalledOnce();
  });
});
