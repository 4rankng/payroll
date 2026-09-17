import { renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useDialogFocusReturn } from './useDialogFocusReturn';

describe('dialog focus return', () => {
  it('restores a controlled dialog opener without scrolling the page', () => {
    const button = document.createElement('button');
    document.body.append(button);
    button.focus();
    const { result } = renderHook(() => useDialogFocusReturn());
    result.current.onOpenAutoFocus(new Event('open', { cancelable: true }));
    button.blur();
    const event = new Event('close', { cancelable: true });
    result.current.onCloseAutoFocus(event);
    expect(event.defaultPrevented).toBe(true);
    expect(button).toHaveFocus();
    button.remove();
  });

  it('respects caller focus handling and removed openers', () => {
    const button = document.createElement('button');
    document.body.append(button);
    button.focus();
    const focus = vi.spyOn(button, 'focus');
    const onClose = vi.fn((event: Event) => event.preventDefault());
    const { result, rerender } = renderHook(({ callback }) => useDialogFocusReturn(undefined, callback), { initialProps: { callback: onClose } });
    result.current.onOpenAutoFocus(new Event('open'));
    result.current.onCloseAutoFocus(new Event('close', { cancelable: true }));
    expect(onClose).toHaveBeenCalledOnce();
    expect(focus).not.toHaveBeenCalled();
    rerender({ callback: vi.fn() });
    result.current.onOpenAutoFocus(new Event('open'));
    button.remove();
    const event = new Event('close', { cancelable: true });
    result.current.onCloseAutoFocus(event);
    expect(event.defaultPrevented).toBe(false);
    expect(focus).not.toHaveBeenCalled();
  });
});
