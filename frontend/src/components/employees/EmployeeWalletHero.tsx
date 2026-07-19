import {
  Bell,
  ClipboardList,
  CreditCard,
  History,
  MapPinCheck,
  WalletCards,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";
import type {
  EmployeeActionIcon,
  EmployeeHomeViewModel,
  EmployeeNudge,
  EmployeeNudgeTone,
  EmployeeQuickAction,
} from "@/utils/employeePortal/mobileHome";

const ACTION_ICONS: Record<EmployeeActionIcon, LucideIcon> = {
  advance: WalletCards,
  attendance: MapPinCheck,
  history: History,
  account: CreditCard,
  timesheet: ClipboardList,
  notification: Bell,
};

const toneClass: Record<EmployeeNudgeTone, string> = {
  employee: "border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]",
  amber: "border-[var(--employee-warning-border)] bg-[var(--employee-warning-soft)] text-[var(--employee-warning)]",
  slate: "border-[var(--employee-border)] bg-[var(--employee-page)] text-[var(--employee-text-secondary)]",
};

const metricToneClass: Record<EmployeeNudgeTone, string> = {
  employee: "text-[var(--employee-accent)]",
  amber: "text-[var(--employee-warning)]",
  slate: "text-[var(--employee-text)]",
};

interface EmployeeWalletHeroProps {
  model: EmployeeHomeViewModel;
  onAction: (action: EmployeeQuickAction | EmployeeNudge) => void;
  className?: string;
}

export function EmployeeWalletHero({ model, onAction, className }: EmployeeWalletHeroProps) {
  const actionGridClass =
    model.quickActions.length <= 2
      ? "grid-cols-2"
      : model.quickActions.length <= 3
        ? "grid-cols-3"
        : "grid-cols-4";

  return (
    <section
      className={cn(
        "ct-card employee-surface-card overflow-hidden bg-base-100",
        className
      )}
      aria-label={model.title}
    >
      <div className="bg-gradient-to-br from-primary to-neutral px-4 pb-9 pt-4 text-primary-content">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="employee-type-label text-primary-content/80">
              {model.eyebrow}
            </p>
            <h2 className="employee-type-hero-title mt-0.5 text-primary-content">
              {model.title}
            </h2>
          </div>
          {model.periodLabel && (
            <span className="ct-badge ct-badge-outline employee-type-pill h-auto shrink-0 border-primary-content/25 bg-primary-content/10 px-2.5 py-1.5 text-primary-content">
              {model.periodLabel}
            </span>
          )}
        </div>
      </div>

      <div className="-mt-6 px-3 pb-3">
        <div className="ct-card-body gap-0 rounded-t-[var(--employee-radius-card)] bg-base-100 px-3 pb-1 pt-4">
          <p className="employee-type-label-caps text-[var(--employee-text-secondary)]">
            {model.amountLabel}
          </p>
          <p className="employee-type-hero-amount mt-1 break-words text-[var(--employee-accent)] tabular-nums">
            {model.amount}
          </p>
          <p className="employee-type-body mt-1.5 text-[var(--employee-text-secondary)]">
            {model.amountDescription}
          </p>

          {model.metrics.length > 0 && (
            <>
              {model.metricsTitle && (
                <p className="employee-type-label-caps mt-3 text-slate-500">
                  {model.metricsTitle}
                </p>
              )}
              <div
                className={cn(
                  "divide-y divide-slate-100 border-y border-slate-100",
                  model.metricsTitle ? "mt-1.5" : "mt-3"
                )}
              >
                {model.metrics.map((metric) => (
                  <div
                    key={metric.label}
                    className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-baseline gap-3 py-2"
                  >
                    <p className="employee-type-label min-w-0 text-[var(--employee-text-secondary)]">
                      {metric.label}
                    </p>
                    <p
                      className={cn(
                        "employee-type-inline-amount whitespace-nowrap text-right tabular-nums",
                        metricToneClass[metric.tone ?? "slate"]
                      )}
                    >
                      {metric.value}
                    </p>
                  </div>
                ))}
              </div>
            </>
          )}

          <div className={cn("mt-3 grid gap-1", actionGridClass)}>
            {model.quickActions.map((action) => {
              const Icon = ACTION_ICONS[action.icon];
              return (
                <button
                  key={action.id}
                  type="button"
                  disabled={action.disabled}
                  onClick={() => onAction(action)}
                  className="ct-btn ct-btn-ghost group h-auto min-h-11 min-w-0 justify-center gap-2 rounded-[var(--employee-radius-control)] px-2 py-1.5 text-left font-normal normal-case shadow-none disabled:cursor-not-allowed disabled:opacity-45"
                >
                  <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-[var(--employee-accent-soft)] text-[var(--employee-accent)] transition-colors group-active:bg-[var(--employee-accent)] group-active:text-white">
                    <Icon className="h-4 w-4" strokeWidth={2.2} />
                  </span>
                  <span className="employee-type-label min-w-0 text-wrap text-[var(--employee-text)]">
                    {action.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {model.nudges.length > 0 && (
          <div className="mt-2 divide-y divide-slate-100 border-t border-slate-100">
            {model.nudges.slice(0, 3).map((nudge) => {
              const Icon = ACTION_ICONS[nudge.icon];
              return (
                <button
                  key={nudge.id}
                  type="button"
                  onClick={() => onAction(nudge)}
                  className="flex min-h-11 w-full items-center gap-2.5 px-1 py-2 text-left transition-transform active:scale-[0.99]"
                >
                  <span
                    className={cn(
                      "flex h-9 w-9 shrink-0 items-center justify-center rounded-full border",
                      toneClass[nudge.tone]
                    )}
                  >
                    <Icon className="h-5 w-5" />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="employee-type-row-amount block truncate text-slate-950">
                      {nudge.title}
                    </span>
                    <span className="employee-type-body-sm mt-0.5 block text-slate-500">
                      {nudge.description}
                    </span>
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </section>
  );
}
