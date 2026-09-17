import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';

export interface HeroMetric {
  label: string;
  value: string | number;
  unit?: string;
  icon?: LucideIcon;
  trend?: number;
}

export interface PremiumDashboardHeroProps {
  title: string;
  subtitle?: string;
  metrics?: HeroMetric[];
  action?: {
    label: string;
    onClick: () => void;
  };
  className?: string;
}

export const PremiumDashboardHero = ({
  title,
  subtitle,
  metrics = [],
  action,
  className,
}: PremiumDashboardHeroProps) => {
  return (
    <div className={cn('relative overflow-hidden rounded-2xl bg-gradient-navy p-6 lg:p-8 shadow-navy', className)}>
      {/* Decorative pattern overlay */}
      <div className="absolute inset-0 opacity-5">
        <div className="absolute inset-0" style={{
          backgroundImage: `radial-gradient(circle at 2px 2px, white 1px, transparent 0)`,
          backgroundSize: '32px 32px'
        }} />
      </div>

      <div className="relative z-10">
        {/* Header section */}
        <div className="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-6 mb-6">
          <div className="flex-1">
            <h1 className="font-display font-extrabold text-2xl lg:text-3xl text-white mb-2 tracking-tight">
              {title}
            </h1>
            {subtitle && (
              <p className="text-sm lg:text-base text-white/70 font-medium">
                {subtitle}
              </p>
            )}
          </div>

          {action && (
            <button
              onClick={action.onClick}
              className="inline-flex items-center gap-2 px-5 py-2.5 bg-white/10 backdrop-blur-sm text-white font-semibold rounded-lg border border-white/20 hover:bg-white/20 active:scale-[0.98] transition-all duration-200"
            >
              {action.label}
            </button>
          )}
        </div>

        {/* Metrics grid - asymmetric layout */}
        {metrics.length > 0 && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4">
            {metrics.map((metric, index) => {
              const formattedValue = typeof metric.value === 'number'
                ? metric.value.toLocaleString('vi-VN')
                : metric.value;

              return (
                <div
                  key={index}
                  className={cn(
                    'relative rounded-xl p-4 backdrop-blur-sm border transition-all duration-300',
                    'bg-white/5 border-white/10 hover:bg-white/10 hover:border-white/20',
                    'hover:scale-[1.02] group'
                  )}
                >
                  <div className="flex items-start justify-between gap-2 mb-2">
                    {metric.icon && (
                      <metric.icon className="h-4 w-4 text-white/60 group-hover:text-white/80 transition-colors" />
                    )}
                    {metric.trend !== undefined && (
                      <span className={cn(
                        'text-xs font-bold uppercase tracking-wider',
                        metric.trend >= 0 ? 'text-financial-positive' : 'text-financial-negative'
                      )}>
                        {metric.trend >= 0 ? '↑' : '↓'} {Math.abs(metric.trend)}%
                      </span>
                    )}
                  </div>
                  <div className="flex items-baseline gap-1">
                    <span className="font-display font-extrabold text-xl lg:text-2xl text-white tabular-nums tracking-tight">
                      {formattedValue}
                    </span>
                    {metric.unit && (
                      <span className="text-xs text-white/60 font-medium">
                        {metric.unit}
                      </span>
                    )}
                  </div>
                  <p className="text-xs font-medium text-white/50 uppercase tracking-wider mt-1 truncate">
                    {metric.label}
                  </p>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
