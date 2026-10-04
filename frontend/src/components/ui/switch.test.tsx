import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';

import { Switch } from './switch';

describe('Switch contract', () => {
  it('forwards the ref to the Radix root button', () => {
    const ref = createRef<HTMLButtonElement>();
    render(<Switch ref={ref} aria-label="Bật/tắt" />);

    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
    expect(screen.getByRole('switch')).toBe(ref.current);
  });

  it('off state uses utility-gray-200, on state fills brand-solid', () => {
    render(<Switch aria-label="Tắt" />);
    const off = screen.getByRole('switch', { name: 'Tắt' });
    expect(off).toHaveClass(
      'data-[state=unchecked]:bg-utility-gray-200',
      'data-[state=checked]:bg-brand-solid',
    );
    expect(off).toHaveAttribute('data-state', 'unchecked');

    render(<Switch defaultChecked aria-label="Bật" />);
    expect(screen.getByRole('switch', { name: 'Bật' })).toHaveAttribute('data-state', 'checked');
  });

  it('thumb stays white with the UU shadow-xs elevation', () => {
    const { container } = render(<Switch aria-label="Bật/tắt" />);

    const thumb = container.querySelector('span');
    expect(thumb).toHaveClass('bg-white', 'shadow-xs');
  });

  it('keeps the 44px hit-area pseudo-element and UU outline focus', () => {
    render(<Switch aria-label="Bật/tắt" />);

    const root = screen.getByRole('switch');
    expect(root).toHaveClass('before:absolute', 'outline-brand', 'focus-visible:outline-2');
  });

  it('lets caller track colors override through the merge (CronJobTable pattern)', () => {
    render(<Switch className="data-[state=checked]:bg-emerald-500" aria-label="Bật/tắt" />);

    expect(screen.getByRole('switch')).toHaveClass('data-[state=checked]:bg-emerald-500');
  });
});
