import {
  Bell,
  CircleDollarSign,
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
  limit: CircleDollarSign,
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
  const actionGridClass = model.quickActions.length <= 3 ? "grid-cols-3" : "grid-cols-4";

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
            <p className="text-[13px] font-semibold leading-5 text-white/80">
              {model.eyebrow}
            </p>
            <h2 className="mt-0.5 text-[24px] font-extrabold leading-8 tracking-normal text-white">
              {model.title}
            </h2>
          </div>
          {model.periodLabel && (
            <span className="shrink-0 rounded-full bg-white/15 px-3 py-1.5 text-[12px] font-bold leading-none text-white">
              {model.periodLabel}
            </span>
          )}
        </div>
      </div>

      <div className="-mt-9 px-4 pb-5">
        <div className="rounded-t-[30px] border border-white/90 bg-white px-4 pb-1 pt-5 shadow-[0_-12px_26px_-24px_rgba(15,23,42,0.7)]">
          <p className="text-[13px] font-semibold uppercase leading-5 text-slate-500">
            {model.amountLabel}
          </p>
          <p className="mt-1 break-words text-[34px] font-extrabold leading-none tracking-normal text-employee tabular-nums">
            {model.amount}
          </p>
          <p className="mt-2 text-[14px] font-medium leading-6 text-slate-500">
            {model.amountDescription}
          </p>

          {model.metrics.length > 0 && (
            <div className="mt-5 grid grid-cols-3 divide-x divide-slate-100 border-y border-slate-100 py-4">
              {model.metrics.map((metric) => (
                <div key={metric.label} className="min-w-0 px-3 first:pl-0 last:pr-0">
                  <p className="truncate text-[12px] font-semibold leading-4 text-slate-500">
                    {metric.label}
                  </p>
                  <p
                    className={cn(
                      "mt-1 truncate text-[14px] font-extrabold leading-5 tabular-nums",
                      metricToneClass[metric.tone ?? "slate"]
                    )}
                  >
                    {metric.value}
                  </p>
                </div>
              ))}
            </div>
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
                  <span className="w-full text-wrap text-[12px] font-bold leading-4 text-slate-700">
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
                    <span className="block truncate text-[14px] font-extrabold leading-5 text-slate-950">
                      {nudge.title}
                    </span>
                    <span className="mt-0.5 block text-[13px] font-medium leading-5 text-slate-500">
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
