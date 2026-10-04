import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';
import { Label } from './label';

describe('Label contract', () => {
  it('associates with its control and keeps the compact UU label scale', () => {
    render(
      <>
        <Label htmlFor="ho-ten">Họ và tên</Label>
        <input id="ho-ten" />
      </>,
    );
    const label = screen.getByText('Họ và tên');
    expect(label).toHaveAttribute('for', 'ho-ten');
    expect(label).toHaveClass('text-xs', 'font-medium', 'leading-none', 'text-fg-primary');
  });

  it('lets the call-site className win the merge', () => {
    render(<Label className="text-sm">Nhãn</Label>);
    const el = screen.getByText('Nhãn');
    expect(el).toHaveClass('text-sm');
    expect(el.className).not.toContain('text-xs');
  });

  it('passes the ref through to the label element', () => {
    const ref = createRef<HTMLLabelElement>();
    render(<Label ref={ref}>Nhãn</Label>);
    expect(ref.current).toBeInstanceOf(HTMLLabelElement);
  });
});
