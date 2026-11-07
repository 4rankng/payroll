import type { SVGProps } from 'react';
import { ArrowLeft } from '@untitledui/icons';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface MobileSubPageHeaderProps {
  /** Title of the sub-page */
  title: string;
  /** Optional subtitle (date, description, etc.) */
  subtitle?: string;
  /** Mono domain kicker above the title (console contract) */
  kicker?: string;
  /** Icon displayed in a soft primary container — widened from LucideIcon so
   * both lucide and @untitledui/icons components fit (W13a contract). */
  icon: React.ComponentType<SVGProps<SVGSVGElement>>;
  /** Back navigation handler — typically `() => navigate(-1)` */
  onBack: () => void;
  /** Optional action button rendered on the right */
  actions?: React.ReactNode;
  /** Optional class override */
  className?: string;
}

/**
 * Sub-page header with back button, kicker, icon, title, and optional actions.
 * Used for drill-down pages like ActivityUsersPage, LendersPage, etc.
 *
 * Console contract: card surface, hairline border, 20px bold title, mono
 * kicker, ≥44px back button and actions.
 */
export const MobileSubPageHeader = ({
  title,
  subtitle,
  kicker,
  icon: Icon,
  onBack,
  actions,
  className,
}: MobileSubPageHeaderProps) => {
  return (
    <div
      className={cn(
        'sticky top-0 z-20 bg-card',
        'border-b border-border shrink-0',
        className,
      )}
      style={{ paddingTop: 'var(--mobile-header-top-padding, calc(env(safe-area-inset-top, 0px) + 1rem))' }}
    >
      <div className="flex items-center gap-2 px-3 py-2">
        <Button
          variant="ghost"
          size="icon"
          className="-ml-1 h-11 w-11 shrink-0 rounded-xl text-foreground"
          onClick={onBack}
          aria-label="Quay lại"
        >
          <ArrowLeft className="h-5 w-5" aria-hidden="true" />
        </Button>
        <div className="hidden h-8 w-8 items-center justify-center rounded-lg border border-primary/10 bg-primary/[0.07] shrink-0 min-[420px]:flex">
          <Icon className="h-[18px] w-[18px] text-primary" strokeWidth={2} />
        </div>
        <div className="min-w-0 flex-1">
          {kicker && <p className="console-kicker mb-1">{kicker}</p>}
          <h1 className="break-words font-display text-xl font-bold leading-tight tracking-tight text-foreground">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-0.5 break-words text-xs font-medium leading-normal text-muted-foreground">
              {subtitle}
            </p>
          )}
        </div>
        {actions && <div className="shrink-0 [&_button]:min-h-11 [&_button]:min-w-11 [&_button]:rounded-xl">{actions}</div>}
      </div>
    </div>
  );
};
