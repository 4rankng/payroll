import { useState, useMemo } from 'react';
import { ClipboardList, Filter, X, Check } from 'lucide-react';
import { useInfiniteAuditLogs } from '@/hooks/api/useAuditLogs';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll.tsx';
import { AuditLogCard, AuditLogCardSkeleton } from '../../../admin/AuditLogPage/AuditLogCard';
import { AuditLogDetailSheet } from '../../../admin/AuditLogPage/AuditLogDetailSheet';
import { MobilePageShell } from '@/components/shared/MobilePageShell';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { MobileSectionHeader } from '@/components/shared/MobileSectionHeader';
import { EmptyState } from '@/components/shared/EmptyState';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { VIETNAMESE_AUDIT_LABELS } from '@/types/api/audit.types';
import type { AuditLogsQueryParams } from '@/types/api/audit.types';
import { cn } from '@/lib/utils';

const ALL_ACTIONS = Object.keys(VIETNAMESE_AUDIT_LABELS.actions) as string[];
const ALL_ENTITY_TYPES = Object.keys(VIETNAMESE_AUDIT_LABELS.entities) as string[];

const EMPTY_FILTERS: AuditLogsQueryParams = { page: 1, pageSize: 30 };

/** Count of active filter dimensions for the badge. */
function countActive(f: AuditLogsQueryParams): number {
  return (
    (f.action?.length ?? 0) +
    (f.entityType?.length ?? 0) +
    (f.fromDate ? 1 : 0) +
    (f.toDate ? 1 : 0)
  );
}

/** Tappable multi-select chip row for a filter dimension. */
function FilterChipGroup({
  options,
  selected,
  labels,
  onToggle,
}: {
  options: string[];
  selected: string[];
  labels: Record<string, string>;
  onToggle: (value: string) => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((opt) => {
        const isActive = selected.includes(opt);
        return (
          <button
            key={opt}
            type="button"
            onClick={() => onToggle(opt)}
            className={cn(
              'inline-flex min-h-11 items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-semibold transition-colors touch-manipulation active:scale-95',
              isActive
                ? 'border-primary bg-primary text-primary-foreground'
                : 'border-slate-200 bg-white text-slate-600',
            )}
          >
            {isActive && <Check className="h-3 w-3" />}
            {labels[opt] ?? opt}
          </button>
        );
      })}
    </div>
  );
}

