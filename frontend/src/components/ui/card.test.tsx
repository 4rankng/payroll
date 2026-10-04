import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './card';

describe('Card contract', () => {
  it('renders every variant with its UU surface signature', () => {
    const signatures = [
      ['default', 'border-utility-gray-200'],
      ['elevated', 'hover:shadow-sm'],
      ['outlined', 'border-utility-gray-300'],
      ['filled', 'bg-utility-gray-50'],
    ] as const;
    for (const [variant, signature] of signatures) {
      const { unmount } = render(<Card variant={variant} data-testid={`card-${variant}`} />);
      expect(screen.getByTestId(`card-${variant}`)).toHaveClass(signature);
      unmount();
    }
  });

  it('keeps the compound layout parts rendering', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Phiếu lương</CardTitle>
          <CardDescription>Tháng 10</CardDescription>
        </CardHeader>
        <CardContent>Nội dung</CardContent>
        <CardFooter>Chân</CardFooter>
      </Card>,
    );
    expect(screen.getByText('Phiếu lương')).toHaveClass('text-lg', 'font-semibold', 'text-fg-primary');
    expect(screen.getByText('Tháng 10')).toHaveClass('text-sm', 'text-fg-tertiary');
  });

  it('lets the call-site className win the merge', () => {
    render(<Card variant="default" className="shadow-lg" data-testid="card-merge" />);
    const el = screen.getByTestId('card-merge');
    expect(el).toHaveClass('shadow-lg');
    expect(el.className).not.toContain('shadow-xs');
  });

  it('passes the ref through to the div', () => {
    const ref = createRef<HTMLDivElement>();
    render(<Card ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLDivElement);
  });
});
