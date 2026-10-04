import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Input } from './input';

describe('Input contract', () => {
  it('renders a native input and forwards the ref to the DOM node (react-hook-form register)', () => {
    const ref = createRef<HTMLInputElement>();
    render(<Input ref={ref} aria-label="Tiền lương" />);

    const input = screen.getByLabelText('Tiền lương');
    expect(input).toBeInstanceOf(HTMLInputElement);
    expect(ref.current).toBe(input);
  });

  it('keeps the Vietnamese locale attribute on date inputs only', () => {
    const { rerender } = render(<Input type="date" aria-label="Ngày công" />);
    expect(screen.getByLabelText('Ngày công')).toHaveAttribute('lang', 'vi-VN');

    rerender(<Input type="text" aria-label="Ngày công" />);
    expect(screen.getByLabelText('Ngày công')).not.toHaveAttribute('lang');
  });

  it('renders every variant with its UU surface classes', () => {
    const { rerender } = render(<Input variant="default" aria-label="Ô nhập" />);
    expect(screen.getByLabelText('Ô nhập')).toHaveClass('bg-card', 'border-input', 'rounded-lg');

    rerender(<Input variant="filled" aria-label="Ô nhập" />);
    expect(screen.getByLabelText('Ô nhập')).toHaveClass('bg-muted', 'border-0');

    rerender(<Input variant="outlined" aria-label="Ô nhập" />);
    expect(screen.getByLabelText('Ô nhập')).toHaveClass('bg-transparent', 'border-2', 'border-input');
  });

  it('lets caller classes override variant classes through the merge', () => {
    render(<Input className="rounded-none" aria-label="Ô nhập" />);

    const input = screen.getByLabelText('Ô nhập');
    expect(input).toHaveClass('rounded-none');
    expect(input).not.toHaveClass('rounded-lg');
  });

  it('applies the UU focus-visible outline treatment', () => {
    render(<Input aria-label="Ô nhập" />);

    expect(screen.getByLabelText('Ô nhập')).toHaveClass(
      'outline-brand',
      'focus-visible:outline',
      'focus-visible:outline-2',
      'focus-visible:outline-offset-2',
    );
  });

  it('styles disabled and invalid states with UU tokens', () => {
    render(<Input disabled aria-invalid="true" aria-label="Ô nhập" />);

    const input = screen.getByLabelText('Ô nhập');
    expect(input).toBeDisabled();
    expect(input).toHaveClass('disabled:cursor-not-allowed', 'disabled:opacity-50');
    expect(input).toHaveClass(
      'aria-[invalid=true]:border-utility-error-300',
      'aria-[invalid=true]:focus-visible:outline-error',
    );
  });

  it('keeps the data-dense control heights (44px mobile floor, compact desktop)', () => {
    render(<Input aria-label="Ô nhập" />);

    expect(screen.getByLabelText('Ô nhập')).toHaveClass('h-11', 'sm:h-9');
  });
});
