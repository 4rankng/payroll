import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Dialog, DialogContent, DialogNavyHeader, DialogTrigger } from './dialog';
import { useIsMobile } from '@/hooks/useBreakpoint';

vi.mock('@/hooks/useBreakpoint', () => ({ useIsMobile: vi.fn(() => false) }));

beforeEach(() => vi.mocked(useIsMobile).mockReturnValue(false));

describe('shared dialog header', () => {
  it('names the dialog and describes it using its visible header', () => {
    render(
      <Dialog>
        <DialogTrigger>Mở kết quả</DialogTrigger>
        <DialogContent contentPadding="none" hideCloseButton>
          <DialogNavyHeader title="Kết quả chuyển tiền" description="Kiểm tra kết quả trước khi tiếp tục." />
          <p>Nội dung kết quả</p>
        </DialogContent>
      </Dialog>,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Mở kết quả' }));
    const dialog = screen.getByRole('dialog', { name: 'Kết quả chuyển tiền' });
    expect(dialog).toHaveAccessibleDescription('Kiểm tra kết quả trước khi tiếp tục.');
    fireEvent.click(screen.getByRole('button', { name: 'Đóng' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

describe('responsive dialog width', () => {
  it('keeps mobile bottom sheets edge to edge despite caller desktop constraints', () => {
    vi.mocked(useIsMobile).mockReturnValue(true);
    render(<Dialog open><DialogContent title="Chuyển tiền" description="Nhập tài khoản"
      className="w-[calc(100vw-1rem)] sm:max-w-[580px]"
      style={{ width: 374, maxWidth: 580, marginLeft: 8, marginRight: 8, backgroundColor: 'white' }}
    /></Dialog>);
    expect(screen.getByRole('dialog')).toHaveStyle({ width: '100%', maxWidth: 'none', left: '0px', right: '0px', marginLeft: '0px', marginRight: '0px' });
    expect(screen.getByRole('dialog').style.backgroundColor).toBe('white');
  });

  it('retains desktop dimensions supplied by the caller', () => {
    render(<Dialog open><DialogContent title="Chuyển tiền" description="Nhập tài khoản" style={{ width: 580, maxWidth: 580 }} /></Dialog>);
    expect(screen.getByRole('dialog')).toHaveStyle({ width: '580px', maxWidth: '580px' });
    expect(screen.getByRole('dialog').style.left).toBe('');
  });
});
