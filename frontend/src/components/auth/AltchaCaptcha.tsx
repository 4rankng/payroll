import { useEffect, useRef, useState } from "react";
import "altcha";
import "altcha/i18n/vi";
import { AuthSpinner } from "@/components/auth/AuthSpinner";

interface AltchaCaptchaProps {
  /** Stringified ALTCHA challenge JSON from GET /auth/captcha. */
  challenge: string;
  /** Called with the base64 proof-of-work payload once the widget verifies. */
  onVerified: (payload: string) => void;
}

/**
 * Self-hosted ALTCHA proof-of-work checkbox (PAYR-16). With auto="onload" the
 * browser solves the challenge in a background worker as soon as the widget
 * mounts — the user only sees a checkbox that ticks itself. `language="vi"` +
 * the i18n import above render the widget labels in Vietnamese.
 *
 * The widget dispatches `verified` as a NON-BUBBLING CustomEvent from an inner
 * node, so a host-element listener can miss it; a document-level capture
 * listener reliably observes it regardless of target/bubbles.
 */
export const AltchaCaptcha = ({ challenge, onVerified }: AltchaCaptchaProps) => {
  // Delivered payloads are tracked per challenge: a fresh challenge (reset
  // after a consumed solution) must be allowed to verify again, while a single
  // challenge's event is delivered exactly once.
  const [verifiedChallenge, setVerifiedChallenge] = useState<string | null>(null);
  const onVerifiedRef = useRef(onVerified);

  // Keep the latest callback without re-binding the DOM listener.
  useEffect(() => {
    onVerifiedRef.current = onVerified;
  }, [onVerified]);

  useEffect(() => {
    const handleVerified = (event: Event) => {
      const payload = (event as CustomEvent<{ payload?: string }>).detail?.payload;
      if (payload && verifiedChallenge !== challenge) {
        setVerifiedChallenge(challenge);
        onVerifiedRef.current(payload);
      }
    };
    document.addEventListener("verified", handleVerified, true);
    return () => document.removeEventListener("verified", handleVerified, true);
  }, [challenge, verifiedChallenge]);

  if (!challenge) {
    return (
      <div
        className="flex h-11 items-center gap-2 rounded-lg border border-input bg-muted px-3 text-sm text-fg-tertiary sm:h-9"
        role="status"
        aria-live="polite"
      >
        <AuthSpinner className="size-4 text-fg-tertiary" />
        Đang tải mã xác nhận...
      </div>
    );
  }

  return <altcha-widget challenge={challenge} language="vi" auto="onload" className="block w-full" />;
};
