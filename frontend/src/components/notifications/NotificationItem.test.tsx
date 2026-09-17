import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NotificationItem } from './NotificationItem';
import type { Notification } from '@/types/api/notification.types';

const { markRead, markReadSilent } = vi.hoisted(() => ({ markRead: vi.fn(), markReadSilent: vi.fn() }));
vi.mock('@/hooks/api/useNotifications', () => ({
  useMarkAsRead: () => ({ mutate: markRead, isPending: false }),
  useMarkAsReadSilent: () => ({ mutate: markReadSilent, isPending: false }),
}));

const notification: Notification = {
  id: 17, type: 'custom', channel: 'push', title: 'Thông báo kiểm thử',
  message: 'Nội dung thông báo', read_at: null, created_at: '2026-09-17T08:00:00Z',
};

describe('NotificationItem', () => {
  beforeEach(() => vi.clearAllMocks());

  it('marks an item read without opening its details', () => {
    const onClick = vi.fn();
    render(<NotificationItem notification={notification} onClick={onClick} />);
    fireEvent.click(screen.getByRole('button', { name: 'Đánh dấu đã đọc' }));
    expect(markRead).toHaveBeenCalledExactlyOnceWith(17);
    expect(markReadSilent).not.toHaveBeenCalled();
    expect(onClick).not.toHaveBeenCalled();
  });

  it('opens details once with its separate native button', () => {
    const onClick = vi.fn();
    render(<NotificationItem notification={notification} onClick={onClick} />);
    fireEvent.click(screen.getByRole('button', { name: 'Thông báo kiểm thử. Chưa đọc.' }));
    expect(markReadSilent).toHaveBeenCalledExactlyOnceWith(17);
    expect(onClick).toHaveBeenCalledTimes(1);
    expect(markRead).not.toHaveBeenCalled();
  });
});
