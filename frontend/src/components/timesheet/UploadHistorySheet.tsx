import { useState, useEffect, useMemo, memo } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { SearchableSelect } from '@/components/ui/searchable-select';
import {
  Download,
  ChevronLeft,
  ChevronRight,
  ChevronRight as Chevron,
  FileSpreadsheet,
  History,
  Search,
  X,
} from 'lucide-react';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { usePartnerImportHistory } from '@/hooks/timesheet/usePartnerImportHistory';
import { useDebounce } from '@/hooks/useDebounce';
import { timesheetService } from '@/services/api/timesheet.service';
import { parseImportErrors, groupImportErrors, describeGroupedError } from '@/utils/import-errors';
import { EmptyState } from '@/components/shared/EmptyState';
import { cn } from '@/lib/utils';
import type { PartnerImportFile } from '@/types/api/timesheet.types';

interface UploadHistorySheetProps {
  open: boolean;
  onClose: () => void;
  projectId?: number;
  projects?: { id: number; name: string }[];
}

// Process state the row counts cannot express (Đang chờ / Đang xử lý) and the
// partial-success distinction stay visible as a quiet dot + label — the pill
// chrome it replaces cost ~40px of width per row.
function StatusFlag({ item }: { item: PartnerImportFile }) {
  const flag = (() => {
    if (item.status === 'completed' && item.error_count > 0) {
      return { dot: 'bg-amber-500', text: 'text-amber-700', label: 'Hoàn tất một phần' };
    }
    const map = {
      completed: { dot: 'bg-emerald-500', text: 'text-emerald-700', label: 'Thành công' },
      pending: { dot: 'bg-muted-foreground/40', text: 'text-muted-foreground', label: 'Đang chờ' },
      processing: {
        dot: 'bg-blue-500 animate-pulse motion-reduce:animate-none',
        text: 'text-blue-700',
        label: 'Đang xử lý',
      },
      failed: { dot: 'bg-red-500', text: 'text-red-700', label: 'Thất bại' },
    } as const;
    return map[item.status] ?? { dot: 'bg-muted-foreground/40', text: 'text-muted-foreground', label: item.status };
  })();
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5">
      <span aria-hidden="true" className={cn('h-1.5 w-1.5 rounded-full', flag.dot)} />
      <span className={cn('text-[11px] font-medium', flag.text)}>{flag.label}</span>
    </span>
  );
}

// Density: counts live in one right-aligned muted string on the meta line —
// only non-zero segments render, numbers carry the color, fixed order
// tạo mới → bỏ qua → lỗi. Terminal all-zero success says "0 thay đổi" so an
// empty string never silently reads as "no data yet".
function ResultCounts({ item, errors, groups, onToggleErrors, errorsOpen }: {
  item: PartnerImportFile;
  errors: ReturnType<typeof parseImportErrors>;
  groups: ReturnType<typeof groupImportErrors>;
  onToggleErrors: () => void;
  errorsOpen: boolean;
}) {
  if (item.status !== 'completed' && item.status !== 'failed') return null;

  const segments: React.ReactNode[] = [];
  if (item.created_count > 0) {
    segments.push(
      <span key="created">
        <span className="font-medium tabular-nums text-emerald-600">{item.created_count}</span> tạo mới
      </span>,
    );
  }
  if (item.skipped_count > 0) {
    segments.push(
      <span key="skipped">
        <span className="font-medium tabular-nums text-amber-600">{item.skipped_count}</span> bỏ qua
      </span>,
    );
  }
  if (errors.length > 0) {
    const suffix = groups.length < errors.length ? ` (${groups.length} nhóm)` : '';
    segments.push(
      <button
        key="errors"
        type="button"
        onClick={onToggleErrors}
        aria-expanded={errorsOpen}
        aria-label={errorsOpen ? 'Ẩn lỗi' : `Xem ${errors.length} lỗi${suffix}`}
        className="inline-flex items-center gap-0.5 text-red-600 transition-colors hover:underline underline-offset-2"
      >
        <span className="font-medium tabular-nums">{item.error_count}</span> lỗi
        <Chevron className={cn('h-3 w-3 transition-transform duration-150', errorsOpen && 'rotate-90')} aria-hidden="true" />
      </button>,
    );
  } else if (item.error_count > 0) {
    segments.push(
      <span key="errors">
        <span className="font-medium tabular-nums text-red-600">{item.error_count}</span> lỗi
      </span>,
    );
  }

  return (
    <div className="shrink-0 text-xs text-muted-foreground" data-testid="import-result-counts">
      {segments.length > 0 ? (
        segments.map((segment, index) => (
          <span key={index === 0 ? 0 : index} className="inline-flex items-center">
            {index > 0 && <span className="px-1 text-muted-foreground/50" aria-hidden="true">·</span>}
            {segment}
          </span>
        ))
      ) : (
        <span className="tabular-nums">0 thay đổi</span>
      )}
    </div>
  );
}

