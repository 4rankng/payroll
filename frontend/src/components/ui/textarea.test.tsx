import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Textarea } from './textarea';

describe('Textarea contract', () => {
  it('renders a native textarea and forwards the ref to the DOM node', () => {
    const ref = createRef<HTMLTextAreaElement>();
    render(<Textarea ref={ref} aria-label="Ghi chú" />);

    const textarea = screen.getByLabelText('Ghi chú');
    expect(textarea).toBeInstanceOf(HTMLTextAreaElement);
    expect(ref.current).toBe(textarea);
  });

  it('keeps the data-dense height floor and UU surface classes', () => {
    render(<Textarea aria-label="Ghi chú" />);

    expect(screen.getByLabelText('Ghi chú')).toHaveClass(
      'min-h-[80px]',
      'rounded-lg',
      'border-input',
      'bg-card',
      'shadow-xs',
    );
  });

  it('lets caller classes override the base classes through the merge', () => {
    render(<Textarea className="rounded-none" aria-label="Ghi chú" />);

    const textarea = screen.getByLabelText('Ghi chú');
    expect(textarea).toHaveClass('rounded-none');
    expect(textarea).not.toHaveClass('rounded-lg');
  });

  it('styles disabled and invalid states with UU tokens', () => {
    render(<Textarea disabled aria-invalid="true" aria-label="Ghi chú" />);

    const textarea = screen.getByLabelText('Ghi chú');
    expect(textarea).toBeDisabled();
    expect(textarea).toHaveClass('disabled:cursor-not-allowed', 'disabled:opacity-50');
    expect(textarea).toHaveClass(
      'aria-[invalid=true]:border-utility-error-300',
      'aria-[invalid=true]:focus-visible:outline-error',
    );
  });
});
