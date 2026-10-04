import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Checkbox } from './checkbox';

describe('Checkbox contract', () => {
  it('forwards the ref to the Radix root button', () => {
    const ref = createRef<HTMLButtonElement>();
    render(<Checkbox ref={ref} aria-label="Chọn" />);

    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
    expect(screen.getByRole('checkbox')).toBe(ref.current);
  });

  it('unselected state uses the UU gray surface and border', () => {
    render(<Checkbox aria-label="Chọn" />);

    expect(screen.getByRole('checkbox')).toHaveClass(
      'border-utility-gray-300',
      'bg-card',
      'shadow-xs',
    );
  });

  it('checked state fills brand-solid with the check in white', () => {
    render(<Checkbox defaultChecked aria-label="Chọn" />);

    expect(screen.getByRole('checkbox')).toHaveClass(
      'data-[state=checked]:bg-brand-solid',
      'data-[state=checked]:border-brand-solid',
      'data-[state=checked]:text-white',
    );
  });

  it('keeps the UU outline-brand focus treatment', () => {
    render(<Checkbox aria-label="Chọn" />);

    expect(screen.getByRole('checkbox')).toHaveClass(
      'outline-brand',
      'focus-visible:outline',
      'focus-visible:outline-2',
      'focus-visible:outline-offset-2',
    );
  });

  it('lets caller border tints override the base border through the merge', () => {
    render(<Checkbox className="border-amber-600" aria-label="Chọn" />);

    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).toHaveClass('border-amber-600');
    expect(checkbox).not.toHaveClass('border-utility-gray-300');
  });
});