export default function AuditLogPageMobile() {
  const [selectedLogId, setSelectedLogId] = useState<number | null>(null);
  const [filters, setFilters] = useState<AuditLogsQueryParams>(EMPTY_FILTERS);
  const [isFilterOpen, setIsFilterOpen] = useState(false);

  // Strip pagination before passing — the hook manages page internally.
  const { data, isLoading, isFetchingNextPage, fetchNextPage, hasNextPage } =
    useInfiniteAuditLogs({
      fromDate: filters.fromDate,
      toDate: filters.toDate,
      action: filters.action,
      entityType: filters.entityType,
    });

  const logs = useMemo(
    () => data?.pages.flatMap((p) => p.data ?? []) ?? [],
    [data],
  );
  const totalRecords = data?.pages[0]?.pagination?.totalRecords ?? 0;

  const { observerRef } = useInfiniteScroll({
    hasMore: hasNextPage ?? false,
    isLoading: isFetchingNextPage,
    onLoadMore: () => {
      fetchNextPage();
    },
    rootMargin: '300px',
    threshold: 0.1,
  });

  const activeCount = countActive(filters);

  const toggleAction = (a: string) =>
    setFilters((prev) => ({
      ...prev,
      page: 1,
      action: prev.action?.includes(a)
        ? prev.action.filter((x) => x !== a)
        : [...(prev.action ?? []), a],
    }));

  const toggleEntity = (et: string) =>
    setFilters((prev) => ({
      ...prev,
      page: 1,
      entityType: prev.entityType?.includes(et)
        ? prev.entityType.filter((x) => x !== et)
        : [...(prev.entityType ?? []), et],
    }));

  const reset = () => setFilters(EMPTY_FILTERS);

  return (
    <MobilePageShell>
      <MobilePageHeader
        title="Nhật ký hoạt động"
        icon={ClipboardList}
        subtitle={
          !isLoading && totalRecords > 0
            ? `${totalRecords.toLocaleString('vi-VN')} bản ghi`
            : undefined
        }
        actions={
          <Button
            variant="outline"
            size="sm"
            className="min-h-11 gap-1.5 border-slate-200 bg-white px-3 text-xs font-semibold text-slate-700"
            onClick={() => setIsFilterOpen(true)}
          >
            <Filter className="h-3.5 w-3.5" />
            Lọc
            {activeCount > 0 && (
              <Badge className="h-4 min-w-4 border-0 bg-primary px-1 text-[11px] text-primary-foreground">
                {activeCount}
              </Badge>
            )}
          </Button>
        }
      />

      {/* Active-filter summary chips (shown only when filtering) */}
      {activeCount > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 px-4 pb-2 pt-3">
          {filters.fromDate && (
            <Badge variant="secondary" className="gap-1 text-[11px]">
              Từ {filters.fromDate}
            </Badge>
          )}
          {filters.toDate && (
            <Badge variant="secondary" className="gap-1 text-[11px]">
              Đến {filters.toDate}
            </Badge>
          )}
          {(filters.action ?? []).map((a) => (
            <Badge key={a} variant="secondary" className="text-[11px]">
              {(VIETNAMESE_AUDIT_LABELS.actions as Record<string, string>)[a] ??
                a}
            </Badge>
          ))}
          {(filters.entityType ?? []).map((et) => (
            <Badge key={et} variant="secondary" className="text-[11px]">
              {(VIETNAMESE_AUDIT_LABELS.entities as Record<string, string>)[et] ??
                et}
            </Badge>
          ))}
          <button
            onClick={reset}
            className="inline-flex min-h-11 items-center gap-1 px-1 text-[11px] font-semibold text-destructive active:opacity-70"
          >
            <X className="h-3 w-3" />
            Xóa lọc
          </button>
        </div>
      )}

      <div className="flex flex-col gap-3 p-4">
        {isLoading ? (
          <>
            {Array.from({ length: 6 }).map((_, i) => (
              <AuditLogCardSkeleton key={i} />
            ))}
          </>
        ) : logs.length === 0 ? (
          <EmptyState
            icon={ClipboardList}
            title="Không có bản ghi nào"
            description={
              activeCount > 0
                ? 'Không có bản ghi khớp với bộ lọc hiện tại'
                : 'Chưa có nhật ký hoạt động'
            }
          />
        ) : (
          <>
            {logs.map((log) => (
              <AuditLogCard key={log.id} log={log} onClick={setSelectedLogId} />
            ))}
            {isFetchingNextPage && (
              <>
                {Array.from({ length: 3 }).map((_, i) => (
                  <AuditLogCardSkeleton key={i} />
                ))}
              </>
            )}
            <div ref={observerRef} className="h-1" />
          </>
        )}
      </div>

      {/* Filter sheet */}
      <Sheet open={isFilterOpen} onOpenChange={setIsFilterOpen}>
        <SheetContent
          side="bottom"
          className="flex h-auto max-h-[85dvh] flex-col overflow-hidden rounded-t-[28px] border-t border-white/70 bg-slate-50/95 p-0 shadow-[0_-24px_80px_-36px_rgba(15,23,42,0.65)] backdrop-blur-xl"
          title="Bộ lọc nhật ký"
          description="Thu hẹp kết quả theo thời gian, hành động và đối tượng"
        >
          <div className="flex justify-center pt-3">
            <div className="h-1.5 w-12 rounded-full bg-slate-300" />
          </div>
          <SheetHeader className="px-5 pb-3 pt-4 text-left">
            <SheetTitle className="text-xl font-bold tracking-tight text-slate-950">
              Bộ lọc nhật ký
            </SheetTitle>
            <p className="text-sm font-medium text-slate-500">
              Thu hẹp kết quả theo thời gian, hành động và đối tượng
            </p>
          </SheetHeader>

          <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-[calc(env(safe-area-inset-bottom)+1.25rem)]">
            {/* Date range */}
            <div className="mb-5">
              <MobileSectionHeader icon={ClipboardList} title="Khoảng thời gian" />
              <div className="grid grid-cols-1 gap-3 min-[380px]:grid-cols-2">
                <div className="min-w-0">
                  <label className="mb-1 block text-xs font-medium text-slate-500">
                    Từ ngày
                  </label>
                  <Input
                    type="date"
                    value={filters.fromDate ?? ''}
                    onChange={(e) =>
                      setFilters((prev) => ({
                        ...prev,
                        page: 1,
                        fromDate: e.target.value || undefined,
                      }))
                    }
                    className="h-11 w-full text-sm"
                  />
                </div>
                <div className="min-w-0">
                  <label className="mb-1 block text-xs font-medium text-slate-500">
                    Đến ngày
                  </label>
                  <Input
                    type="date"
                    value={filters.toDate ?? ''}
                    onChange={(e) =>
                      setFilters((prev) => ({
                        ...prev,
                        page: 1,
                        toDate: e.target.value || undefined,
                      }))
                    }
                    className="h-11 w-full text-sm"
                  />
                </div>
              </div>
            </div>

            {/* Action multi-select */}
            <div className="mb-5">
              <MobileSectionHeader icon={ClipboardList} title="Hành động" />
              <FilterChipGroup
                options={ALL_ACTIONS}
                selected={filters.action ?? []}
                labels={VIETNAMESE_AUDIT_LABELS.actions as Record<string, string>}
                onToggle={toggleAction}
              />
            </div>

            {/* Entity type multi-select */}
            <div className="mb-5">
              <MobileSectionHeader icon={ClipboardList} title="Đối tượng" />
              <FilterChipGroup
                options={ALL_ENTITY_TYPES}
                selected={filters.entityType ?? []}
                labels={
                  VIETNAMESE_AUDIT_LABELS.entities as Record<string, string>
                }
                onToggle={toggleEntity}
              />
            </div>

            <div className="grid grid-cols-1 gap-2 pt-2 min-[380px]:grid-cols-2">
              <Button
                variant="outline"
                className="h-11 w-full border-slate-200"
                onClick={reset}
                disabled={activeCount === 0}
              >
                Xóa bộ lọc
              </Button>
              <Button
                className="btn-admin-primary h-11 w-full"
                onClick={() => setIsFilterOpen(false)}
              >
                Xem {totalRecords > 0 ? `${totalRecords.toLocaleString('vi-VN')} kết quả` : 'kết quả'}
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>

      <AuditLogDetailSheet
        logId={selectedLogId}
        onClose={() => setSelectedLogId(null)}
      />
    </MobilePageShell>
  );
}
