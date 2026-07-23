import { useState } from "react";
import { CalendarDays, Clock3, Eye, EyeOff, WalletCards } from "lucide-react";
import { cn } from "@/lib/utils";
import type { EmployeeHomeViewModel } from "@/utils/employeePortal/mobileHome";

const MASKED_AMOUNT = "••••••";

// On-brand emerald hero surface (brand primary #08783e → #066534), deepened
// slightly for a richer premium-card feel. All stops are emerald shades.
const HERO_GRADIENT =
  "linear-gradient(150deg, #0a8f4d 0%, #08783e 46%, #065f33 100%)";

interface EmployeeWalletHeroProps {
  model: EmployeeHomeViewModel;
  className?: string;
}

export function EmployeeWalletHero({ model, className }: EmployeeWalletHeroProps) {
  const [amountVisible, setAmountVisible] = useState(true);

  return (
    <section
      className={cn(
        "ct-card relative isolate overflow-hidden rounded-[var(--employee-radius-card)] border border-white/12 text-neutral-content",
        className
      )}
      style={{ backgroundImage: HERO_GRADIENT, boxShadow: "var(--employee-cta-shadow)" }}
      aria-label={model.title}
    >
      {/* Decorative depth layers — all abstract, on-palette, no imagery. */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(circle_at_82%_-12%,rgba(255,255,255,0.32),transparent_60%)]"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-10 opacity-[0.10] [background-image:radial-gradient(rgba(255,255,255,0.85)_1px,transparent_0)] [background-size:16px_16px]"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-12 -top-14 -z-10 h-40 w-40 rounded-full bg-white/10 blur-2xl"
      />

      <div className="ct-card-body relative gap-0 p-5 sm:p-6">
        <div className="flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl border border-white/20 bg-white/15 text-white/90 backdrop-blur-sm">
              <WalletCards className="h-5 w-5" strokeWidth={1.8} aria-hidden="true" />
            </span>
            <p className="employee-type-label-caps truncate text-white/65">
              {model.eyebrow}
            </p>
          </div>
          {model.periodLabel && (
            <span className="ct-badge ct-badge-ghost employee-type-pill h-auto shrink-0 gap-1.5 border border-white/20 bg-white/15 px-2.5 py-2 text-white/85 backdrop-blur-sm">
              <CalendarDays className="h-3.5 w-3.5" strokeWidth={1.9} aria-hidden="true" />
              {model.periodLabel}
            </span>
          )}
        </div>

        <div className="mt-6 flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="employee-type-hero-title text-white">
              {model.title}
            </h2>
            <p className="employee-type-label mt-3 text-white/60">
              {model.amountLabel}
            </p>
          </div>
          <button
            type="button"
            onClick={() => setAmountVisible((visible) => !visible)}
            className="ct-btn ct-btn-ghost ct-btn-circle h-10 min-h-10 w-10 shrink-0 border border-white/20 bg-white/15 text-white/80 backdrop-blur-sm transition-colors hover:bg-white/25 hover:text-white"
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

        <p className="employee-type-hero-amount mt-1.5 break-words text-white tabular-nums [text-shadow:0_2px_12px_rgba(0,0,0,0.18)]">
          {amountVisible ? model.amount : MASKED_AMOUNT}
        </p>

        {!model.amountDescriptionLabel && (
          <p className="employee-type-body mt-2 max-w-md text-white/60">
            {model.amountDescription}
          </p>
        )}

        {(model.amountDescriptionLabel || model.metrics.length > 0) && (
          <div className="ct-stats mt-5 grid w-full grid-cols-2 overflow-hidden border border-white/15 bg-white/10 backdrop-blur-sm sm:grid-cols-4">
            {model.amountDescriptionLabel && (
              <div className="ct-stat min-w-0 px-3 py-3.5">
                <CalendarDays className="ct-stat-figure h-4 w-4 text-white/55" aria-hidden="true" />
                <p className="ct-stat-title employee-type-label truncate text-white/55">
                  {model.amountDescriptionLabel}
                </p>
                <p className="ct-stat-value employee-type-inline-amount mt-0.5 truncate text-white tabular-nums">
                  {model.amountDescription}
                </p>
              </div>
            )}
            {model.metrics.map((metric) => (
              <div key={metric.label} className="ct-stat min-w-0 px-3 py-3.5">
                <Clock3 className="ct-stat-figure h-4 w-4 text-white/55" aria-hidden="true" />
                <p className="ct-stat-title employee-type-label truncate text-white/55">
                  {metric.label}
                </p>
                <p className="ct-stat-value employee-type-inline-amount mt-0.5 truncate text-white tabular-nums">
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
