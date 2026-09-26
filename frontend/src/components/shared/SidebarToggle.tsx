import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import { SidebarTrigger, useSidebarOptional } from '@/components/ui/sidebar';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

interface SidebarToggleProps {
  className?: string;
}

/**
 * Sidebar show/hide control, hosted by the sidebar header itself — navigation
 * chrome, never a page title.
 *
 * - Expanded: right edge of the sidebar header, beside the brand.
 * - Collapsed: the top control of the 44px icon rail; restores the sidebar.
 * - Keyboard: Ctrl/Cmd+B toggles the sidebar (SidebarProvider); Escape closes
 *   the off-canvas drawer that the same shortcut opens on phones.
 *
 * 40x40 at 768-1199px (the repo's tablet touch floor in
 * styles/variables.css:454), 32x32 from 1200px.
 * Reports aria-expanded so assistive tech announces the navigation state.
 * Null-safe: renders nothing outside a SidebarProvider.
 */
export const SidebarToggle = ({ className }: SidebarToggleProps) => {
  const sidebar = useSidebarOptional();
  if (!sidebar) return null;

  const expanded = sidebar.open;
  const label = expanded ? 'Ẩn điều hướng' : 'Hiện điều hướng';

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <SidebarTrigger
          aria-label={label}
          aria-expanded={expanded}
          icon={expanded ? <PanelLeftClose className="h-4 w-4" /> : <PanelLeftOpen className="h-4 w-4" />}
          className={cn(
            'h-10 w-10 shrink-0 text-sidebar-foreground/70 hover:bg-card/10 hover:text-sidebar-foreground',
            'sm:h-10 sm:w-10',
            'min-[1200px]:h-8 min-[1200px]:w-8',
            className,
          )}
        />
      </TooltipTrigger>
      <TooltipContent side="right">{label} · Ctrl+B</TooltipContent>
    </Tooltip>
  );
};
