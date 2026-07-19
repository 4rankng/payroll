import { useState } from "react";
import {
  Bell,
  ClipboardList,
  CreditCard,
  Eye,
  EyeOff,
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
  EmployeeWalletMetric,
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

// Maps a metric to the mini-summary card style. Falls back to slate for any
// metric not explicitly flagged with a tone. Picks the first accent-toned and
// first amber-toned metric to feature as the two hero mini-cards.
function pickHeroMetrics(metrics: EmployeeWalletMetric[]): {
  accent?: EmployeeWalletMetric;
  amber?: EmployeeWalletMetric;
} {
  const accent = metrics.find((m) => (m.tone ?? "slate") === "employee");
  const amber = metrics.find((m) => m.tone === "amber");
  return { accent, amber };
}

const MASKED_AMOUNT = "••••••";

interface EmployeeWalletHeroProps {
  model: EmployeeHomeViewModel;
  onAction: (action: EmployeeQuickAction | EmployeeNudge) => void;
  className?: string;
}

export function EmployeeWalletHero({ model, onAction, className }: EmployeeWalletHeroProps) {
  const [amountVisible, setAmountVisible] = useState(true);

  const actionGridClass =
    model.quickActions.length <= 2
      ? "grid-cols-2"
      : model.quickActions.length <= 3
        ? "grid-cols-3"
        : "grid-cols-4";

  const heroMetrics = pickHeroMetrics(model.metrics);
  const primaryAction = model.quickActions[0];
  const secondaryActions = model.quickActions.slice(1);
  const PrimaryIcon = primaryAction ? ACTION_ICONS[primaryAction.icon] : null;

  return (
    <section
      className={cn(
        "ct-card employee-surface-card relative overflow-hidden bg-base-100",
        className
      )}
      aria-label={model.title}
    >
      {/* Premium header — gradient + subtle sheen + chip-style period pill, inspired by 21st.dev Wallet Card 2. */}
      <div className="relative bg-gradient-to-br from-primary to-neutral px-4 pb-9 pt-4 text-primary-content">
        {/* Decorative sheen: soft radial highlight, pointer-events-none so it never blocks taps. */}
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 opacity-60"
          style={{
            background:
              "radial-gradient(120% 80% at 85% -20%, rgba(255,255,255,0.28) 0%, rgba(255,255,255,0) 55%)",
          }}
        />
        <div className="relative flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="employee-type-label text-primary-content/80">
              {model.eyebrow}
            </p>
            <h2 className="employee-type-hero-title mt-0.5 text-primary-content">
              {model.title}
            </h2>
          </div>
          {model.periodLabel && (
            <span className="ct-badge ct-badge-outline employee-type-pill h-auto shrink-0 border-primary-content/25 bg-primary-content/10 px-2.5 py-1.5 text-primary-content backdrop-blur-sm">
              {model.periodLabel}
            </span>
          )}
        </div>
      </div>

      <div className="-mt-6 px-3 pb-3">
        <div className="ct-card-body gap-0 rounded-t-[var(--employee-radius-card)] bg-base-100 px-3 pb-1 pt-4">
          {/* Amount row with eye toggle — a signature wallet-card pattern for financial apps. */}
          <div className="flex items-center justify-between gap-2">
            <p className="employee-type-label-caps text-[var(--employee-text-secondary)]">
              {model.amountLabel}
            </p>
            <button
              type="button"
              onClick={() => setAmountVisible((v) => !v)}
              className="-mr-1 flex h-8 w-8 items-center justify-center rounded-full text-[var(--employee-text-secondary)] transition-colors hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-accent)] active:scale-95"
              aria-label={amountVisible ? "Ẩn số tiền" : "Hiện số tiền"}
              aria-pressed={!amountVisible}
            >
              {amountVisible ? (
                <Eye className="h-4 w-4" strokeWidth={2.2} />
              ) : (
                <EyeOff className="h-4 w-4" strokeWidth={2.2} />
              )}
            </button>
          </div>
          <p className="employee-type-hero-amount mt-1 break-words text-[var(--employee-accent)] tabular-nums transition-colors">
            {amountVisible ? model.amount : MASKED_AMOUNT}
          </p>
          <p className="employee-type-body mt-1.5 text-[var(--employee-text-secondary)]">
            {model.amountDescription}
          </p>

          {/* Featured mini-summary cards — adapted from 21st.dev Wallet Card 2's gradient sub-balance cards.
              Highlights the two most actionable metrics (accent + amber) when present, skipping the same
              rows in the full metrics list below to avoid duplication. */}
          {(heroMetrics.accent || heroMetrics.amber) && (
            <div className="mt-3 grid grid-cols-2 gap-2">
              {heroMetrics.accent && (
                <div
                  className={cn(
                    "rounded-[var(--employee-radius-control)] border px-3 py-2.5",
                    toneClass.employee
                  )}
                >
                  <p className="employee-type-label-caps text-[var(--employee-text-secondary)]">
                    {heroMetrics.accent.label}
                  </p>
                  <p className="employee-type-inline-amount mt-0.5 tabular-nums">
                    {amountVisible ? heroMetrics.accent.value : MASKED_AMOUNT}
                  </p>
                </div>
              )}
              {heroMetrics.amber && (
                <div
                  className={cn(
                    "rounded-[var(--employee-radius-control)] border px-3 py-2.5",
                    toneClass.amber
                  )}
                >
                  <p className="employee-type-label-caps text-[var(--employee-text-secondary)]">
                    {heroMetrics.amber.label}
                  </p>
                  <p className="employee-type-inline-amount mt-0.5 tabular-nums">
                    {amountVisible ? heroMetrics.amber.value : MASKED_AMOUNT}
                  </p>
                </div>
              )}
            </div>
          )}

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
                {model.metrics
                  .filter(
                    (metric) =>
                      metric !== heroMetrics.accent && metric !== heroMetrics.amber
                  )
                  .map((metric) => (
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
                        {amountVisible || (metric.tone ?? "slate") === "slate"
                          ? metric.value
                          : MASKED_AMOUNT}
                      </p>
                    </div>
                  ))}
              </div>
            </>
          )}

          {/* Primary CTA + secondary action grid — adapted from 21st.dev Wallet Card 2's Deposit/Withdraw row.
              First quick action is promoted to a prominent primary button; the rest stay as the compact grid. */}
          {primaryAction && (
            <button
              type="button"
              disabled={primaryAction.disabled}
              onClick={() => onAction(primaryAction)}
              className="ct-btn group mt-3 flex h-12 min-h-12 w-full items-center justify-center gap-2 rounded-[var(--employee-radius-control)] bg-[var(--employee-accent)] px-4 text-sm font-semibold text-white shadow-sm transition-transform active:scale-[0.99] disabled:cursor-not-allowed disabled:opacity-45"
            >
              {PrimaryIcon && <PrimaryIcon className="h-4 w-4" strokeWidth={2.2} />}
              {primaryAction.label}
            </button>
          )}

          {secondaryActions.length > 0 && (
            <div className={cn("mt-2 grid gap-1", actionGridClass)}>
              {secondaryActions.map((action) => {
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
          )}
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
