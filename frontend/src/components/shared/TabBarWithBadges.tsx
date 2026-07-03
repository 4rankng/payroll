import { memo, type ReactNode } from 'react';
import { type LucideIcon } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface TabWithBadges {
  id: string;
  label: string;
  icon?: LucideIcon;
  /** Total count badge */
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
 * Segmented tab bar with optional count and pending badges.
 * Implements the tab bar pattern from AdvancePaymentsPage.
 */
export const TabBarWithBadges = memo(function TabBarWithBadges({
  tabs,
  activeTab,
  onTabChange,
  className,
}: TabBarWithBadgesProps) {
  return (
    <div
      className={cn(
        'flex items-center gap-px rounded-xl border border-border/60 bg-card p-0.5 shrink-0',
        className,
      )}
    >
      {tabs.map((tab) => {
        const isActive = activeTab === tab.id;
        const Icon = tab.icon;
        return (
          <button
            key={tab.id}
            onClick={() => onTabChange(tab.id)}
            aria-pressed={isActive}
            className={cn(
              'flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-sm font-medium transition-all whitespace-nowrap',
              isActive
                ? 'bg-primary text-primary-foreground shadow-sm'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted',
            )}
          >
            {Icon && <Icon className="h-3.5 w-3.5 shrink-0" />}
            {tab.label}
            {tab.count != null && (
              <CountBadge value={tab.count} active={isActive} />
            )}
            {tab.pendingCount != null && tab.pendingCount > 0 && (
              <PendingBadge
                value={tab.pendingCount}
                label={tab.pendingLabel ?? 'chờ'}
                active={isActive}
              />
            )}
          </button>
        );
      })}
    </div>
  );
});

const CountBadge = memo(function CountBadge({
  value,
  active,
}: {
  value: number;
  active: boolean;
}) {
  return (
    <span
      className={cn(
        'tabular-nums text-xs px-1.5 py-px rounded-full font-semibold',
        active
          ? 'bg-primary-foreground/20 text-primary-foreground'
          : 'bg-muted-foreground/20 text-muted-foreground',
      )}
    >
      {value}
    </span>
  );
});

const PendingBadge = memo(function PendingBadge({
  value,
  label,
  active,
}: {
  value: number;
  label: string;
  active: boolean;
}) {
  return (
    <span
      className={cn(
        'tabular-nums text-xs px-1.5 py-px rounded-full font-semibold',
        active
          ? 'bg-amber-300/30 text-amber-100'
          : 'bg-amber-400/15 text-amber-600',
      )}
    >
      {value} {label}
    </span>
  );
});
