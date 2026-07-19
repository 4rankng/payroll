import { useLayoutEffect, useRef } from "react";
import { animate } from "animejs";
import { cn } from "@/lib/utils";

/**
 * Evaluates once at load — avoids per-frame media queries. Mirrors the pattern
 * used by `useMobilePageAnimations` so reduced-motion users see a static value.
 */
const prefersReducedMotion =
  typeof window !== "undefined"
    ? window.matchMedia("(prefers-reduced-motion: reduce)").matches
    : false;

/** Single shared VND formatter — matches `formatCurrency`'s default output. */
const VND_FORMAT = new Intl.NumberFormat("vi-VN", {
  style: "currency",
  currency: "VND",
});

const formatVND = (value: number) => VND_FORMAT.format(value);

interface AnimatedCurrencyProps {
  /** Target amount in VND (whole dong). Animates from the previously shown value. */
  amount: number;
  /** Tween duration in ms. */
  duration?: number;
  /** First-mount start value. Defaults to 0 for a count-up entrance. */
  initialFrom?: number;
  className?: string;
}

/**
 * Animated VND amount — counts up/down to `amount` using the project's anime.js
 * v4 primitive (no new dependency). Flash-free: `useLayoutEffect` snaps the
 * start value before paint. Respects `prefers-reduced-motion` (renders static).
 *
 * Inspired by 21st.dev's "Animate Digits" concept, rebuilt natively to match
 * the employee portal's animation conventions and `--employee-*` token system.
 */
export function AnimatedCurrency({
  amount,
  duration = 650,
  initialFrom = 0,
  className,
}: AnimatedCurrencyProps) {
  const spanRef = useRef<HTMLSpanElement>(null);
  const fromRef = useRef<number>(initialFrom);

  useLayoutEffect(() => {
    const el = spanRef.current;
    if (!el) return;

    const from = fromRef.current;
    const to = amount;

    // Reduced motion or no-op change: snap to the target, no tween.
    if (prefersReducedMotion || from === to) {
      el.textContent = formatVND(to);
      fromRef.current = to;
      return;
    }

    // Snap the start value before paint so the tween reads as a clean count-up
    // rather than a flash of the final value.
    el.textContent = formatVND(from);

    const target = { value: from };
    animate(target, {
      value: to,
      round: 1,
      duration,
      ease: "outExpo",
      onUpdate: () => {
        el.textContent = formatVND(target.value);
      },
      onComplete: () => {
        fromRef.current = to;
      },
    });
  }, [amount, duration]);

  return (
    <span ref={spanRef} className={cn("tabular-nums", className)}>
      {formatVND(amount)}
    </span>
  );
}

export default AnimatedCurrency;
