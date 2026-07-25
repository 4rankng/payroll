import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { GroupedStatCard } from './GroupedStatCard';

vi.mock('@/hooks/useCountUp', () => ({
  useCountUp: (value: number) => value,
}));

describe('GroupedStatCard', () => {
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
