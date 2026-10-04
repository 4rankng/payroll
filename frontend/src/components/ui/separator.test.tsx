import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';
import { Separator } from './separator';

describe('Separator contract', () => {
  it('renders horizontal by default with the UU divider gray', () => {
    render(<Separator data-testid="sep-h" />);
    const el = screen.getByTestId('sep-h');
    expect(el).toHaveClass('bg-utility-gray-200', 'h-[1px]', 'w-full');
    expect(el).toHaveAttribute('data-orientation', 'horizontal');
  });

  it('flips dimensions for vertical orientation', () => {
    render(<Separator orientation="vertical" data-testid="sep-v" />);
    expect(screen.getByTestId('sep-v')).toHaveClass('h-full', 'w-[1px]');
  });

  it('lets the call-site className win the merge', () => {
    render(<Separator className="bg-red-500" data-testid="sep-merge" />);
    const el = screen.getByTestId('sep-merge');
    expect(el).toHaveClass('bg-red-500');
    expect(el.className).not.toContain('bg-utility-gray-200');
  });

  it('passes the ref through', () => {
    const ref = createRef<HTMLDivElement>();
    render(<Separator ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLDivElement);
  });
});
