import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NotificationDetailModal } from './NotificationDetailModal';
import type { Notification } from '@/types/api/notification.types';

vi.mock('@/hooks/api/useNotifications', () => ({
  useMarkAsReadSilent: () => ({ mutate: vi.fn(), isPending: false }),
}));

// The global jsdom matchMedia mock (src/test/setup.ts) only matches
// `prefers-reduced-motion`, so `useIsMobile` (backed by `useBreakpoint('md')`,
// i.e. `(max-width: 767px)` since the tablet overhaul) always reads false
// there — the desktop branch only. Override it here so this test exercises
// the phone bottom-sheet branch, which is the one the regression broke.
function mockMobileViewport() {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: (query: string): MediaQueryList => ({
      matches: query.includes('max-width: 767px'),
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  });
}

const notification = {
  id: 1,
  type: 'system_alert',
  channel: 'push',
  title: 'High Error Rate Alert',
  message: 'GET /api/v1/projects/*/payrate: 31.08% (46/148 errors)',
  read_at: null,
  created_at: '2026-09-04T10:00:00+07:00',
} satisfies Notification;

describe('NotificationDetailModal', () => {
  beforeEach(() => {
    mockMobileViewport();
  });

  it('renders as a full-width mobile bottom sheet, not a shrunk/clipped fixed-width dialog', () => {
    render(
      <NotificationDetailModal notification={notification} isOpen onClose={vi.fn()} />
    );

    const dialog = screen.getByRole('dialog');
    // Regression: the component's own `w-[90vw]` overrode the shared
    // DialogContent's mobile `w-full`, since tailwind-merge keeps only the
    // last-listed width utility for a given property. That narrowed and
    // left-shifted the bottom sheet, clipping content on the right and
    // leaving a large empty gap above a short card.
    expect(dialog.className).toContain('w-full');
    expect(dialog.className).not.toMatch(/w-\[\d/);
    // Sanity check we actually hit the mobile bottom-sheet branch.
    expect(dialog.className).toContain('inset-x-0');
    expect(dialog.className).toContain('bottom-0');
  });
});
