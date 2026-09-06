import { cn } from '@/lib/utils';

/**
 * Shared admin page shell — the layout idiom established by the advance
 * payments workspace: an ambient emerald canvas, a glass header card, and
 * white instrument sections with a soft emerald ambient shadow. Extracted
 * here so list pages (users, projects, employees, payment history, ledger)
 * render the same frame without duplicating the long utility strings.
 */

/** Outer page gradient + centered content container. */
export const AdminPageCanvas = ({
  children,
  className,
  contentClassName,
}: {
  children: React.ReactNode;
  className?: string;
  /** Override the inner container width (e.g. narrower form pages). */
  contentClassName?: string;
}) => (
  <div
    className={cn(
      'min-h-full bg-[radial-gradient(circle_at_top_left,rgba(8,120,62,0.07),transparent_32rem),linear-gradient(180deg,rgba(248,250,249,0.96),#f5f7f9_34rem)]',
      className,
    )}
  >
    <div
      className={cn(
        'mx-auto max-w-[1480px] space-y-4 p-4 lg:space-y-5 lg:p-6',
        contentClassName,
      )}
    >
      {children}
    </div>
  </div>
);

/** Frosted card that carries the page title and header actions. */
export const AdminPageHeaderCard = ({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    data-mobile-header
    className={cn(
      'rounded-2xl border border-white/80 bg-white/82 px-4 py-3 shadow-[0_1px_2px_rgba(16,24,40,0.05),0_18px_48px_-32px_rgba(8,120,62,0.22)] backdrop-blur sm:px-5',
      className,
    )}
  >
    {children}
  </div>
);

/** White instrument section (stats band or operations card). */
export const AdminSectionCard = ({
  children,
  className,
  'aria-label': ariaLabel,
}: {
  children: React.ReactNode;
  className?: string;
  'aria-label'?: string;
}) => (
  <section
    aria-label={ariaLabel}
    data-mobile-stats
    className={cn(
      'overflow-hidden rounded-2xl border border-slate-200/80 bg-white shadow-[0_1px_2px_rgba(16,24,40,0.04),0_20px_56px_-42px_rgba(8,120,62,0.26)]',
      className,
    )}
  >
    {children}
  </section>
);

/** Toolbar strip inside an operations card — carries the list label and
 *  filters, separated from the table below by a hairline. */
export const AdminFilterRow = ({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={cn(
      'flex flex-wrap items-center justify-between gap-3 border-b border-slate-200/70 bg-white px-3 py-3 sm:px-5',
      className,
    )}
  >
    {children}
  </div>
);
