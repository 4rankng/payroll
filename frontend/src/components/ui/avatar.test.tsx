import { render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it } from 'vitest';
import { Avatar, AvatarFallback, AvatarImage } from './avatar';

describe('Avatar compound contract', () => {
  it('renders image and fallback through the UU restyle', () => {
    render(
      <Avatar>
        <AvatarImage src="/anh-dai-dien.png" alt="Nhân viên" />
        <AvatarFallback>NV</AvatarFallback>
      </Avatar>,
    );
    const root = screen.getByText('NV').parentElement as HTMLElement;
    expect(root).toHaveClass('bg-utility-gray-100', 'outline-utility-gray-200');
    expect(screen.getByText('NV')).toHaveClass('text-fg-quaternary');
  });

  it('lets the call-site className win the merge', () => {
    render(<Avatar className="h-14 w-14" data-testid="avatar-sized" />);
    const el = screen.getByTestId('avatar-sized');
    expect(el).toHaveClass('h-14', 'w-14');
    expect(el.className).not.toContain('h-10');
  });

  it('passes the ref through to the root span', () => {
    const ref = createRef<HTMLSpanElement>();
    render(<Avatar ref={ref} />);
    expect(ref.current).toBeInstanceOf(HTMLSpanElement);
  });
});
