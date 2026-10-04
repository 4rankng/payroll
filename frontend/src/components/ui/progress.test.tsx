import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Progress } from './progress';

describe('Progress contract', () => {
  it('forwards the ref to the Radix root div', () => {
    const ref = createRef<HTMLDivElement>();
    render(<Progress ref={ref} value={40} aria-label="Tiến độ" />);

    expect(ref.current).toBeInstanceOf(HTMLDivElement);
    expect(screen.getByRole('progressbar')).toBe(ref.current);
  });

  it('track utility-gray-100, indicator brand-solid', () => {
    const { container } = render(<Progress value={40} aria-label="Tiến độ" />);

    expect(container.querySelector('.bg-utility-gray-100')).toBeTruthy();
    expect(container.querySelector('.bg-brand-solid')).toBeTruthy();
  });

  it('lets callers override the height through the merge', () => {
    render(<Progress className="h-1.5" value={40} aria-label="Tiến độ" />);

    const bar = screen.getByRole('progressbar');
    expect(bar).toHaveClass('h-1.5');
    expect(bar).not.toHaveClass('h-4');
  });
});
