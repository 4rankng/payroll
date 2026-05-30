import { useState, useMemo, useEffect, useCallback } from 'react';
import { FileText, Search, ArrowUpDown, ChevronDown, X } from 'lucide-react';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { FilterChipBar } from '@/components/ui/filter-chip-bar';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { UnifiedFileCard } from './UnifiedFileCard';
import type { AdvancePaymentFileType } from './UnifiedFileCard';
import { useAdvancePaymentFileHistory } from '@/hooks/api/useAdvancePayments';
import { groupFilesByDate } from '@/utils/fileGrouping';
import { useMediaQuery } from '@/hooks/use-media-query';
import type { AdvancePaymentFileHistoryItem } from '@/types/api/advance-payment.types';

interface FileHistorySheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

type FilterValue = 'all' | AdvancePaymentFileType;

const PAGE_SIZE = 20;

export const FileHistorySheet = ({ open, onOpenChange }: FileHistorySheetProps) => {
  const isMobile = useMediaQuery('(max-width: 767px)');
  const { data: fileHistory, isLoading } = useAdvancePaymentFileHistory({ enabled: open });

  const allFiles = fileHistory?.data ?? [];

  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [activeFilter, setActiveFilter] = useState<FilterValue>('all');
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);
  const [sortNewest, setSortNewest] = useState(true);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchQuery), 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Reset visible count when filters change
  useEffect(() => {
    setVisibleCount(PAGE_SIZE);
  }, [activeFilter, debouncedSearch]);

  // Type counts for chip badges
  const typeCounts = useMemo(() => {
    const counts: Record<string, number> = { all: allFiles.length };
    for (const f of allFiles) {
      counts[f.uploadType] = (counts[f.uploadType] ?? 0) + 1;
    }
    return counts;
  }, [allFiles]);

  // Filter + search + sort
  const filteredFiles = useMemo(() => {
    let result = allFiles;
    if (activeFilter !== 'all') {
      result = result.filter(f => f.uploadType === activeFilter);
    }
    if (debouncedSearch.trim()) {
      const q = debouncedSearch.toLowerCase();
      result = result.filter(f =>
        f.filename.toLowerCase().includes(q) ||
        (f.uploadedBy ?? '').toLowerCase().includes(q)
      );
    }
    return [...result].sort((a, b) => {
      const diff = new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
      return sortNewest ? diff : -diff;
    });
  }, [allFiles, activeFilter, debouncedSearch, sortNewest]);

  // Paginate then group by date
  const visibleGroups = useMemo(
    () => groupFilesByDate(filteredFiles.slice(0, visibleCount)),
    [filteredFiles, visibleCount]
  );

  const hasMore = filteredFiles.length > visibleCount;

  const handleClearFilters = useCallback(() => {
    setActiveFilter('all');
    setSearchQuery('');
  }, []);

  const filterChips = useMemo(() => [
    { value: 'all', label: 'Tất cả', count: typeCounts.all },
    { value: 'advance_payment_result', label: 'Kết quả ứng lương', count: typeCounts.advance_payment_result, dotClass: 'bg-emerald-500' },
    { value: 'advance_payment_export', label: 'Xuất chuyển lô', count: typeCounts.advance_payment_export, dotClass: 'bg-amber-500' },
    { value: 'flex_pay_import', label: 'Nhập bảng lương', count: typeCounts.flex_pay_import, dotClass: 'bg-blue-500' },
    { value: 'advance_payment_sao_ke_export', label: 'Xuất sao kê', count: typeCounts.advance_payment_sao_ke_export, dotClass: 'bg-purple-500' },
  ], [typeCounts]);

  const renderContent = () => {
    if (isLoading) {
      return (
        <div className="space-y-3 px-1">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-3 px-3 py-2.5">
              <Skeleton className="w-10 h-10 rounded-[10px] shrink-0" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-3 w-1/2" />
              </div>
            </div>
          ))}
        </div>
      );
    }

    if (allFiles.length === 0) {
      return (
        <div className="text-center py-16 text-muted-foreground">
          <FileText className="mx-auto h-12 w-12 mb-3 opacity-40" />
          <p className="text-sm">Chưa có file nào được tải lên</p>
        </div>
      );
    }

    if (filteredFiles.length === 0) {
      return (
        <div className="text-center py-16 text-muted-foreground">
          <Search className="mx-auto h-10 w-10 mb-3 opacity-40" />
          <p className="text-sm mb-3">Không tìm thấy file nào</p>
          <Button variant="outline" size="sm" onClick={handleClearFilters}>
            Xóa bộ lọc
          </Button>
        </div>
      );
    }

    return (
      <div className="space-y-0.5">
        {visibleGroups.map(group => (
          <div key={group.key}>
            <div className="flex items-center gap-3 px-4 py-3">
              <span className="text-[11px] font-bold text-muted-foreground uppercase tracking-wider whitespace-nowrap">
                {group.label}
              </span>
              <div className="flex-1 border-t border-border" />
            </div>
            {group.files.map(file => (
              <UnifiedFileCard
                key={`${file.uploadType}-${file.id}`}
                file={{
                  id: file.id,
                  filename: file.filename,
                  type: file.uploadType as AdvancePaymentFileType,
                  date: file.createdAt,
                  uploadedBy: file.uploadedBy ?? '',
                }}
                alwaysShowActions={isMobile}
              />
            ))}
          </div>
        ))}
      </div>
    );
  };

  return (
    <SlideSheetTemplate
      isOpen={open}
      onClose={() => onOpenChange(false)}
      size="full"
      className="w-full sm:max-w-[540px] md:max-w-[640px]"
      compact
      avatar={{
        custom: (
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-bold tracking-tight">Lịch sử File</h2>
                {allFiles.length > 0 && (
                  <Badge variant="secondary" className="rounded-full px-2 py-0 text-[11px] font-mono">
                    {allFiles.length} file
                  </Badge>
                )}
              </div>
              <p className="text-xs text-muted-foreground mt-0.5 truncate">
                Tất cả file đã tải lên — bảng lương và kết quả chuyển khoản
              </p>
            </div>
          </div>
        ),
      }}
      footer={
        filteredFiles.length > 0 ? (
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">
              Đang hiển thị <strong className="font-mono text-foreground">{Math.min(visibleCount, filteredFiles.length)}</strong>
              {' / '}
              <strong className="font-mono text-foreground">{filteredFiles.length}</strong> file
            </span>
            {hasMore && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setVisibleCount(prev => prev + PAGE_SIZE)}
                className="h-7 text-xs font-semibold"
              >
                Tải thêm
                <ChevronDown className="w-3 h-3 ml-1" />
              </Button>
            )}
          </div>
        ) : undefined
      }
    >
      {/* Toolbar: search + sort */}
      <div className="flex items-center gap-2 mb-1">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Tìm theo tên file hoặc người tải lên…"
            className="h-9 pl-9 pr-8 text-sm"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setSortNewest(prev => !prev)}
          className="h-9 px-2.5 shrink-0"
          title={sortNewest ? 'Mới nhất trước' : 'Cũ nhất trước'}
        >
          <ArrowUpDown className="w-4 h-4" />
        </Button>
      </div>

      {/* Filter chips */}
      {allFiles.length > 0 && (
        <FilterChipBar
          chips={filterChips}
          value={activeFilter}
          onChange={(v) => setActiveFilter(v as FilterValue)}
        />
      )}

      {/* Content */}
      {renderContent()}
    </SlideSheetTemplate>
  );
};
