import { cn } from '@/lib/utils';

interface MobileCardProps {
  children: React.ReactNode;
  /** Visual variant */
  variant?: 'default' | 'elevated' | 'ghost';
  /** Padding size */
  padding?: 'sm' | 'default' | 'lg';
  /** Optional class override */
  className?: string;
  /** Click handler — makes card interactive */
  onClick?: () => void;
}

const paddingMap = {
  sm: 'p-2.5',
  default: 'p-3',
  lg: 'p-4',
};

/**
 * Consistent mobile card with rounded corners and subtle border.
 * Eliminates `rounded-xl border bg-card p-3` duplication across 20+ files.
 *
 * Variants:
 * - default: bordered card (most common)
 * - elevated: shadow + border for emphasis
 * - ghost: no border, muted background
 */
export const MobileCard = ({
  children,
  variant = 'default',
  padding = 'default',
  className,
  onClick,
}: MobileCardProps) => {
  const interactive = !!onClick;
  return (
    <div
      className={cn(
        'rounded-xl',
        variant === 'default' && 'border bg-card',
        variant === 'elevated' && 'border bg-card shadow-sm',
        variant === 'ghost' && 'bg-muted/40',
        paddingMap[padding],
        interactive && 'hover:bg-muted/50 active:bg-muted/70 transition-colors touch-manipulation cursor-pointer',
        className,
      )}
      onClick={onClick}
      role={interactive ? 'button' : undefined}
      tabIndex={interactive ? 0 : undefined}
    >
      {children}
    </div>
  );
};

interface MobileInfoCardProps {
  label: string;
  value: React.ReactNode;
  sublabel?: string;
  className?: string;
}

/**
 * Compact info card: label on top, value below.
 * Used for stat displays, summary cards, etc.
 */
export const MobileInfoCard = ({
  label,
  value,
  sublabel,
  className,
}: MobileInfoCardProps) => {
  return (
    <MobileCard className={cn('text-center', className)}>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-semibold tabular-nums mt-0.5">{value}</p>
      {sublabel && (
        <p className="text-xs text-muted-foreground/70 mt-0.5">{sublabel}</p>
      )}
    </MobileCard>
  );
};

interface MobileInfoRowProps {
  label: string;
  value: React.ReactNode;
  className?: string;
}

/**
 * Horizontal info row: label left, value right.
 * For inline display of key-value pairs.
 */
export const MobileInfoRow = ({
  label,
  value,
  className,
}: MobileInfoRowProps) => {
  return (
    <div className={cn('flex items-center justify-between gap-2', className)}>
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm font-medium tabular-nums text-right">{value}</span>
    </div>
  );
};
