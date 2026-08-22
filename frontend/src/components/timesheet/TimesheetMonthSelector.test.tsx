import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { TimesheetMonthSelector } from './TimesheetMonthSelector';

describe('TimesheetMonthSelector', () => {
  it('keeps 44px mobile navigation targets and compacts them on desktop', () => {
    render(
      <TimesheetMonthSelector
        value="2026-07"
        onChange={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Tháng trước' })).toHaveClass(
      'h-11',
      'w-11',
      'sm:h-9',
      'sm:w-9',
    );
    expect(screen.getByRole('button', { name: 'Tháng sau' })).toHaveClass(
      'h-11',
      'w-11',
      'sm:h-9',
      'sm:w-9',
    );
  });
});
