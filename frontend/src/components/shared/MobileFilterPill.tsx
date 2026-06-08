import { cn } from '@/lib/utils';

interface MobileFilterPillProps {
  children: React.ReactNode;
  /** Whether the pill is active/selected */
  active?: boolean;
  /** Click handler */
  onClick?: () => void;
  /** Optional class override */
  className?: string;
}

/**
 * Compact filter/label pill.
 * Used for info badges, schedule labels, bank info, etc.
 */
export const MobileFilterPill = ({
  children,
  active = false,
  onClick,
  className,
}: MobileFilterPillProps) => {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs transition-colors',
        active
          ? 'bg-primary/10 text-primary font-medium'
          : 'bg-muted text-muted-foreground',
        onClick && 'cursor-pointer hover:bg-muted/80 active:bg-muted',
        className,
      )}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {children}
    </span>
  );
};
