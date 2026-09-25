import { type LucideIcon } from 'lucide-react';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface MobileSubPageHeaderProps {
  /** Title of the sub-page */
  title: string;
  /** Optional subtitle (date, description, etc.) */
  subtitle?: string;
  /** Icon displayed in a soft primary container */
  icon: LucideIcon;
  /** Back navigation handler — typically `() => navigate(-1)` */
  onBack: () => void;
  /** Optional action button rendered on the right */
  actions?: React.ReactNode;
  /** Optional class override */
  className?: string;
}

/**
 * Premium sub-page header with back button, icon, title, and optional actions.
 * Used for drill-down pages like ActivityUsersPage, LendersPage, etc.
 *
 * Design: now shares the canonical MobilePageHeader visual language
 * (solid white surface, slate border, soft primary icon chip) for consistency.
 * The frosted-glass variant is retired in favor of an opaque surface that
 * matches the top-level page headers.
 */
export const MobileSubPageHeader = ({
  title,
  subtitle,
  icon: Icon,
  onBack,
  actions,
  className,
}: MobileSubPageHeaderProps) => {
  return (
    <div
      className={cn(
        'sticky top-0 z-20 bg-white',
        'border-b border-slate-200 shadow-[0_1px_0_rgba(15,23,42,0.04)] shrink-0',
        className,
      )}
      style={{ paddingTop: 'var(--mobile-header-top-padding, calc(env(safe-area-inset-top, 0px) + 1rem))' }}
    >
      <div className="flex items-center gap-2 px-3 py-2">
        <Button
          variant="ghost"
          size="icon"
          className="-ml-1 h-11 w-11 shrink-0 rounded-xl text-slate-700"
          onClick={onBack}
          aria-label="Quay lại"
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="hidden h-8 w-8 items-center justify-center rounded-lg border border-primary/10 bg-primary/[0.07] shrink-0 min-[420px]:flex">
          <Icon className="h-[18px] w-[18px] text-primary" strokeWidth={2} />
        </div>
        <div className="min-w-0 flex-1">
          <h1 className="break-words font-display text-lg font-extrabold leading-tight tracking-tight text-slate-950">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-0.5 break-words text-xs font-medium leading-normal text-slate-600">
              {subtitle}
            </p>
          )}
        </div>
        {actions && <div className="shrink-0 [&_button]:min-h-11 [&_button]:min-w-11 [&_button]:rounded-xl">{actions}</div>}
      </div>
    </div>
  );
};
