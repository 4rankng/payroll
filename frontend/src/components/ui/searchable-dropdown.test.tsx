import { fireEvent, render, screen, within } from '@testing-library/react';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { SearchableDropdown } from './searchable-dropdown';

const useIsMobileMock = vi.hoisted(() => vi.fn());

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: useIsMobileMock,
}));

const options = [
  { value: 'yv001', label: 'Yusen (YV001)' },
  { value: 'lgdisplay', label: 'LGDISPLAY (LGDISPLAY Lương Tuần)' },
];

const renderDropdown = (onValueChange = vi.fn()) => render(
  <div role="dialog" aria-label="Bộ lọc" data-testid="filter-sheet">
    <SearchableDropdown
      value="all"
      onValueChange={onValueChange}
      options={options}
      placeholder="Tất cả dự án"
      searchPlaceholder="Tìm dự án..."
      emptyMessage="Không tìm thấy dự án nào."
      allOption={{ value: 'all', label: 'Tất cả dự án' }}
      mobileTitle="Chọn dự án"
    />
  </div>
);

describe('SearchableDropdown', () => {
  beforeAll(() => {
    vi.stubGlobal('ResizeObserver', class {
      observe() {}
      unobserve() {}
      disconnect() {}
    });
    Element.prototype.scrollIntoView = vi.fn();
  });

  beforeEach(() => {
    useIsMobileMock.mockReturnValue(false);
  });

  it('opens an in-place mobile picker with native vertical scrolling', () => {
    useIsMobileMock.mockReturnValue(true);
    renderDropdown();

    const trigger = screen.getByRole('combobox');
    expect(trigger).toHaveClass('min-h-11');
    fireEvent.click(trigger);

    const picker = within(screen.getByTestId('filter-sheet')).getByRole('region', { name: 'Chọn dự án' });
    expect(picker).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Quay lại bộ lọc' })).toHaveClass('size-11');
    expect(screen.getByRole('button', { name: 'Quay lại bộ lọc' })).toHaveFocus();
    expect(screen.getByPlaceholderText('Tìm dự án...')).not.toHaveFocus();
    expect(screen.getByRole('listbox')).toHaveClass(
      '!max-h-none',
      'touch-pan-y',
      'overscroll-contain',
      '[-webkit-overflow-scrolling:touch]',
    );

    fireEvent.keyDown(picker, { key: 'Tab', shiftKey: true });
    expect(screen.getByPlaceholderText('Tìm dự án...')).toHaveFocus();
  });

  it('selects a mobile option, closes the picker, and restores trigger focus', () => {
    useIsMobileMock.mockReturnValue(true);
    const onValueChange = vi.fn();
    renderDropdown(onValueChange);

    const trigger = screen.getByRole('combobox');
    fireEvent.click(trigger);
    fireEvent.click(screen.getByText('Yusen (YV001)'));

    expect(onValueChange).toHaveBeenCalledWith('yv001');
    expect(screen.queryByRole('region', { name: 'Chọn dự án' })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('returns to the filter sheet on Escape', () => {
    useIsMobileMock.mockReturnValue(true);
    renderDropdown();

    const trigger = screen.getByRole('combobox');
    fireEvent.click(trigger);
    fireEvent.keyDown(screen.getByRole('region', { name: 'Chọn dự án' }), { key: 'Escape' });

    expect(screen.queryByRole('region', { name: 'Chọn dự án' })).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('retains the anchored popover and clean keyboard focus contract on desktop', () => {
    renderDropdown();
    const trigger = screen.getByRole('combobox');
    trigger.focus();
    fireEvent.keyDown(trigger, { key: 'Enter' });
    fireEvent.click(trigger, { detail: 0 });

    const popover = screen.getAllByRole('dialog').at(-1);
    const searchInput = screen.getByPlaceholderText('Tìm dự án...');
    expect(popover).toHaveClass('w-[var(--radix-popover-trigger-width)]');
    expect(popover).not.toHaveAttribute('data-vaul-drawer');
    expect(screen.getByRole('listbox')).toHaveClass('max-h-[300px]');
    expect(searchInput).toHaveFocus();
    expect(searchInput).toHaveClass('!outline-none');
    expect(searchInput.parentElement).toHaveClass(
      'focus-within:ring-2',
      'focus-within:ring-inset',
      'focus-within:ring-ring',
    );
  });
});
