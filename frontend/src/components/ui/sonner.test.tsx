import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Toaster, toast } from './sonner';

// The restyle pins the Toaster's prop contract rather than sonner's DOM
// (no async toast-appearance timing in jsdom): the sonner runtime is stubbed
// to capture props, and the toast() wrapper is pinned against a spy. The
// runtime itself is engine-kept (unchanged import, plan-of-record ruling).
const { toasterSpy, baseToastSpy } = vi.hoisted(() => ({
  // Typed parameter so mock.calls elements are indexable tuples (TS2493).
  toasterSpy: vi.fn((props?: Record<string, unknown>) => null),
  baseToastSpy: vi.fn(),
}));

vi.mock('sonner', () => ({
  Toaster: toasterSpy,
  toast: baseToastSpy,
}));

vi.mock('next-themes', () => ({
  useTheme: () => ({ theme: 'light' }),
}));

interface CapturedToasterProps {
  theme: string;
  duration: number;
  className?: string;
  toastOptions: {
    duration: number;
    classNames: Record<string, string>;
  };
}

describe('sonner toaster (UU restyle contract)', () => {
  it('mounts with the UU toast surface on every hook point', () => {
    render(<Toaster />);
    const props = toasterSpy.mock.calls.at(-1)![0] as unknown as CapturedToasterProps;

    // Theme plumbing and durations intact.
    expect(props.theme).toBe('light');
    expect(props.duration).toBe(3000);
    expect(props.toastOptions.duration).toBe(3000);
    expect(props.className).toBe('toaster group');

    const classNames = props.toastOptions.classNames;
    // Existing hook points stay (values restyled; title/closeButton added).
    expect(Object.keys(classNames)).toEqual(
      expect.arrayContaining([
        'toast',
        'title',
        'description',
        'actionButton',
        'cancelButton',
        'closeButton',
      ]),
    );
    // Hook classes preserved; surface on the landed select-content recipe.
    expect(classNames.toast).toContain('group toast');
    expect(classNames.toast).toContain('group-[.toaster]:bg-card');
    expect(classNames.toast).toContain('group-[.toaster]:text-fg-primary');
    expect(classNames.toast).toContain('group-[.toaster]:ring-utility-gray-200');
    expect(classNames.toast).toContain('group-[.toaster]:shadow-lg');
    expect(classNames.toast).toContain('group-[.toaster]:rounded-lg');
    expect(classNames.title).toContain('group-[.toast]:text-fg-primary');
    expect(classNames.description).toContain('group-[.toast]:text-fg-tertiary');
    expect(classNames.actionButton).toContain('group-[.toast]:bg-brand-solid');
    expect(classNames.cancelButton).toContain('group-[.toast]:bg-utility-gray-100');
    expect(classNames.closeButton).toContain('group-[.toast]:text-fg-quaternary');
  });

  it('colors toast icons from the bridge families by data-type', () => {
    render(<Toaster />);
    const props = toasterSpy.mock.calls.at(-1)![0] as unknown as CapturedToasterProps;
    const toastClasses = props.toastOptions.classNames.toast;

    expect(toastClasses).toContain('[&[data-type=success]_[data-icon]]:text-success-solid');
    expect(toastClasses).toContain('[&[data-type=error]_[data-icon]]:text-error-solid');
    expect(toastClasses).toContain('[&[data-type=warning]_[data-icon]]:text-warning-solid');
    // Bridge gap: no info ladder — W4's Tailwind sky interim applies.
    expect(toastClasses).toContain('[&[data-type=info]_[data-icon]]:text-sky-600');
  });

  it('keeps the toast() wrapper contract', () => {
    toast({
      title: 'Đã lưu',
      description: 'Bảng công',
      variant: 'success',
      duration: 8000,
    });
    expect(baseToastSpy).toHaveBeenCalledWith(
      'Đã lưu',
      expect.objectContaining({
        description: 'Bảng công',
        duration: 8000,
        className: 'toast-variant-success',
      }),
    );

    toast('Đồng bộ xong', { duration: 9999 });
    expect(baseToastSpy).toHaveBeenLastCalledWith('Đồng bộ xong', { duration: 9999 });
  });
});
