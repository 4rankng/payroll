import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { TimesheetMonthSelector } from './TimesheetMonthSelector';

describe('TimesheetMonthSelector', () => {
  it('keeps previous and next month controls at least 44px', () => {
    render(
      <TimesheetMonthSelector
        value="2026-07"
        onChange={vi.fn()}
      />,
    );

    expect(screen.getByRole('button', { name: 'Tháng trước' })).toHaveClass('h-11', 'w-11');
    expect(screen.getByRole('button', { name: 'Tháng sau' })).toHaveClass('h-11', 'w-11');
  });
});
