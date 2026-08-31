import { useState, useEffect, useMemo, memo } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Download,
  ChevronLeft,
  ChevronRight,
  FileSpreadsheet,
  UploadCloud,
  CheckCircle2,
  AlertCircle,
  Loader2,
  MinusCircle,
  Calendar,
  History,
  Search,
  X,
} from 'lucide-react';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { usePartnerImportHistory } from '@/hooks/timesheet/usePartnerImportHistory';
import { useDebounce } from '@/hooks/useDebounce';
import { timesheetService } from '@/services/api/timesheet.service';
import { parseImportErrors } from '@/utils/import-errors';
import { EmptyState } from '@/components/shared/EmptyState';
import type { PartnerImportFile } from '@/types/api/timesheet.types';

interface UploadHistorySheetProps {
  open: boolean;
  onClose: () => void;
  projectId?: number;
  projects?: { id: number; name: string }[];
}

function statusBadge(item: PartnerImportFile) {
  if (item.status === 'completed' && item.error_count > 0) {
    return (
      <Badge className="border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-50">
        Hoàn tất một phần
      </Badge>
    );
  }
  const map = {
    completed: (
      <Badge className="bg-green-100 text-green-800 hover:bg-green-100 border-0">
        Thành công
      </Badge>
    ),
    pending: (
      <Badge className="border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-50">
        Đang chờ
      </Badge>
    ),
    processing: (
      <Badge className="border-emerald-200 bg-emerald-50 text-emerald-800 hover:bg-emerald-50">
        <Loader2 className="mr-1 h-3 w-3 animate-spin motion-reduce:animate-none" aria-hidden="true" />
        Đang xử lý
      </Badge>
    ),
    failed: <Badge variant="destructive">Thất bại</Badge>,
  } as const;
  return map[item.status] ?? <Badge variant="outline">{item.status}</Badge>;
}

function ImportCard({ item, projectMap, onDownload }: {
  item: PartnerImportFile;
  projectMap: Map<number, string>;
  onDownload: (id: number, name: string) => void;
}) {
  return (
    <div className="rounded-xl border bg-card p-4 space-y-3 hover:bg-muted/30 transition-colors">
      <div className="flex items-start gap-3">
        <div className="shrink-0 rounded-lg bg-blue-50 p-2 mt-0.5">
          <FileSpreadsheet className="h-4 w-4 text-blue-600" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium leading-snug truncate" title={item.original_name}>
            {item.original_name}
          </p>
          <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5 mt-1 text-xs text-muted-foreground">
            <span className="flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Tháng {item.for_month}
            </span>
            {projectMap.size > 0 && item.project_id != null && (
              <>
                <span>·</span>
                <span className="text-primary/80 font-medium">
                  {projectMap.get(item.project_id) ?? `Dự án #${item.project_id}`}
                </span>
              </>
            )}
            <span>·</span>
            <span>{new Date(item.created_at).toLocaleDateString('vi-VN')}</span>
          </div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {statusBadge(item)}
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-foreground"
            onClick={() => onDownload(item.id, item.original_name)}
            title="Tải về tệp gốc"
          >
            <Download className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-2 flex-wrap">
        <span className="inline-flex items-center gap-1 rounded-full bg-green-50 px-2.5 py-1 text-xs font-medium text-green-700 border border-green-100">
          <CheckCircle2 className="h-3 w-3" />
          {item.created_count} tạo mới
        </span>
        <span className="inline-flex items-center gap-1 rounded-full bg-yellow-50 px-2.5 py-1 text-xs font-medium text-yellow-700 border border-yellow-100">
          <MinusCircle className="h-3 w-3" />
          {item.skipped_count} bỏ qua
        </span>
        {item.error_count > 0 && (
          <span className="inline-flex items-center gap-1 rounded-full bg-red-50 px-2.5 py-1 text-xs font-medium text-red-600 border border-red-100">
            <AlertCircle className="h-3 w-3" />
            {item.error_count} lỗi
          </span>
        )}
      </div>

      <ErrorDetail detail={item.error_detail} />
    </div>
  );
}

function ErrorDetail({ detail }: { detail?: string | null }) {
  const [open, setOpen] = useState(false);
  const errors = parseImportErrors(detail);
  if (errors.length === 0) return null;
  return (
    <div>
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1 text-xs text-destructive/70 hover:text-destructive transition-colors"
      >
        <ChevronRight className={`h-3 w-3 transition-transform duration-150 ${open ? 'rotate-90' : ''}`} />
        {open ? 'Ẩn lỗi' : `Xem ${errors.length} lỗi`}
      </button>
      {open && (
        <ul className="mt-2 max-h-28 overflow-y-auto rounded-md bg-destructive/5 border border-destructive/15 p-2.5 text-xs text-destructive space-y-1">
          {errors.map((e, i) => (
            <li key={i} className="flex gap-1.5">
              <span className="shrink-0 mt-px">•</span>
              <span>
                {e.row > 0 && <strong>Dòng {e.row}: </strong>}
                {e.employee ? <strong>{e.employee}: </strong> : ''}
                {e.reason}
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
  // 'all' or a project id as a string (shadcn Select values are strings).
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
  const pageSize = 10;

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

  const paginationFooter = totalPages > 1 ? (
    <div className="flex items-center justify-between">
      <span className="text-xs text-muted-foreground">
        Trang {page} / {totalPages}
      </span>
      <div className="flex items-center gap-1.5">
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
          disabled={page <= 1}
          onClick={() => setPage((p) => p - 1)}
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
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
              className="h-11 sm:h-11 pl-9 pr-9"
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
            <Select value={projectFilter} onValueChange={setProjectFilter}>
              <SelectTrigger className="h-11 sm:h-11 w-full sm:w-[200px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Tất cả dự án</SelectItem>
                {projects?.map((p) => (
                  <SelectItem key={p.id} value={String(p.id)}>{p.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>

        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24 w-full rounded-xl" />
            ))}
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
          <section key={groupId} className="space-y-3">
            <h3 className="px-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {projectMap.get(groupId) ?? `Dự án #${groupId}`}
            </h3>
            {groupItems.map((item) => (
              <ImportCard
                key={item.id}
                item={item}
                projectMap={projectMap}
                onDownload={handleDownload}
              />
            ))}
          </section>
        ))}

        {!isLoading && !isGrouped && items.map((item) => (
          <ImportCard
            key={item.id}
            item={item}
            projectMap={projectMap}
            onDownload={handleDownload}
          />
        ))}
      </div>
    </SlideSheetTemplate>
  );
});
