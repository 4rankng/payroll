import { type ReactNode, type ElementType, memo } from 'react';
import { Info, AlertTriangle, XCircle, CheckCircle2 } from 'lucide-react';
import { cn } from '@/lib/utils';

// ─── Types ────────────────────────────────────────────────────────────────────

type CalloutVariant = 'info' | 'warning' | 'error' | 'success';

interface CalloutProps {
  /** Visual style variant */
  variant?: CalloutVariant;
  /** Override the default icon per variant */
  icon?: ElementType | null;
  /** Content — can be a string or rich JSX */
  children: ReactNode;
  /** Extra class names on the root */
  className?: string;
}

// ─── Variant config ───────────────────────────────────────────────────────────

const VARIANT_STYLES: Record<CalloutVariant, string> = {
  info: 'border-[#cfe0f3] bg-[#eef4fb] text-[#1f4e79]',
  warning: 'border-amber-200 bg-amber-50 text-amber-800',
  error: 'border-red-200 bg-red-50 text-red-800',
  success: 'border-emerald-200 bg-emerald-50 text-emerald-800',
};

const VARIANT_ICONS: Record<CalloutVariant, ElementType> = {
  info: Info,
  warning: AlertTriangle,
  error: XCircle,
  success: CheckCircle2,
};

// ─── Component ────────────────────────────────────────────────────────────────

/**
 * Reusable callout/banner for tips, warnings, errors, and success messages.
 *
 * Usage:
 * ```tsx
 * <Callout variant="info">
 *   Các ngày trong file sẽ được gán vào tháng đã chọn.
 * </Callout>
 * ```
 */
export const Callout = memo(function Callout({
  variant = 'info',
  icon,
  children,
  className,
}: CalloutProps) {
  const IconComponent = icon === null ? null : (icon ?? VARIANT_ICONS[variant]);

  return (
    <div
      className={cn(
        'flex items-start gap-2.5 rounded-[10px] border px-3 py-[11px]',
        VARIANT_STYLES[variant],
        className,
      )}
    >
      {IconComponent && (
        <IconComponent className="mt-px h-[17px] w-[17px] shrink-0 stroke-current" />
      )}
      <div className="text-[12.5px] font-medium leading-[1.5]">{children}</div>
    </div>
  );
});
