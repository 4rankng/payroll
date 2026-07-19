import { useState } from "react";
import { CalendarDays, Clock3, Eye, EyeOff, WalletCards } from "lucide-react";
import { cn } from "@/lib/utils";
import type { EmployeeHomeViewModel } from "@/utils/employeePortal/mobileHome";

const MASKED_AMOUNT = "••••••";

interface EmployeeWalletHeroProps {
  model: EmployeeHomeViewModel;
  className?: string;
}

export function EmployeeWalletHero({ model, className }: EmployeeWalletHeroProps) {
  const [amountVisible, setAmountVisible] = useState(true);

  return (
    <section
      className={cn(
        "ct-card employee-surface-card relative overflow-hidden border-neutral-content/10 bg-neutral text-neutral-content shadow-xl shadow-neutral/10",
        className
      )}
      aria-label={model.title}
    >
      <div className="ct-card-body gap-0 p-5 sm:p-6">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl border border-neutral-content/10 bg-neutral-content/5 text-neutral-content/70">
              <WalletCards className="h-5 w-5" strokeWidth={1.8} aria-hidden="true" />
            </span>
            <p className="employee-type-label-caps truncate text-neutral-content/55">
              {model.eyebrow}
            </p>
          </div>
          {model.periodLabel && (
            <span className="ct-badge ct-badge-ghost employee-type-pill h-auto shrink-0 gap-1.5 border border-neutral-content/10 bg-neutral-content/5 px-2.5 py-2 text-neutral-content/75">
              <CalendarDays className="h-3.5 w-3.5" strokeWidth={1.9} aria-hidden="true" />
              {model.periodLabel}
            </span>
          )}
        </div>

        <div className="mt-6 flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="employee-type-hero-title text-neutral-content">
              {model.title}
            </h2>
            <p className="employee-type-label mt-3 text-neutral-content/50">
              {model.amountLabel}
            </p>
          </div>
          <button
            type="button"
            onClick={() => setAmountVisible((visible) => !visible)}
            className="ct-btn ct-btn-ghost ct-btn-circle h-10 min-h-10 w-10 shrink-0 border border-neutral-content/10 bg-neutral-content/5 text-neutral-content/65 hover:bg-neutral-content/10 hover:text-neutral-content"
            aria-label={amountVisible ? "Ẩn số tiền" : "Hiện số tiền"}
            aria-pressed={!amountVisible}
          >
            {amountVisible ? (
              <Eye className="h-4.5 w-4.5" strokeWidth={1.9} />
            ) : (
              <EyeOff className="h-4.5 w-4.5" strokeWidth={1.9} />
            )}
          </button>
        </div>

        <p className="employee-type-hero-amount mt-1.5 break-words text-neutral-content tabular-nums">
          {amountVisible ? model.amount : MASKED_AMOUNT}
        </p>

        {!model.amountDescriptionLabel && (
          <p className="employee-type-body mt-2 max-w-md text-neutral-content/55">
            {model.amountDescription}
          </p>
        )}

        {(model.amountDescriptionLabel || model.metrics.length > 0) && (
          <div className="ct-stats mt-5 grid w-full grid-cols-2 overflow-hidden border border-neutral-content/10 bg-neutral-content/[0.04] sm:grid-cols-4">
            {model.amountDescriptionLabel && (
              <div className="ct-stat min-w-0 px-3 py-3.5">
                <CalendarDays className="ct-stat-figure h-4 w-4 text-neutral-content/45" aria-hidden="true" />
                <p className="ct-stat-title employee-type-label truncate text-neutral-content/45">
                  {model.amountDescriptionLabel}
                </p>
                <p className="ct-stat-value employee-type-inline-amount mt-0.5 truncate text-neutral-content tabular-nums">
                  {model.amountDescription}
                </p>
              </div>
            )}
            {model.metrics.map((metric) => (
              <div key={metric.label} className="ct-stat min-w-0 px-3 py-3.5">
                <Clock3 className="ct-stat-figure h-4 w-4 text-neutral-content/45" aria-hidden="true" />
                <p className="ct-stat-title employee-type-label truncate text-neutral-content/45">
                  {metric.label}
                </p>
                <p className="ct-stat-value employee-type-inline-amount mt-0.5 truncate text-neutral-content tabular-nums">
                  {amountVisible || !metric.sensitive
                    ? metric.value
                    : MASKED_AMOUNT}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
