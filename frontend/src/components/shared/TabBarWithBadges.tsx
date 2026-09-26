import { memo } from 'react';
import { type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface TabWithBadges {
  id: string;
  label: string;
  icon?: LucideIcon;
  /** Total count badge (hidden when zero) */
  count?: number;
  /** Secondary accent badge (e.g. pending items) */
  pendingCount?: number;
  pendingLabel?: string;
}

export interface TabBarWithBadgesProps {
  tabs: TabWithBadges[];
  activeTab: string;
  onTabChange: (id: string) => void;
  className?: string;
}

/**
 * Underline tab bar with count badges.
 * Flat single-surface design: no outer pill container; the active tab carries
 * a 2px brand underline. Badges use one consistent muted style and are hidden
 * when zero so an empty tab never reads as actionable.
 */
export const TabBarWithBadges = memo(function TabBarWithBadges({
  tabs,
  activeTab,
  onTabChange,
  className,
}: TabBarWithBadgesProps) {
  return (
    <div className={cn('flex items-center gap-1 border-b border-border', className)}>
      {tabs.map((tab) => {
        const isActive = activeTab === tab.id;
        const Icon = tab.icon;
        return (
          <button
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            aria-pressed={isActive}
            className={cn(
              'relative flex min-h-10 items-center gap-1.5 rounded-t-lg px-3 text-sm font-medium transition-colors',
              isActive
                ? 'text-primary'
                : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
            )}
          >
            {Icon && <Icon className="h-4 w-4 shrink-0" />}
            {tab.label}
            {tab.count != null && tab.count > 0 && (
              <span
                className={cn(
                  'rounded-full px-1.5 text-xs tabular-nums font-semibold leading-5',
                  isActive
                    ? 'bg-primary/10 text-primary'
                    : 'bg-muted text-muted-foreground',
                )}
              >
                {tab.count}
              </span>
            )}
            {tab.pendingCount != null && tab.pendingCount > 0 && (
              <span className="rounded-full bg-warning/15 px-1.5 text-xs font-semibold leading-5 text-warning">
                {tab.pendingCount} {tab.pendingLabel ?? 'chờ'}
              </span>
            )}
            {isActive && (
              <span
                aria-hidden="true"
                className="absolute inset-x-0 bottom-0 h-0.5 rounded-full bg-primary"
              />
            )}
          </button>
        );
      })}
    </div>
  );
});
