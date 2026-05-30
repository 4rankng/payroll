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
    <div className={cn('flex items-center justify-between gap-3 py-2', className)}>
      {/* Left: record range */}
      <p className="text-[11px] text-muted-foreground tabular-nums shrink-0 min-w-[60px]">
        <span className="font-medium text-foreground/70">{from}–{to}</span>
        <span className="mx-1 text-muted-foreground/50">/</span>
        {totalRecords.toLocaleString('vi-VN')}
      </p>

      {/* Center: page buttons */}
      <div className="flex items-center gap-0.5">
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60"
          onClick={() => onPageChange(1)}
          disabled={page <= 1}
          aria-label="Trang đầu"
        >
          <ChevronsLeft className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          aria-label="Trang trước"
        >
          <ChevronLeft className="h-3.5 w-3.5" />
        </Button>

        {getPages().map((p, i) =>
          p === '…' ? (
            <span
              key={`e${i}`}
              className="h-7 w-7 flex items-center justify-center text-[11px] text-muted-foreground/50 select-none"
            >
              …
            </span>
          ) : (
            <button
              key={p}
              onClick={() => onPageChange(p as number)}
              disabled={p === page}
              aria-label={`Trang ${p}`}
              aria-current={p === page ? 'page' : undefined}
              className={cn(
                'h-7 w-7 rounded-md text-[11px] font-medium transition-colors',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                p === page
                  ? 'border border-border/80 bg-card text-foreground shadow-[0_1px_2px_0_rgb(15_23_42/0.06)] cursor-default'
                  : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
              )}
            >
              {p}
            </button>
          )
        )}

        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          aria-label="Trang tiếp"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground/60 hover:text-foreground hover:bg-muted/60"
          onClick={() => onPageChange(totalPages)}
          disabled={page >= totalPages}
          aria-label="Trang cuối"
        >
          <ChevronsRight className="h-3.5 w-3.5" />
        </Button>
      </div>

      {/* Right: page size selector */}
      <div className="flex items-center gap-1.5 shrink-0">
        <span className="text-[11px] text-muted-foreground/70 hidden sm:inline">Hiển thị</span>
        <Select value={String(pageSize)} onValueChange={(v) => onPageSizeChange(Number(v))}>
          <SelectTrigger className="h-7 w-14 text-[11px] px-2 min-h-0">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {[10, 20, 50, 100].map(n => (
              <SelectItem key={n} value={String(n)} className="text-[11px] py-1">{n}</SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
