import { Button } from '@/components/ui/button';
import { SidebarTrigger, useSidebarOptional } from '@/components/ui/sidebar';
import { LucideIcon } from 'lucide-react';
import { TooltipProvider } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

/**
 * Sidebar show/hide rendered inside every page header. Null-safe: outside a
 * SidebarProvider (isolated tests) it renders nothing instead of throwing.
 */
function PageSidebarTrigger() {
  const sidebar = useSidebarOptional();
  if (!sidebar) return null;
  return <SidebarTrigger className="h-8 w-8 shrink-0" />;
}

interface PageHeaderAction {
  label: string;
  onClick: () => void;
  icon?: LucideIcon;
  variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link' | 'monochrome' | 'monochrome-outline';
  className?: string;
  disabled?: boolean;
}

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: PageHeaderAction[];
  children?: React.ReactNode;
  icon?: LucideIcon;
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
          'admin-page-header flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-start sm:justify-between sm:gap-4',
          className,
        )}
      >
        <div className="min-w-0 flex-1 sm:min-w-64 sm:basis-64">
          <div className="flex items-center gap-2">
              {/* Persistent sidebar show/hide — replaces the floating toggle tab */}
              <PageSidebarTrigger />
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
