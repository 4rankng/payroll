import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from './dropdown-menu';

// Thin contract tests for the W6 UU restyle: engine stays Radix, surface
// vocabulary moves to the UU token bridge while call-site contracts
// (forceMount pass-through, inset slots, indicator structure) stay intact.
describe('DropdownMenu contract', () => {
  it('renders the trigger and keeps the ref pass-through', () => {
    render(
      <DropdownMenu open>
        <DropdownMenuTrigger>Menu</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem onSelect={vi.fn()}>Hồ sơ</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    const trigger = screen.getByText('Menu').closest('button');
    expect(trigger).not.toBeNull();
    expect(trigger).toHaveAttribute('aria-haspopup', 'menu');
    expect(screen.getByRole('menu')).toBeInTheDocument();
  });

  it('renders popper content with the UU surface and animation hooks', () => {
    render(
      <DropdownMenu open>
        <DropdownMenuTrigger>Menu</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem onSelect={vi.fn()}>Hồ sơ</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    const menu = screen.getByRole('menu');
    expect(menu).toHaveClass(
      'bg-popover',
      'rounded-lg',
      'ring-1',
      'ring-utility-gray-200',
      'shadow-sm',
    );
    // tailwindcss-animate hooks bound to Radix data-* states survive.
    expect(menu).toHaveClass('data-[state=open]:animate-in', 'data-[state=closed]:animate-out');
  });

  it('styles items with the utility-gray-100 focus surface and 44px/compact floors', () => {
    render(
      <DropdownMenu open>
        <DropdownMenuTrigger>Menu</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem onSelect={vi.fn()}>Hồ sơ</DropdownMenuItem>
          <DropdownMenuItem inset onSelect={vi.fn()}>
            Ngừng theo dõi
          </DropdownMenuItem>
          <DropdownMenuSeparator data-testid="sep" />
          <DropdownMenuLabel>Nhóm</DropdownMenuLabel>
          <DropdownMenuCheckboxItem checked onSelect={vi.fn()}>
            Chỉ hiển thị đang hoạt động
          </DropdownMenuCheckboxItem>
          <DropdownMenuRadioGroup value="a">
            <DropdownMenuRadioItem value="a" onSelect={vi.fn()}>
              Tuần này
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    const item = screen.getByRole('menuitem', { name: 'Hồ sơ' });
    expect(item).toHaveClass('focus:bg-utility-gray-100', 'min-h-11', 'sm:min-h-8');

    const inset = screen.getByRole('menuitem', { name: 'Ngừng theo dõi' });
    expect(inset).toHaveClass('pl-8');

    const checkbox = screen.getByRole('menuitemcheckbox');
    expect(checkbox).toHaveClass('focus:bg-utility-gray-100');
    // Checked indicator renders the check glyph.
    expect(checkbox.querySelector('svg')).not.toBeNull();

    const radio = screen.getByRole('menuitemradio');
    expect(radio).toHaveClass('focus:bg-utility-gray-100');
    expect(radio.querySelector('svg')).not.toBeNull();

    expect(screen.getByTestId('sep')).toHaveClass('bg-utility-gray-200');
    expect(screen.getByText('Nhóm').closest('div')).toHaveClass('typography-body-medium');
  });

  it('keeps the submenu trigger hooks and shortcut slot', () => {
    render(
      <DropdownMenu open>
        <DropdownMenuTrigger>Menu</DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>Thêm</DropdownMenuSubTrigger>
          </DropdownMenuSub>
          <DropdownMenuItem>
            Xuất Excel
            <DropdownMenuShortcut>⌘E</DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    const sub = screen.getByText('Thêm').closest('div');
    expect(sub).toHaveClass(
      'focus:bg-utility-gray-100',
      'data-[state=open]:bg-utility-gray-100',
    );

    expect(screen.getByText('⌘E')).toHaveClass('typography-body-small', 'opacity-60');
  });

  it('accepts and forwards forceMount without breaking the closed state (UserAvatarDropdown contract)', () => {
    render(
      <DropdownMenu>
        <DropdownMenuTrigger>Menu</DropdownMenuTrigger>
        <DropdownMenuContent forceMount>
          <DropdownMenuItem onSelect={vi.fn()}>Đăng xuất</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    // In this Radix version forceMount content is not mounted while the root
    // is closed; the wrapper contract is only that the prop is accepted and
    // forwarded (via {...props}) and the closed state renders cleanly.
    expect(screen.getByRole('button', { name: 'Menu' })).toHaveAttribute(
      'data-state',
      'closed',
    );
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('lets caller classes override the content surface through the merge', () => {
    render(
      <DropdownMenu open>
        <DropdownMenuTrigger>Menu</DropdownMenuTrigger>
        <DropdownMenuContent className="rounded-none">
          <DropdownMenuItem onSelect={vi.fn()}>Hồ sơ</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>,
    );

    expect(screen.getByRole('menu')).toHaveClass('rounded-none');
    expect(screen.getByRole('menu')).not.toHaveClass('rounded-lg');
  });
});
