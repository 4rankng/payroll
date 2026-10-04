import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Slider } from './slider';

describe('Slider contract', () => {
  it('forwards the ref to the Radix root and exposes the slider role', () => {
    const ref = createRef<HTMLSpanElement>();
    render(<Slider ref={ref} defaultValue={[30]} aria-label="Mức" />);

    expect(ref.current).toBeInstanceOf(HTMLSpanElement);
    expect(screen.getByRole('slider')).toBeTruthy();
  });

  it('track is utility-gray-200, range and thumb are brand-solid', () => {
    const { container } = render(<Slider defaultValue={[30]} aria-label="Mức" />);

    expect(container.querySelector('.bg-utility-gray-200')).toBeTruthy();
    expect(container.querySelectorAll('.bg-brand-solid')).toHaveLength(2);
    expect(container.querySelector('.border-brand-solid')).toBeTruthy();
  });

  it('thumb keeps the UU outline-brand focus treatment', () => {
    render(<Slider defaultValue={[30]} aria-label="Mức" />);

    const thumb = screen.getByRole('slider');
    expect(thumb).toHaveClass('outline-brand', 'focus-visible:outline-2');
  });
});
