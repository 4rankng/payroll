import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AdBanner } from '@/types/api/ad-banner.types';

import { EmployeeAdBanner } from './EmployeeAdBanner';

// This vitest jsdom environment exposes `localStorage` as a bare object with
// no Storage methods, so the component's getItem/setItem would silently throw
// (it catches) and stage persistence would never be observable. Install a real
// in-memory Storage for these tests.
Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  writable: true,
  value: (() => {
    const store = new Map<string, string>();
    return {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => { store.set(key, String(value)); },
      removeItem: (key: string) => { store.delete(key); },
      clear: () => store.clear(),
      key: (index: number) => Array.from(store.keys())[index] ?? null,
      get length() { return store.size; },
    };
  })(),
});

const mutate = vi.fn();

vi.mock('@/hooks/api/useEmployeeAdBanner', () => ({
  useEmployeeAdBanner: () => ({ data: bannerFixture(), isLoading: false }),
  useRecordAdBannerClick: () => ({ mutate }),
}));

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: () => true,
}));

function bannerFixture(version = '2026-09-06T00:00:00Z'): AdBanner {
  return {
    id: 7,
    title: 'Ứng lương sớm 24/7',
    body: 'Chấm công tự động và ứng lương ngay trên điện thoại.',
    bullets: ['Chấm công tự động', 'Ứng lương tối đa 70%'],
    ctas: [
      { label: 'Gọi hotline', type: 'phone', value: '0914827988' },
      { label: 'Nhóm Zalo', type: 'url', value: 'https://zalo.me/g/example' },
    ],
    footer: '',
    targetProjectIds: [],
    priority: 0,
    startsAt: '2026-09-06T00:00:00Z',
    endsAt: '2026-10-06T00:00:00Z',
    isActive: true,
    createdAt: '2026-09-06T00:00:00Z',
    updatedAt: version,
  };
}

describe('EmployeeAdBanner three-stage dismissal', () => {
  beforeEach(() => {
    localStorage.removeItem('employee_ad_state');
    mutate.mockClear();
  });

  it('auto-opens the sheet once for an unseen campaign version', () => {
    render(<EmployeeAdBanner />);

    // Sheet content is rendered in a portal — title must be visible once the
    // sheet is open, and no compact card yet.
    expect(screen.getAllByText('Ứng lương sớm 24/7').length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: 'Đóng' })).toBeInTheDocument();
  });

  it('dismiss sheet → compact card stays in the feed', () => {
    render(<EmployeeAdBanner />);

    fireEvent.click(screen.getByRole('button', { name: 'Đóng' }));

    expect(screen.getByLabelText('Quảng cáo')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Gọi hotline' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Xem chi tiết' })).toBeInTheDocument();

    const stored = JSON.parse(localStorage.getItem('employee_ad_state') || 'null');
    expect(stored).toMatchObject({ id: 7, stage: 'card' });
  });

  it('dismiss card → nothing rendered for that version', () => {
    render(<EmployeeAdBanner />);

    fireEvent.click(screen.getByRole('button', { name: 'Đóng' }));
    fireEvent.click(screen.getByLabelText('Đóng quảng cáo'));

    expect(screen.queryByText('Ứng lương sớm 24/7')).not.toBeInTheDocument();

    const stored = JSON.parse(localStorage.getItem('employee_ad_state') || 'null');
    expect(stored).toMatchObject({ id: 7, stage: 'hidden' });
  });

  it('stays hidden when the stored state matches the current version', () => {
    localStorage.setItem(
      'employee_ad_state',
      JSON.stringify({ id: 7, version: '2026-09-06T00:00:00Z', stage: 'hidden' }),
    );

    render(<EmployeeAdBanner />);

    expect(screen.queryByText('Ứng lương sớm 24/7')).not.toBeInTheDocument();
  });

  it('re-shows the sheet when the campaign is edited (version bump = republish)', async () => {
    localStorage.setItem(
      'employee_ad_state',
      JSON.stringify({ id: 7, version: '2026-09-05T00:00:00Z', stage: 'hidden' }),
    );

    render(<EmployeeAdBanner />);

    // New version invalidates the stored dismissal → sheet re-opens.
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Đóng' })).toBeInTheDocument();
    });
  });
});

describe('EmployeeAdBanner CTA behaviour', () => {
  beforeEach(() => {
    localStorage.removeItem('employee_ad_state');
    mutate.mockClear();
  });

  it('fires the click mutation before opening a url CTA, never gating navigation', () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    render(<EmployeeAdBanner />);
    fireEvent.click(screen.getByRole('button', { name: 'Đóng' })); // to card stage

    fireEvent.click(screen.getByRole('button', { name: 'Nhóm Zalo' }));

    expect(mutate).toHaveBeenCalledWith({ bannerId: 7, ctaIndex: 1 });
    expect(openSpy).toHaveBeenCalledWith('https://zalo.me/g/example', '_blank', 'noopener,noreferrer');
    openSpy.mockRestore();
  });

  it('records phone CTA taps with the banner id and CTA index', () => {
    render(<EmployeeAdBanner />);
    fireEvent.click(screen.getByRole('button', { name: 'Đóng' })); // to card stage

    fireEvent.click(screen.getByRole('button', { name: 'Gọi hotline' }));

    expect(mutate).toHaveBeenCalledWith({ bannerId: 7, ctaIndex: 0 });
  });
});
