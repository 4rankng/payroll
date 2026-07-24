import { useState, useEffect, useMemo, memo } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
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
} from 'lucide-react';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { usePartnerImportHistory } from '@/hooks/timesheet/usePartnerImportHistory';
import { timesheetService } from '@/services/api/timesheet.service';
import { parseImportErrors } from '@/utils/import-errors';
import type { PartnerImportFile } from '@/types/api/timesheet.types';

interface UploadHistorySheetProps {
  open: boolean;
  onClose: () => void;
  projectId?: number;
  projects?: { id: number; name: string }[];
}

function statusBadge(status: PartnerImportFile['status']) {
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
  return map[status] ?? <Badge variant="outline">{status}</Badge>;
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
              <span>{e.employee ? <strong>{e.employee}:</strong> : ''} {e.reason}</span>
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

  const projectMap = useMemo(() => {
    if (!projects) return new Map<number, string>();
    return new Map(projects.map(p => [p.id, p.name]));
  }, [projects]);
  const pageSize = 10;

  useEffect(() => {
    if (open) setPage(1);
  }, [open, projectId]);

  const { data, isLoading } = usePartnerImportHistory({
    params: open ? { project_id: projectId, page, page_size: pageSize } : undefined,
    enabled: open,
  });

  const items = data?.data ?? [];
  const pagination = data?.pagination;
  const totalPages = pagination?.totalPages ?? 1;

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
        {isLoading && (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24 w-full rounded-xl" />
            ))}
          </div>
        )}

        {!isLoading && items.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="rounded-full bg-muted p-5 mb-4">
              <UploadCloud className="h-8 w-8 text-muted-foreground" />
            </div>
            <p className="text-sm font-medium text-foreground">Chưa có file nào được tải lên</p>
            <p className="text-xs text-muted-foreground mt-1.5 max-w-xs">
              Sử dụng nút "Tải lên BCC" để nhập dữ liệu bảng công từ file Excel
            </p>
          </div>
        )}

        {!isLoading && items.map((item) => (
          <div
            key={item.id}
            className="rounded-xl border bg-card p-4 space-y-3 hover:bg-muted/30 transition-colors"
          >
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
                {statusBadge(item.status)}
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-muted-foreground hover:text-foreground"
                  onClick={() => handleDownload(item.id, item.original_name)}
                  title="Tải về file gốc"
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
        ))}
      </div>
    </SlideSheetTemplate>
  );
});
