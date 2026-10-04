import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { Popover, PopoverContent, PopoverTrigger } from './popover';

describe('popover UU surface (W5)', () => {
  it('renders the UU popover surface inside the portal and merges caller classes', () => {
    render(
      <Popover open>
        <PopoverTrigger>Lọc</PopoverTrigger>
        <PopoverContent className="w-80">Nội dung bộ lọc</PopoverContent>
      </Popover>,
    );
    expect(screen.getByText('Nội dung bộ lọc')).toHaveClass(
      'bg-popover',
      'border-utility-gray-200',
      'rounded-xl',
      'shadow-lg',
      'outline-none',
      'w-80',
    );
  });
});
