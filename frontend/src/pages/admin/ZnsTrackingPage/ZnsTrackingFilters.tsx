import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  ZNS_STATUSES,
  ZNS_FEATURES,
  ZNS_STATUS_LABELS,
  ZNS_FEATURE_LABELS,
} from '@/types/api/zns.types';
import type { ZnsMessagesQueryParams } from '@/types/api/zns.types';
import { cn } from '@/lib/utils';

interface ZnsTrackingFiltersProps {
  filters: ZnsMessagesQueryParams;
  onChange: (filters: ZnsMessagesQueryParams) => void;
}

/**
 * Status is a multi-select chip row (the tracking table mixes sent/failed with
 * the backfilled queue states); feature is a single-select dropdown. Date
 * range and free-text search live next to this in the filter row.
 */
export function ZnsTrackingFilters({ filters, onChange }: ZnsTrackingFiltersProps) {
  const toggleStatus = (status: string) => {
    const selected = filters.status ?? [];
    onChange({
      ...filters,
      status: selected.includes(status)
        ? selected.filter((s) => s !== status)
        : [...selected, status],
    });
  };

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="flex flex-wrap gap-1.5" role="group" aria-label="Lọc trạng thái">
        {ZNS_STATUSES.map((status) => {
          const isActive = filters.status?.includes(status) ?? false;
          return (
            <button
              key={status}
              type="button"
              aria-pressed={isActive}
              onClick={() => toggleStatus(status)}
              className={cn(
                'inline-flex min-h-11 items-center rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors',
                isActive
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-utility-gray-300 bg-white text-fg-secondary hover:bg-utility-gray-50'
              )}
            >
              {ZNS_STATUS_LABELS[status] ?? status}
            </button>
          );
        })}
      </div>

      <Select
        value={filters.feature ?? 'all'}
        onValueChange={(value) =>
          onChange({ ...filters, feature: value === 'all' ? undefined : value })
        }
      >
        <SelectTrigger className="min-h-11 w-[190px] text-xs" aria-label="Lọc tính năng">
          <SelectValue placeholder="Tính năng" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Mọi tính năng</SelectItem>
          {ZNS_FEATURES.map((feature) => (
            <SelectItem key={feature} value={feature}>
              {ZNS_FEATURE_LABELS[feature] ?? feature}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {(filters.status?.length ?? 0) > 0 && (
        <Badge variant="outline" className="gap-1 text-xs">
          {filters.status!.length} trạng thái
        </Badge>
      )}
    </div>
  );
}
