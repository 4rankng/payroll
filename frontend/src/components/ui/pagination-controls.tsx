import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-react';
import { cn } from '@/lib/utils';

interface PaginationControlsProps {
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
  className?: string;
}

export function PaginationControls({
  pagination,
  onPageChange,
  onPageSizeChange,
  className,
}: PaginationControlsProps) {
  if (!pagination) return null;

  const { page, pageSize, totalPages, totalRecords } = pagination;
  const from = totalRecords === 0 ? 0 : (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, totalRecords);

  const getPages = (): (number | '…')[] => {
    if (totalPages <= 7) return Array.from({ length: totalPages }, (_, i) => i + 1);
    const pages: (number | '…')[] = [];
    const add = (n: number) => { if (!pages.includes(n)) pages.push(n); };

    add(1);
    if (page > 3) pages.push('…');
    for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) add(i);
    if (page < totalPages - 2) pages.push('…');
    add(totalPages);
    return pages;
  };

  return (
    <div
      data-slot="pagination-controls"
      className={cn('admin-pagination flex flex-wrap items-center justify-between gap-3 py-2.5', className)}
    >
      <p className="text-xs text-fg-tertiary tabular-nums shrink-0 min-w-[68px]">
        <span className="font-medium text-fg-secondary">{from}–{to}</span>
        <span className="mx-1 text-fg-tertiary">/</span>
        {totalRecords.toLocaleString('vi-VN')}
      </p>

      <nav aria-label="Phân trang" className="order-3 flex w-full items-center justify-center gap-1 sm:order-none sm:w-auto">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-11 w-11 shrink-0 sm:h-9 sm:w-9"
          onClick={() => onPageChange(1)}
          disabled={page <= 1}
          aria-label="Trang đầu"
        >
          <ChevronsLeft className="h-4 w-4" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-11 w-11 shrink-0 sm:h-9 sm:w-9"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          aria-label="Trang trước"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>

        <span className="min-w-0 px-2 text-xs tabular-nums sm:hidden" aria-live="polite">
          {page} / {Math.max(1, totalPages)}
        </span>
        {getPages().map((p, i) =>
          p === '…' ? (
            <span
              key={`e${i}`}
              className="hidden h-9 w-9 items-center justify-center text-xs text-fg-tertiary select-none sm:flex"
            >
              …
            </span>
          ) : (
            <button
              type="button"
              key={p}
              onClick={() => onPageChange(p as number)}
              disabled={p === page}
              aria-label={`Trang ${p}`}
              aria-current={p === page ? 'page' : undefined}
              className={cn(
                'hidden h-9 w-9 rounded-lg text-xs font-medium transition-colors sm:block',
                // Landed button vocabulary (W4): UU brand outline focus ring.
                'outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2',
                p === page
                  ? 'bg-brand-solid text-white shadow-xs-skeumorphic cursor-default'
                  : 'text-tertiary hover:bg-secondary_hover hover:text-tertiary_hover'
              )}
            >
              {p}
            </button>
          )
        )}

        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-11 w-11 shrink-0 sm:h-9 sm:w-9"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          aria-label="Trang tiếp"
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-11 w-11 shrink-0 sm:h-9 sm:w-9"
          onClick={() => onPageChange(totalPages)}
          disabled={page >= totalPages}
          aria-label="Trang cuối"
        >
          <ChevronsRight className="h-4 w-4" />
        </Button>
      </nav>

      <div className="flex items-center gap-1.5 shrink-0">
        <span className="text-xs text-fg-tertiary hidden sm:inline">Hiển thị</span>
        <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(Number(v))}>
          <SelectTrigger aria-label="Số dòng mỗi trang" className="h-11 w-16 text-xs px-2 sm:h-9">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {[10, 20, 50, 100].map((n) => (
              <SelectItem key={n} value={String(n)} className="text-xs py-2">{n}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
