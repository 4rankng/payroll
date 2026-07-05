import { type LucideIcon } from 'lucide-react';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface MobilePageHeaderProps {
  /** Page title — rendered in display font, extrabold */
  title: string;
  /** Optional subtitle below the title (date, record count, etc.) */
  subtitle?: React.ReactNode;
  /** Optional icon displayed in a soft colored container */
  icon?: LucideIcon;
  /** When provided, renders a back chevron button on the left.
   *  Collapses the role of MobileSubPageHeader for drill-down pages. */
  back?: () => void;
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
 * Premium mobile page header — single canonical header for the admin mobile app.
 *
 * Design: sticky by default, opaque white surface, subtle border-bottom,
 * soft primary icon chip, display-font title. Optional `back` handler turns
 * it into a drill-down header (replaces MobileSubPageHeader usage).
 *
 * Typography: font-display text-[21px] font-extrabold tracking-tight.
 */
export const MobilePageHeader = ({
  title,
  subtitle,
  icon: Icon,
  back,
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
      style={{
        paddingTop: sticky
          ? 'var(--mobile-header-top-padding, calc(env(safe-area-inset-top, 0px) + 1rem))'
          : 'var(--mobile-nonsticky-header-top-padding, var(--mobile-header-top-padding, calc(env(safe-area-inset-top, 0px) + 1rem)))',
      }}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          {back && (
            <Button
              variant="ghost"
              size="icon"
              className="-ml-1 h-11 w-11 shrink-0 rounded-xl text-slate-700"
              onClick={back}
              aria-label="Quay lại"
            >
              <ArrowLeft className="h-5 w-5" />
            </Button>
          )}
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
        {actions && (
          <div className="flex max-w-[58%] shrink-0 flex-wrap items-center justify-end gap-2 [&_button]:min-h-11 [&_button]:min-w-11 [&_button]:rounded-xl">
            {actions}
          </div>
        )}
      </div>
    </div>
  );
};
