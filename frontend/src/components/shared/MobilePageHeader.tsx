// W13a follow-up (approved): icon slot widened from LucideIcon so both lucide
// and @untitledui/icons components fit (UU's are plain FunctionComponents).
import type { ComponentType, SVGProps } from 'react';
import { ArrowLeft } from '@untitledui/icons';
import { Button } from '@/components/ui/button';
import { ThemeToggle } from '@/components/shared/ThemeToggle';
import { cn } from '@/lib/utils';

interface MobilePageHeaderProps {
  /** Page title — rendered in display font, bold */
  title: string;
  /** Optional subtitle below the title (date, record count, etc.) */
  subtitle?: React.ReactNode;
  /** Mono domain kicker above the title (console contract). */
  kicker?: string;
  /** Optional icon displayed in a soft colored container */
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  /** When provided, renders a back chevron button on the left.
   *  Collapses the role of MobileSubPageHeader for drill-down pages. */
  back?: () => void;
  /** Action buttons rendered on the right side */
  actions?: React.ReactNode;
  /** Layout for header actions. Compact inline placement is the default for mobile. */
  actionsLayout?: 'stacked' | 'inline';
  /** Whether the header sticks to top on scroll (default: false — the header
   *  scrolls away with the page so the full viewport belongs to content). */
  sticky?: boolean;
  /** Whether to show the bottom border (default: false — the header sits on
   *  the page surface as a large title, not as a separate app bar). */
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
  kicker,
  icon: Icon,
  back,
  actions,
  actionsLayout = 'inline',
  sticky = false,
  bordered = false,
  className,
}: MobilePageHeaderProps) => {
  // Admin routes carry the toto theme toggle in the title row on phones — the
  // viewport corner belongs to the OS status bar there. Partner/employee
  // mobile headers (no [data-admin-ui] wrapper) render without it.
  const showThemeToggle = typeof document !== 'undefined' && !!document.querySelector('[data-admin-ui]');
  return (
    <div
      className={cn(
        'z-20',
        sticky || bordered ? 'bg-card' : 'bg-transparent',
        bordered && 'border-b border-border',
        sticky && 'sticky top-0',
        'px-4 pb-2 pt-2.5',
        className,
      )}
      style={{
        paddingTop: sticky
          ? 'var(--mobile-header-top-padding, calc(env(safe-area-inset-top, 0px) + 1rem))'
          : 'var(--mobile-nonsticky-header-top-padding, var(--mobile-header-top-padding, calc(env(safe-area-inset-top, 0px) + 1rem)))',
      }}
    >
      <div
        className={cn(
          'w-full min-h-0 p-0 flex items-center justify-between gap-3 flex-wrap',
          // PAYR-38: the row must wrap at every width. The actions block owns a
          // full row (w-full) in stacked layout, so a nowrap row above sm left
          // the flex line with negative free space and crushed the title
          // block's min-w-0 to 0px — the h1 rendered one glyph per line
          // (B/à/n/g/…) on 640–767px viewports. Wrapping keeps the title row
          // full-width at all sizes.
        )}
      >
        <div
          className={cn(
            'flex flex-1 items-center gap-2.5',
            actionsLayout === 'inline' ? 'min-w-[10rem]' : 'min-w-0',
          )}
        >
          {back && (
            <Button
              variant="ghost"
              size="icon"
              className="-ml-1 h-11 w-11 shrink-0 rounded-xl text-foreground"
              onClick={back}
              aria-label="Quay lại"
            >
              <ArrowLeft className="h-5 w-5" aria-hidden="true" />
            </Button>
          )}
          {Icon && (
            <Icon
              className="h-5 w-5 shrink-0 text-muted-foreground"
              aria-hidden="true"
              strokeWidth={2}
            />
          )}
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
          {/* Stacked headers put the full width on the action row below, so a
              toggle living there always wraps onto its own line (the buttons
              div is w-full). Title-row placement is the established pattern —
              the wallet page carries the toggle the same way. Inline headers
              keep the toggle beside the actions. */}
          {actionsLayout === 'stacked' && showThemeToggle && (
            <div className="ml-auto flex shrink-0 items-center md:hidden [&_button]:h-11 [&_button]:w-11 [&_button]:min-h-0 [&_button]:rounded-xl [&_button]:p-0">
              <ThemeToggle />
            </div>
          )}
        </div>
        {(actions || (showThemeToggle && actionsLayout === 'inline')) && (
          <div
            className={cn(
              'flex flex-wrap items-center gap-2 [&_button]:min-h-11 [&_button]:min-w-11 [&_button]:rounded-xl',
              actionsLayout === 'inline'
                ? 'ml-auto basis-auto max-w-[52%] justify-end'
                : 'w-full justify-end',
            )}
          >
            {actions}
            {showThemeToggle && actionsLayout === 'inline' && (
              <div className="ml-auto flex shrink-0 items-center md:hidden [&_button]:h-11 [&_button]:w-11 [&_button]:min-h-0 [&_button]:rounded-xl [&_button]:p-0">
                <ThemeToggle />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
