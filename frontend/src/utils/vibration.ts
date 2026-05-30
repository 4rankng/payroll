/**
 * Vibration utility for haptic feedback
 * Works on Android, silently ignored on iOS
 */

/**
 * Trigger a short vibration (50ms) for notification feedback
 */
export function vibrateOnNotification(): void {
  if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
    navigator.vibrate(50);
  }
}

/**
 * Trigger a success vibration pattern
 */
export function vibrateSuccess(): void {
  if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
    navigator.vibrate([50, 30, 50]);
  }
}

/**
 * Trigger an error vibration pattern
 */
export function vibrateError(): void {
  if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
    navigator.vibrate([100, 50, 100, 50, 100]);
  }
}
