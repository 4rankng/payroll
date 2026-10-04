import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from './select';

// Thin contract tests for the W6 UU restyle: engine stays Radix, surface
// vocabulary moves to the UU token bridge while call-site contracts
// (trigger sizing, popper viewport vars, item slots) stay intact.
describe('Select contract', () => {
  it('renders the trigger as a combobox, forwards the ref, and keeps the data-dense heights', () => {
    const ref = createRef<HTMLButtonElement>();
    render(
      <Select defaultValue="a">
        <SelectTrigger ref={ref} aria-label="Kỳ lương">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="a">Kỳ A</SelectItem>
        </SelectContent>
      </Select>,
    );

    const trigger = screen.getByRole('combobox', { name: 'Kỳ lương' });
    expect(ref.current).toBe(trigger);
    expect(trigger).toHaveClass('h-11', 'sm:h-9');
    expect(trigger).toHaveClass('rounded-lg', 'border-input', 'bg-card', 'shadow-xs');
  });

  it('applies the UU focus-visible outline treatment on the trigger', () => {
    render(
      <Select defaultValue="a">
        <SelectTrigger aria-label="Kỳ lương">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="a">Kỳ A</SelectItem>
        </SelectContent>
      </Select>,
    );

    expect(screen.getByRole('combobox', { name: 'Kỳ lương' })).toHaveClass(
      'outline-brand',
      'focus-visible:outline',
      'focus-visible:outline-2',
      'focus-visible:outline-offset-2',
    );
  });

  it('renders popper content with the UU surface and keeps the trigger-width sizing vars', () => {
    render(
      <Select open defaultValue="a">
        <SelectTrigger aria-label="Kỳ lương">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="a">Kỳ A</SelectItem>
        </SelectContent>
      </Select>,
    );

    const content = screen.getByRole('listbox');
    expect(content).toHaveClass(
      'bg-popover',
      'rounded-lg',
      'ring-1',
      'ring-utility-gray-200',
      'shadow-sm',
    );

    // Popper contract: the listbox spans at least the trigger width.
    const viewport = content.querySelector('[data-radix-select-viewport]');
    expect(viewport).not.toBeNull();
    expect(viewport).toHaveClass(
      'min-w-[var(--radix-select-trigger-width)]',
      'h-[var(--radix-select-trigger-height)]',
    );
  });

  it('styles items with the utility-gray-100 focus surface and 44px/compact floors', () => {
    render(
      <Select open defaultValue="a">
        <SelectTrigger aria-label="Kỳ lương">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="a">Kỳ A</SelectItem>
          <SelectItem value="b" disabled>
            Kỳ B
          </SelectItem>
          <SelectSeparator data-testid="sep" />
          <SelectItem value="c">Kỳ C</SelectItem>
        </SelectContent>
      </Select>,
    );

    const item = screen.getByRole('option', { name: 'Kỳ A' });
    expect(item).toHaveClass('focus:bg-utility-gray-100', 'min-h-11', 'sm:min-h-8');
    expect(item.querySelector('svg')).not.toBeNull();

    const disabled = screen.getByRole('option', { name: 'Kỳ B' });
    expect(disabled).toHaveAttribute('data-disabled');
    expect(disabled).toHaveClass(
      'data-[disabled]:pointer-events-none',
      'data-[disabled]:opacity-50',
    );

    expect(screen.getByTestId('sep')).toHaveClass('bg-utility-gray-200');
  });

  it('lets caller classes override the content surface through the merge', () => {
    render(
      <Select open defaultValue="a">
        <SelectTrigger aria-label="Kỳ lương">
          <SelectValue />
        </SelectTrigger>
        <SelectContent className="rounded-none">
          <SelectItem value="a">Kỳ A</SelectItem>
        </SelectContent>
      </Select>,
    );

    expect(screen.getByRole('listbox')).toHaveClass('rounded-none');
    expect(screen.getByRole('listbox')).not.toHaveClass('rounded-lg');
  });
});
