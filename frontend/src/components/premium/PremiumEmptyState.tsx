import { cn } from '@/lib/utils';
import { LucideIcon } from 'lucide-react';

export interface PremiumEmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  className?: string;
  variant?: 'default' | 'compact';
}

export const PremiumEmptyState = ({
  icon: Icon,
  title,
  description,
  action,
  className,
  variant = 'default',
}: PremiumEmptyStateProps) => {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center',
        variant === 'default' ? 'py-16 px-6' : 'py-12 px-4',
        className
      )}
    >
      {/* Icon with animated background */}
      {Icon && (
        <div className="relative mb-6">
          <div className="absolute inset-0 bg-primary/5 rounded-full blur-xl animate-pulse" />
          <div className="relative bg-gradient-to-br from-primary/10 to-primary/5 rounded-2xl p-4 shadow-soft">
            <Icon className="h-8 w-8 text-primary" />
          </div>
        </div>
      )}

      {/* Title */}
      <h3 className="font-display font-bold text-lg text-foreground mb-2">
        {title}
      </h3>

      {/* Description */}
      <p className="text-sm text-muted-foreground max-w-md mb-6 leading-relaxed">
        {description}
      </p>

      {/* Action button */}
      {action && (
        <button
          onClick={action.onClick}
          className="inline-flex items-center gap-2 px-5 py-2.5 bg-gradient-navy text-white font-semibold rounded-lg shadow-navy hover:shadow-lg hover:scale-[1.02] active:scale-[0.98] transition-all duration-200"
        >
          <span>{action.label}</span>
        </button>
      )}
    </div>
  );
};
