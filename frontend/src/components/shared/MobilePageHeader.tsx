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
 * Premium mobile page header with solid surfaces, consistent typography,
 * and optional icon + subtitle + action buttons.
 *
 * Design: sticky by default, opaque background, subtle border-bottom.
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
        'z-20 bg-white',
        bordered && 'border-b border-slate-200 shadow-[0_1px_0_rgba(15,23,42,0.04)]',
        sticky && 'sticky top-0',
        'px-4 pb-3 pt-4',
        className,
      )}
      style={{ paddingTop: 'max(1rem, env(safe-area-inset-top, 1rem))' }}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2.5">
            {Icon && (
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-primary/10 bg-primary/[0.07] shadow-sm">
                <Icon className="h-[18px] w-[18px] text-primary/75" strokeWidth={2} />
              </div>
            )}
            <div className="min-w-0">
              <h1 className="truncate font-display text-[21px] font-extrabold leading-tight tracking-tight text-slate-950">
                {title}
              </h1>
              {subtitle && (
                <p className="mt-0.5 text-xs font-medium leading-normal text-slate-500">
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
