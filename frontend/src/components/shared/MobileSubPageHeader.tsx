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
 * Design: sticky, backdrop-blur, subtle border-bottom.
 * Replaces the duplicated inline header pattern across 5+ sub-pages.
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
        'sticky top-0 z-20 bg-background/95 backdrop-blur-lg supports-[backdrop-filter]:bg-background/80',
        'border-b border-border/30 shrink-0',
        className,
      )}
    >
      <div className="flex items-center gap-3 px-4 pt-4 pb-3">
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 shrink-0 -ml-1"
          onClick={onBack}
          aria-label="Quay lại"
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="flex items-center gap-2.5 min-w-0 flex-1">
          <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/8 shrink-0">
            <Icon className="h-[16px] w-[16px] text-primary/70" strokeWidth={2} />
          </div>
          <div className="min-w-0">
            <h1 className="font-display text-base font-bold text-foreground leading-tight truncate">
              {title}
            </h1>
            {subtitle && (
              <p className="text-xs text-muted-foreground leading-tight mt-0.5">
                {subtitle}
              </p>
            )}
          </div>
        </div>
        {actions && <div className="shrink-0">{actions}</div>}
      </div>
    </div>
  );
};
