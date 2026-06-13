import { type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

interface MobilePageHeaderProps {
  /** Page title — rendered in display font, extrabold */
  title: string;
  /** Optional subtitle below the title (date, record count, etc.) */
  subtitle?: React.ReactNode;
  /** Optional icon displayed in a soft colored container */
  icon?: LucideIcon;
  /** Action buttons rendered on the right side */
  actions?: React.ReactNode;
  /** Whether the header sticks to top on scroll (default: true) */
  sticky?: boolean;
  /** Whether to show the bottom border (default: true) */
  bordered?: boolean;
  /** Optional class override */
  className?: string;
}

/**
 * Premium mobile page header with backdrop blur, consistent typography,
 * and optional icon + subtitle + action buttons.
 *
 * Design: sticky by default, backdrop-blur, subtle border-bottom.
 * Typography: font-display text-[22px] font-extrabold tracking-tight.
 */
export const MobilePageHeader = ({
  title,
  subtitle,
  icon: Icon,
  actions,
  sticky = true,
  bordered = true,
  className,
}: MobilePageHeaderProps) => {
  return (
    <div
      className={cn(
        'z-20 bg-background/95 backdrop-blur-lg supports-[backdrop-filter]:bg-background/80',
        bordered && 'border-b border-border/30',
        sticky && 'sticky top-0',
        'px-4 pt-4 pb-3',
        className,
      )}
      style={{ paddingTop: 'max(1rem, env(safe-area-inset-top, 1rem))' }}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2.5">
            {Icon && (
              <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/5 shrink-0">
                <Icon className="h-[18px] w-[18px] text-primary/70" strokeWidth={2} />
              </div>
            )}
            <div className="min-w-0">
              <h1 className="font-display text-[22px] font-extrabold text-foreground tracking-tight leading-tight truncate">
                {title}
              </h1>
              {subtitle && (
                <p className="text-xs font-medium text-muted-foreground mt-0.5 leading-normal">
                  {subtitle}
                </p>
              )}
            </div>
          </div>
        </div>
        {actions && (
          <div className="flex items-center gap-2 shrink-0">
            {actions}
          </div>
        )}
      </div>
    </div>
  );
};
