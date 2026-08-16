import { render, screen } from '@testing-library/react';
import { Banknote } from 'lucide-react';
import { describe, expect, it, vi } from 'vitest';

import { KpiHeroCard } from './KpiHeroCard';

vi.mock('@/hooks/useCountUp', () => ({
  useCountUp: (value: number) => value,
}));

describe('KpiHeroCard', () => {
  it('renders every unit as secondary typography', () => {
    render(
      <KpiHeroCard
        label="Năng suất"
        value={24}
        unit="giờ/người"
        icon={Banknote}
        color="blue"
      />,
    );

    expect(screen.getByText('giờ/người')).toHaveClass('text-[0.58em]');
  });
});
