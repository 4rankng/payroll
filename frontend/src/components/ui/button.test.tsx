import { fireEvent, render, screen } from '@testing-library/react';
import { createRef } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { Button, buttonVariants } from './button';

const VARIANTS = [
  'default',
  'destructive',
  'secondary',
  'outline',
  'ghost',
  'success',
  'info',
  'warning',
  'monochrome',
  'monochrome-outline',
  'link',
] as const;

const SIGNATURE: Record<(typeof VARIANTS)[number], string> = {
  default: 'bg-brand-solid',
  destructive: 'bg-error-solid',
  secondary: 'bg-utility-gray-100',
  outline: 'border-border',
  ghost: 'text-tertiary',
  success: 'bg-success-solid',
  info: 'bg-sky-600',
  warning: 'bg-warning-solid',
  monochrome: 'bg-utility-gray-700',
  'monochrome-outline': 'text-utility-gray-700',
  link: 'text-brand-secondary',
};

describe('Button contract', () => {
  it('renders every variant and size with its UU-token signature class', () => {
    for (const variant of VARIANTS) {
      const { unmount } = render(<Button variant={variant}>Nhan</Button>);
      const el = screen.getByRole('button', { name: 'Nhan' });
      expect(el).toHaveClass(SIGNATURE[variant]);
      expect(el).toHaveClass('h-11', 'text-sm', 'rounded-lg');
      unmount();
    }
    for (const size of ['sm', 'lg'] as const) {
      const { unmount } = render(<Button size={size}>Nhan</Button>);
      expect(screen.getByRole('button', { name: 'Nhan' })).toHaveClass('h-11');
      unmount();
    }
    render(<Button size="xl">Nhan</Button>);
    expect(screen.getByRole('button', { name: 'Nhan' })).toHaveClass('h-12', 'sm:h-11');
    render(<Button size="icon" aria-label="Icon">+</Button>);
    expect(screen.getByRole('button', { name: 'Icon' })).toHaveClass('h-11', 'w-11');
  });

  it('keeps a native button element with the DOM attribute contract', () => {
    render(
      <Button type="submit" form="ky-gui" name="luu" value="1">
        Luu
      </Button>,
    );
    const el = screen.getByRole('button', { name: 'Luu' });
    expect(el).toBeInstanceOf(HTMLButtonElement);
    expect(el).toHaveAttribute('type', 'submit');
    expect(el).toHaveAttribute('form', 'ky-gui');
    expect(el).toHaveAttribute('name', 'luu');
    expect(el).toHaveAttribute('value', '1');
  });

  it('leaves type unset by default so in-form buttons stay implicit submits', () => {
    render(<Button>Gui</Button>);
    expect(screen.getByRole('button', { name: 'Gui' })).not.toHaveAttribute('type');
  });

  it('forwards a real HTMLButtonElement ref', () => {
    const ref = createRef<HTMLButtonElement>();
    render(<Button ref={ref} variant="outline">Chon</Button>);
    expect(ref.current).toBeInstanceOf(HTMLButtonElement);
    ref.current?.focus();
    expect(ref.current).toHaveFocus();
  });

  it('disables through the native disabled attribute with UU disabled tokens', () => {
    const onClick = vi.fn();
    render(<Button disabled onClick={onClick}>Xoa</Button>);
    const el = screen.getByRole('button', { name: 'Xoa' });
    expect(el).toBeDisabled();
    expect(el).toHaveClass('disabled:bg-disabled', 'disabled:text-fg-disabled');
    fireEvent.click(el);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('renders asChild through Slot with call-site className winning the merge', () => {
    render(
      <Button asChild variant="ghost" className="custom-override">
        <a href="/cong-no">Cong no</a>
      </Button>,
    );
    const anchor = screen.getByRole('link', { name: 'Cong no' });
    expect(anchor).toHaveAttribute('href', '/cong-no');
    expect(anchor).toHaveClass('text-tertiary');
    expect(anchor).toHaveClass('custom-override');
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('lets call-site className override variant classes through tailwind-merge', () => {
    render(<Button size="icon" className="h-7 w-7">Thu gon</Button>);
    const el = screen.getByRole('button', { name: 'Thu gon' });
    expect(el).toHaveClass('h-7', 'w-7');
    expect(el).not.toHaveClass('h-11');
  });

  it('keeps buttonVariants callable for pagination and calendar composition', () => {
    const ghost = buttonVariants({ variant: 'ghost', size: 'sm' });
    expect(ghost).toContain('text-tertiary');
    expect(ghost).toContain('h-11');

    const fallback = buttonVariants({});
    expect(fallback).toContain('bg-brand-solid');
    expect(fallback).toContain('h-11');

    const outlined = buttonVariants({ variant: 'outline', className: 'w-full' });
    expect(outlined).toContain('border-border');
    expect(outlined).toContain('w-full');
  });
});
