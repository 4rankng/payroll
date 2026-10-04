import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Skeleton } from './skeleton';

describe('Skeleton contract', () => {
  it('renders with the UU skeleton gray and pulse animation', () => {
    render(<Skeleton data-testid="sk" />);
    expect(screen.getByTestId('sk')).toHaveClass('bg-utility-gray-100', 'animate-pulse', 'rounded-md');
  });

  it('lets the call-site className win the merge', () => {
    render(<Skeleton className="rounded-full" data-testid="sk-merge" />);
    const el = screen.getByTestId('sk-merge');
    expect(el).toHaveClass('rounded-full');
    expect(el.className).not.toContain('rounded-md');
  });
});
