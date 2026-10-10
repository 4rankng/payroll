import { Button } from '@/components/ui/button';
// W13a follow-up (approved): icon slots widened from lucide's LucideIcon to
// plain ComponentType so both lucide and @untitledui/icons fit — LucideIcon
// is structurally assignable to this; UU's plain-FC icons were not to the old type.
import type { ComponentType, SVGProps } from 'react';
import { TooltipProvider } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

interface PageHeaderAction {
  label: string;
  onClick: () => void;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'monochrome' | 'monochrome-outline';
  className?: string;
  disabled?: boolean;
}

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: PageHeaderAction[];
  children?: React.ReactNode;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  className?: string;
}

export const PageHeader = ({
  title,
  description,
  actions = [],
  children,
  icon: Icon,
  className,
}: PageHeaderProps) => {
  const hasControls = actions.length > 0 || !!children;

  return (
    <TooltipProvider>
      <section
        data-slot="page-header"
        data-admin-surface="header"
        className={cn(
          // Self-sufficient card surface: the page canvas is a dark sage for
          // WCAG 1.4.11 card/canvas separation, so header text must sit on a
          // light surface. Pages that already wrap this in AdminPageHeaderCard
          // get a seamless inner fill; pages that don't (timesheet, dashboard,
          // ledger, wallet) stay readable instead of falling onto the canvas.
          'admin-page-header flex flex-col gap-2 rounded-xl bg-card px-4 py-3 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between sm:gap-4',
          className,
        )}
      >
        <div className="min-w-0 flex-1 sm:min-w-64 sm:basis-64">
          <div className="flex items-center gap-2">
              {Icon && (
                <div data-slot="page-header-icon" className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-muted">
                  <Icon className="h-4 w-4 text-muted-foreground" />
                </div>
              )}
              <div className="min-w-0">
                <p data-slot="page-header-eyebrow" className="hidden">Quản trị</p>
                <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
                  {title}
                </h1>
              </div>
          </div>
          {description && (
            <p data-slot="page-header-description" className={cn(
              'mt-1 text-sm text-muted-foreground leading-normal',
              Icon ? 'sm:ml-9' : '',
            )}>
              {description}
            </p>
          )}
        </div>

        {hasControls && (
          <div
            data-slot="page-header-controls"
            className="flex max-w-full shrink-0 flex-wrap items-center gap-2 sm:justify-end"
          >
            {actions.map((action, index) => (
              <Button
                key={index}
                onClick={action.onClick}
                variant={action.variant || 'default'}
                size="sm"
                className={action.className}
                disabled={action.disabled}
              >
                {action.icon && <action.icon className="h-3.5 w-3.5 mr-1" />}
                {action.label}
              </Button>
            ))}
            {children}
          </div>
        )}
      </section>
    </TooltipProvider>
  );
};
