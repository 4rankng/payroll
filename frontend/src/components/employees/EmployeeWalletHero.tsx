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
  employee: "border-employee/20 bg-employee/10 text-employee",
  amber: "border-amber-200 bg-amber-50 text-amber-700",
  slate: "border-slate-200 bg-slate-50 text-slate-600",
};

const metricToneClass: Record<EmployeeNudgeTone, string> = {
  employee: "text-employee",
  amber: "text-amber-700",
  slate: "text-slate-900",
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
        "overflow-hidden rounded-[32px] border border-white/80 bg-white/95 shadow-[0_20px_50px_-34px_rgba(15,23,42,0.8)]",
        className
      )}
      aria-label={model.title}
    >
      <div className="bg-employee px-5 pb-14 pt-5 text-white">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="employee-type-label text-white/80">
              {model.eyebrow}
            </p>
            <h2 className="employee-type-hero-title mt-0.5 text-white">
              {model.title}
            </h2>
          </div>
          {model.periodLabel && (
            <span className="employee-type-pill shrink-0 rounded-full bg-white/15 px-3 py-1.5 text-white">
              {model.periodLabel}
            </span>
          )}
        </div>
      </div>

      <div className="-mt-9 px-4 pb-5">
        <div className="rounded-t-[30px] bg-white px-4 pb-1 pt-5">
          <p className="employee-type-label-caps text-slate-500">
            {model.amountLabel}
          </p>
          <p className="employee-type-hero-amount mt-1 break-words text-employee tabular-nums">
            {model.amount}
          </p>
          <p className="employee-type-body mt-2 text-slate-500">
            {model.amountDescription}
          </p>

          {model.metrics.length > 0 && (
            <>
              {model.metricsTitle && (
                <p className="employee-type-label-caps mt-5 text-slate-500">
                  {model.metricsTitle}
                </p>
              )}
              <div
                className={cn(
                  "divide-y divide-slate-100 border-y border-slate-100",
                  model.metricsTitle ? "mt-2" : "mt-5"
                )}
              >
                {model.metrics.map((metric) => (
                  <div
                    key={metric.label}
                    className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-baseline gap-3 py-2.5"
                  >
                    <p className="employee-type-label min-w-0 text-slate-500">
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

          <div className={cn("mt-5 grid gap-1", actionGridClass)}>
            {model.quickActions.map((action) => {
              const Icon = ACTION_ICONS[action.icon];
              return (
                <button
                  key={action.id}
                  type="button"
                  disabled={action.disabled}
                  onClick={() => onAction(action)}
                  className="group flex min-h-[74px] min-w-0 flex-col items-center justify-center gap-2 px-1 py-2 text-center transition-all active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-45"
                >
                  <span className="flex h-11 w-11 items-center justify-center rounded-full bg-employee/10 text-employee transition-colors group-active:bg-employee group-active:text-white">
                    <Icon className="h-5 w-5" strokeWidth={2.2} />
                  </span>
                  <span className="employee-type-label w-full text-wrap text-slate-700">
                    {action.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {model.nudges.length > 0 && (
          <div className="mt-4 divide-y divide-slate-100 border-t border-slate-100">
            {model.nudges.slice(0, 3).map((nudge) => {
              const Icon = ACTION_ICONS[nudge.icon];
              return (
                <button
                  key={nudge.id}
                  type="button"
                  onClick={() => onAction(nudge)}
                  className="flex min-h-[64px] w-full items-center gap-3 px-1 py-3 text-left transition-transform active:scale-[0.99]"
                >
                  <span
                    className={cn(
                      "flex h-10 w-10 shrink-0 items-center justify-center rounded-full border",
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