function ImportRow({ item, projectMap, onDownload, showProject }: {
  item: PartnerImportFile;
  projectMap: Map<number, string>;
  onDownload: (id: number, name: string) => void;
  showProject: boolean;
}) {
  const [errorsOpen, setErrorsOpen] = useState(false);
  const errors = useMemo(() => parseImportErrors(item.error_detail), [item.error_detail]);
  const groups = useMemo(() => groupImportErrors(errors), [errors]);

  return (
    <div className="px-3 py-2 transition-colors hover:bg-muted/30">
      <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3">
        <p className="flex min-w-0 items-center gap-1.5 text-[13px] font-medium leading-5">
          <FileSpreadsheet className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span className="truncate" title={item.original_name}>{item.original_name}</span>
        </p>
        <div className="flex shrink-0 items-center gap-2">
          <StatusFlag item={item} />
          <Button
            variant="ghost"
            size="icon"
            className="h-11 w-11 sm:h-7 sm:w-7 text-muted-foreground hover:text-foreground"
            onClick={() => onDownload(item.id, item.original_name)}
            title="Tải về tệp gốc"
          >
            <Download className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
      <div className="mt-0.5 grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3">
        <p className="truncate text-xs text-muted-foreground">
          {new Date(item.created_at).toLocaleDateString('vi-VN')}
          <span className="px-1 text-muted-foreground/50" aria-hidden="true">·</span>
          Tháng {item.for_month}
          {showProject && item.project_id != null && (
            <>
              <span className="px-1 text-muted-foreground/50" aria-hidden="true">·</span>
              <span className="font-medium text-primary">
                {projectMap.get(item.project_id) ?? `Dự án #${item.project_id}`}
              </span>
            </>
          )}
        </p>
        <ResultCounts
          item={item}
          errors={errors}
          groups={groups}
          errorsOpen={errorsOpen}
          onToggleErrors={() => setErrorsOpen((v) => !v)}
        />
      </div>
      {errorsOpen && errors.length > 0 && (
        <ul className="mt-2 max-h-28 overflow-y-auto rounded-md bg-destructive/5 border border-destructive/15 p-2.5 text-xs text-destructive space-y-1">
          {groups.map((group, i) => (
            <li key={i} className="flex gap-1.5">
              <span className="shrink-0 mt-px">•</span>
              <span>
                {group.employee ? <strong>{group.employee}: </strong> : ''}
                {group.reason}
                {describeGroupedError(group) && (
                  <span className="text-destructive">{describeGroupedError(group)}</span>
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export const UploadHistorySheet = memo(function UploadHistorySheet({
  open,
  onClose,
  projectId,
  projects,
}: UploadHistorySheetProps) {
  const [page, setPage] = useState(1);
  const [searchInput, setSearchInput] = useState('');
  const debouncedSearch = useDebounce(searchInput, 350);
  // 'all' or a project id as a string (dropdown values are strings).
  const [projectFilter, setProjectFilter] = useState('all');
  const showProjectFilter = !projectId && !!projects && projects.length > 0;
  const effectiveProjectId = projectFilter !== 'all' ? Number(projectFilter) : projectId;
  // Backend rejects searches over 100 runes; the sheet has no error state,
  // so clamp here and never let the request leave the UI too long.
  const clampedSearch = debouncedSearch.trim().slice(0, 100);
  const isFiltered = clampedSearch !== '' || projectFilter !== 'all';

  const projectMap = useMemo(() => {
    if (!projects) return new Map<number, string>();
    return new Map(projects.map(p => [p.id, p.name]));
  }, [projects]);
  const pageSize = 20;

  useEffect(() => {
    setPage(1);
  }, [open, projectId, clampedSearch, projectFilter]);

  // The component stays mounted at every call site; stale filters would hide
  // fresh uploads and invite duplicate re-uploads. Reset when the sheet closes.
  useEffect(() => {
    if (!open) {
      setSearchInput('');
      setProjectFilter('all');
    }
  }, [open]);

  const { data, isLoading } = usePartnerImportHistory({
    params: open ? {
      project_id: effectiveProjectId,
      search: clampedSearch || undefined,
      sort: effectiveProjectId ? undefined : 'project',
      page,
      page_size: pageSize,
    } : undefined,
    enabled: open,
  });

  const items = useMemo(() => data?.data ?? [], [data]);
  const pagination = data?.pagination;
  const totalPages = pagination?.totalPages ?? 1;

  // Group items by project preserving server order (sort=project keeps each
  // project's uploads newest-first and contiguous across pages).
  const groups = useMemo(() => {
    const map = new Map<number, PartnerImportFile[]>();
    for (const item of items) {
      const bucket = map.get(item.project_id);
      if (bucket) bucket.push(item);
      else map.set(item.project_id, [item]);
    }
    return Array.from(map.entries());
  }, [items]);
  // Headers must persist even on continuation pages holding a single project,
  // otherwise the user loses track of what they are scrolling.
  const isGrouped = effectiveProjectId == null && groups.length >= 1;

  const handleDownload = async (id: number, name: string) => {
    await timesheetService.downloadPartnerImport(id, name);
  };

  const renderRow = (item: PartnerImportFile) => (
    <ImportRow
      key={item.id}
      item={item}
      projectMap={projectMap}
      onDownload={handleDownload}
      // The project group header already names the project; repeating it on
      // every row only holds in the filtered view where headers are absent.
      showProject={!isGrouped}
    />
  );

  const paginationFooter = totalPages > 1 ? (
    <div className="flex items-center justify-between">
      <span className="text-xs text-muted-foreground">
        Trang {page} / {totalPages}
      </span>
      <div className="flex items-center gap-1.5">
        <Button
          variant="outline"
          size="icon"
          className="h-11 w-11 sm:h-7 sm:w-7"
          disabled={page <= 1}
          onClick={() => setPage((p) => p - 1)}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          className="h-11 w-11 sm:h-7 sm:w-7"
          disabled={page >= totalPages}
          onClick={() => setPage((p) => p + 1)}
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  ) : undefined;

  return (
    <SlideSheetTemplate
      isOpen={open}
      onClose={onClose}
      title="Lịch sử tải lên BCC"
      avatar={{ icon: History }}
      size="default"
      footer={paginationFooter}
    >
      <div className="space-y-3">
        <div className="flex flex-wrap gap-2">
          <div className="relative min-w-0 flex-1 basis-48">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
            <Input
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              placeholder="Tìm theo tên tệp…"
              aria-label="Tìm theo tên tệp"
              className="h-11 sm:h-11 pl-9 pr-9 sm:pl-9 sm:pr-9"
            />
            {searchInput && (
              <button
                type="button"
                onClick={() => setSearchInput('')}
                aria-label="Xóa từ khóa"
                className="absolute right-1.5 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          {showProjectFilter && (
            <SearchableSelect
              value={projectFilter}
              onChange={setProjectFilter}
              placeholder="Tất cả dự án"
              searchPlaceholder="Tìm dự án..."
              triggerClassName="w-full sm:w-[200px]"
              options={[
                { value: 'all', label: 'Tất cả dự án' },
                ...(projects?.map((p) => ({ value: String(p.id), label: p.name })) ?? []),
              ]}
            />
          )}
        </div>

        {isLoading && (
          <div className="overflow-hidden rounded-lg border border-border bg-card">
            <div className="divide-y divide-border/60">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full rounded-none" />
              ))}
            </div>
          </div>
        )}

        {!isLoading && items.length === 0 && (isFiltered ? (
          <EmptyState
            title="Không tìm thấy tệp phù hợp"
            description="Thử từ khóa khác hoặc xóa bộ lọc."
            size="sm"
          />
        ) : (
          <EmptyState
            title="Chưa có tệp nào được tải lên"
            description="Sử dụng nút Tải lên BCC để nhập dữ liệu bảng công từ tệp Excel"
            size="sm"
          />
        ))}

        {!isLoading && isGrouped && groups.map(([groupId, groupItems]) => (
          <section key={groupId}>
            {/* Sticky scan anchor: with 20 rows per page the header must
                survive scrolling inside the sheet. */}
            <h3 className="sticky top-0 z-10 -mx-4 bg-card/95 px-4 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground backdrop-blur sm:-mx-6 sm:px-6">
              {projectMap.get(groupId) ?? `Dự án #${groupId}`}
            </h3>
            <div className="mt-1 overflow-hidden rounded-lg border border-border bg-card">
              <div className="divide-y divide-border/60">
                {groupItems.map(renderRow)}
              </div>
            </div>
          </section>
        ))}

        {!isLoading && !isGrouped && items.length > 0 && (
          <div className="overflow-hidden rounded-lg border border-border bg-card">
            <div className="divide-y divide-border/60">
              {items.map(renderRow)}
            </div>
          </div>
        )}
      </div>
    </SlideSheetTemplate>
  );
});
