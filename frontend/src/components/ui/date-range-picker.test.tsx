import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { DateRangePicker } from './date-range-picker';

describe('DateRangePicker', () => {
  it('keeps both date inputs at least 44px in the mobile variant', () => {
    render(
      <DateRangePicker
        variant="mobile"
        onStartDateChange={vi.fn()}
        onEndDateChange={vi.fn()}
      />,
    );

    expect(screen.getByLabelText('Ngày bắt đầu')).toHaveClass('h-11');
    expect(screen.getByLabelText('Ngày kết thúc')).toHaveClass('h-11');
  });
});
