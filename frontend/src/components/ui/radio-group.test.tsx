import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { RadioGroup, RadioGroupItem } from './radio-group';

describe('RadioGroup contract', () => {
  it('forwards the ref to the Radix item button', () => {
    const ref = createRef<HTMLButtonElement>();
    render(
      <RadioGroup>
        <RadioGroupItem value="a" ref={ref} aria-label="Lựa chọn A" />
      </RadioGroup>,
    );

    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
    expect(screen.getByRole('radio')).toBe(ref.current);
  });

  it('unselected state uses the UU gray surface and border', () => {
    render(
      <RadioGroup>
        <RadioGroupItem value="a" aria-label="Lựa chọn A" />
      </RadioGroup>,
    );

    expect(screen.getByRole('radio')).toHaveClass(
      'border-utility-gray-300',
      'bg-card',
      'shadow-xs',
    );
  });

  it('selected state fills brand-solid with the dot in white', () => {
    render(
      <RadioGroup defaultValue="a">
        <RadioGroupItem value="a" aria-label="Lựa chọn A" />
      </RadioGroup>,
    );

    expect(screen.getByRole('radio')).toHaveClass(
      'data-[state=checked]:bg-brand-solid',
      'data-[state=checked]:border-brand-solid',
      'data-[state=checked]:text-white',
    );
  });

  it('lets caller border tints override the base border through the merge', () => {
    render(
      <RadioGroup>
        <RadioGroupItem value="a" className="border-amber-600" aria-label="Lựa chọn A" />
      </RadioGroup>,
    );

    const item = screen.getByRole('radio');
    expect(item).toHaveClass('border-amber-600');
    expect(item).not.toHaveClass('border-utility-gray-300');
  });
});
