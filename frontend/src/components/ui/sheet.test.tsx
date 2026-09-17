import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Sheet, SheetContent, SheetDescription, SheetTitle } from './sheet';
import { useIsMobile } from '@/hooks/useBreakpoint';

vi.mock('@/hooks/useBreakpoint', () => ({ useIsMobile: vi.fn(() => false) }));

beforeEach(() => vi.mocked(useIsMobile).mockReturnValue(false));

describe('sheet accessible names', () => {
  it('uses the visible title instead of a competing generic label', () => {
    render(<Sheet open><SheetContent><SheetTitle>Bộ lọc dự án</SheetTitle><SheetDescription>Chọn trạng thái dự án.</SheetDescription></SheetContent></Sheet>);
    expect(screen.getByRole('dialog', { name: 'Bộ lọc dự án' })).toHaveAccessibleDescription('Chọn trạng thái dự án.');
    expect(screen.queryByText('Sheet')).not.toBeInTheDocument();
  });
  it('supports explicit names for sheets with custom visual headers', () => {
    render(<Sheet open><SheetContent title="Thông báo" description="Thông báo tài khoản"><p>Nội dung</p></SheetContent></Sheet>);
    expect(screen.getByRole('dialog', { name: 'Thông báo' })).toHaveAccessibleDescription('Thông báo tài khoản');
  });
});

describe('responsive sheet width', () => {
  it.each(['bottom', 'top'] as const)('keeps mobile %s sheets flush to both viewport edges', side => {
    vi.mocked(useIsMobile).mockReturnValue(true);
    render(<Sheet open><SheetContent side={side} title="Chi tiết" description="Thông tin"
      className="sm:w-[480px] md:max-w-[520px]"
      style={{ width: 480, maxWidth: 520, marginLeft: 8, marginRight: 8 }}
    /></Sheet>);
    expect(screen.getByRole('dialog')).toHaveStyle({ width: '100%', maxWidth: 'none', left: '0px', right: '0px', marginLeft: '0px', marginRight: '0px' });
  });

  it.each([
    { mobile: true, side: 'right' as const },
    { mobile: true, side: 'left' as const },
    { mobile: false, side: 'bottom' as const },
  ])('preserves caller dimensions for $side sheets when mobile=$mobile', ({ mobile, side }) => {
    vi.mocked(useIsMobile).mockReturnValue(mobile);
    render(<Sheet open><SheetContent side={side} title="Chi tiết" description="Thông tin" style={{ width: 480, maxWidth: 520 }} /></Sheet>);
    expect(screen.getByRole('dialog')).toHaveStyle({ width: '480px', maxWidth: '520px' });
    expect(screen.getByRole('dialog').style.left).toBe('');
  });
});
