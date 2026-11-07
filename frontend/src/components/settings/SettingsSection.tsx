/**
 * Settings section primitives.
 *
 * The settings surface is a form, not a dashboard, so controls keep their
 * comfortable size (44px mobile / 36px desktop, 16px mobile inputs). Density
 * applies to the chrome: one panel per group, a 9px mono kicker, a 13px title,
 * and help text only where it explains a field. No 40px icon chips, no section
 * paragraph restating the title.
 */
import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface SettingsSectionProps {
  /** Domain label, e.g. "Chu kỳ tuần" or "Cảnh báo". */
  kicker: string;
  title: string;
  actions?: ReactNode;
  className?: string;
  children: ReactNode;
}

/** Panel with a header strip and hairline-divided rows. */
export function SettingsSection({
  kicker,
  title,
  actions,
  className,
  children,
}: SettingsSectionProps) {
  return (
    <section className={cn('console-panel min-w-0 overflow-hidden', className)}>
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border/70 bg-muted/40 px-3 py-2">
        <div className="min-w-0">
          <p className="console-kicker text-primary">{kicker}</p>
          <h2 className="mt-0.5 font-display text-[13px] font-semibold leading-5 tracking-tight text-foreground">
            {title}
          </h2>
        </div>
        {actions}
      </header>
      <div data-slot="settings-group" className="min-w-0 divide-y divide-border/60">
        {children}
      </div>
    </section>
  );
}

interface SettingsSectionHeaderProps {
  kicker: string;
  title: string;
  /** One short meta line — no paragraphs. */
  meta?: string;
  actions?: ReactNode;
  className?: string;
}

/** Header-only variant for sections whose body is already a set of panels. */
export function SettingsSectionHeader({
  kicker,
  title,
  meta,
  actions,
  className,
}: SettingsSectionHeaderProps) {
  return (
    <div
      className={cn(
        'flex flex-wrap items-end justify-between gap-3 border-b border-border/70 pb-2',
        className,
      )}
    >
      <div className="min-w-0">
        <p className="console-kicker text-primary">{kicker}</p>
        <h2 className="mt-0.5 font-display text-[13px] font-semibold leading-5 tracking-tight text-foreground">
          {title}
        </h2>
        {meta && <p className="mt-0.5 text-[11px] leading-4 text-muted-foreground">{meta}</p>}
      </div>
      {actions}
    </div>
  );
}
