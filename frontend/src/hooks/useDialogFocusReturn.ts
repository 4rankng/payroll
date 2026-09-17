import { useRef } from 'react';

/** Preserve keyboard position for controlled dialogs opened without a Radix Trigger. */
export function useDialogFocusReturn(
  onOpenAutoFocus?: (event: Event) => void,
  onCloseAutoFocus?: (event: Event) => void,
) {
  const opener = useRef<HTMLElement | null>(null);

  return {
    onOpenAutoFocus: (event: Event) => {
      opener.current = document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
      onOpenAutoFocus?.(event);
    },
    onCloseAutoFocus: (event: Event) => {
      onCloseAutoFocus?.(event);
      const target = opener.current;
      if (!event.defaultPrevented && target?.isConnected && target !== document.body) {
        event.preventDefault();
        target.focus({ preventScroll: true });
      }
      opener.current = null;
    },
  };
}
