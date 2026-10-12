import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { TimesheetMonthSelector } from './TimesheetMonthSelector';

describe('TimesheetMonthSelector', () => {
  it('keeps every navigation control at one height on all device sizes', () => {
    render(
      <TimesheetMonthSelector
        value="2026-07"
        onChange={vi.fn()}
      />,
    );

    for (const name of ['Tháng trước', 'Tháng sau']) {
      const el = screen.getByRole('button', { name });
      expect(el).toHaveClass('h-9', 'w-9');
      expect(el.className.split(/\s+/).filter((c) => /(^|:)h-/.test(c))).toEqual(['h-9']);
    }
  });
});
