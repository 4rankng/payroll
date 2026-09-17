import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { MonthPicker } from './month-picker';

describe('MonthPicker', () => {
  it('shows the selected payroll month and emits the unchanged API period format', () => {
    const onChange = vi.fn();
    render(<MonthPicker value="2026-07" onChange={onChange} />);
    expect(screen.getByRole('button', { name: 'Tháng 7 năm 2026' })).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByRole('button', { name: 'Tháng 2 năm 2026' }));
    expect(onChange).toHaveBeenCalledWith('2026-02');
  });

  it('can choose a month in a different year without changing the period during navigation', () => {
    const onChange = vi.fn();
    render(<MonthPicker value="2026-12" onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: 'Năm sau' }));
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Tháng 1 năm 2027' }));
    expect(onChange).toHaveBeenCalledWith('2027-01');
    fireEvent.click(screen.getByRole('button', { name: 'Năm trước' }));
    expect(screen.getByRole('button', { name: 'Tháng 12 năm 2026' })).toHaveAttribute('aria-pressed', 'true');
  });

  it('does not select an arbitrary month for an all-period filter', () => {
    render(<MonthPicker value="all" onChange={vi.fn()} />);
    expect(screen.getAllByRole('button', { pressed: false })).toHaveLength(12);
  });
});
