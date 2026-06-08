import { memo, useState, useMemo, useCallback } from 'react';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetClose } from '@/components/ui/sheet';
import { useIsMobile } from '@/hooks/use-mobile';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { X, Phone, Hash, Briefcase, Calendar, DollarSign, UserCheck, UserX, Search } from 'lucide-react';
import { usePartnerEmployeeList } from '@/hooks/api/useDashboard';
import { normalizeVietnamese } from '@/utils/vietnamese';
import type { PartnerEmployeeListType, PartnerEmployeeDetailItem } from '@/types/api/dashboard.types';

// ── helpers ───────────────────────────────────────────────────────────────────
function formatVND(value: number): string {
  const abs = Math.abs(value);
  if (abs >= 1e9) return `${(abs / 1e9).toFixed(1)}tỷ đ`;
  if (abs >= 1e6) return `${(abs / 1e6).toFixed(1)}M đ`;
  if (abs >= 1e3) return `${(abs / 1e3).toFixed(0)}K đ`;
  return `${abs.toLocaleString('vi-VN')} đ`;
}

function formatDate(d: string): string {
  if (!d) return '—';
  // Handle both YYYY-MM-DD and DD/MM/YYYY formats
  if (d.includes('-')) {
    const [y, m, day] = d.split('-');
    return `${day}/${m}/${y}`;
  }
  // Already in DD/MM/YYYY format
  return d;
}

const TYPE_CONFIG: Record<PartnerEmployeeListType, {
  title: string;
  emptyText: string;
  badgeLabel: string;
  badgeClass: string;
}> = {
  active: {
    title: 'Nhân viên đang làm việc',
    emptyText: 'Không có nhân viên nào đang làm việc',
    badgeLabel: 'Đang làm',
    badgeClass: 'border-emerald-300 text-emerald-700 bg-emerald-50',
  },
  dropped: {
    title: 'Nhân viên có thể đã nghỉ',
    emptyText: 'Không có nhân viên nào nghỉ việc',
    badgeLabel: 'Nghỉ việc',
    badgeClass: 'border-rose-300 text-rose-600 bg-rose-50',
  },
  paid: {
    title: 'Nhân viên được trả lương',
    emptyText: 'Không có nhân viên nào được trả lương trong kỳ này',
    badgeLabel: 'Đã trả lương',
    badgeClass: 'border-blue-300 text-blue-700 bg-blue-50',
  },
};

// ── Info row ───────────────────────────────────────────────────────────────────
function InfoRow({ icon: Icon, label, children }: {
  icon: React.ElementType;
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center gap-1.5 min-w-0">
      <Icon className="w-3 h-3 text-muted-foreground shrink-0" />
      <span className="text-[10px] text-muted-foreground shrink-0">{label}</span>
      <span className="text-xs text-foreground tabular-nums truncate">{children}</span>
    </div>
  );
}

// ── Employee card ─────────────────────────────────────────────────────────────
const EmployeeCard = memo(function EmployeeCard({
  item,
}: {
  item: PartnerEmployeeDetailItem;
  type: PartnerEmployeeListType;
}) {
  return (
    <div className="rounded-xl border border-border/60 bg-card p-3 space-y-2 hover:bg-muted/20 transition-colors">
      {/* Header: name + status */}
      <div className="flex items-center gap-2">
        <span className="text-sm font-semibold text-foreground truncate">{item.employee_name}</span>
        <Badge
          variant="outline"
          className={`text-[10px] px-1.5 py-0 h-4 shrink-0 ${item.is_active ? TYPE_CONFIG.active.badgeClass : TYPE_CONFIG.dropped.badgeClass}`}
        >
          {item.is_active
            ? <><UserCheck className="w-2.5 h-2.5 mr-0.5" />Đang làm</>
            : <><UserX className="w-2.5 h-2.5 mr-0.5" />Nghỉ</>
          }
        </Badge>
      </div>

      {/* Details grid */}
      <div className="grid grid-cols-2 gap-x-3 gap-y-1">
        {item.project_name && <InfoRow icon={Briefcase} label="Dự án">{item.project_name}</InfoRow>}
        {item.mobile && (
          <a href={`tel:${item.mobile}`} onClick={(e) => e.stopPropagation()} className="contents">
            <InfoRow icon={Phone} label="SĐT">
              <span className="text-primary font-medium group-hover:underline">{item.mobile}</span>
            </InfoRow>
          </a>
        )}
        {item.cccd && <InfoRow icon={Hash} label="CCCD">{item.cccd}</InfoRow>}
        {item.last_paid_date && <InfoRow icon={Calendar} label="Trả gần nhất">{formatDate(item.last_paid_date)}</InfoRow>}
        {item.last_paid_vnd > 0 && <InfoRow icon={DollarSign} label="Tiền gần nhất">{formatVND(item.last_paid_vnd)}</InfoRow>}
        {item.total_paid_vnd > 0 && <InfoRow icon={DollarSign} label="Tổng đã trả">{formatVND(item.total_paid_vnd)}</InfoRow>}
      </div>
    </div>
  );
});

// ── Sheet ─────────────────────────────────────────────────────────────────────
interface PartnerEmployeeListSheetProps {
  type: PartnerEmployeeListType | null;
  month?: string;
  onClose: () => void;
}

export function PartnerEmployeeListSheet({ type, month, onClose }: PartnerEmployeeListSheetProps) {
  const isMobile = useIsMobile();
  const [search, setSearch] = useState('');
  const { data, isLoading } = usePartnerEmployeeList(
    type ? { type, month } : null
  );

  const cfg = type ? TYPE_CONFIG[type] : null;
  const employees = useMemo(() => data?.employees ?? [], [data?.employees]);

  const filteredEmployees = useMemo(() => {
    if (!search.trim()) return employees;
    const q = normalizeVietnamese(search.trim());
    return employees.filter((e) =>
      normalizeVietnamese(e.employee_name).includes(q) ||
      (e.mobile && e.mobile.includes(q)) ||
      (e.cccd && e.cccd.includes(q)) ||
      (e.project_name && normalizeVietnamese(e.project_name).includes(q))
    );
  }, [employees, search]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setSearch(e.target.value);
  }, []);

  const handleOpenChange = useCallback((open: boolean) => {
    if (!open) {
      setSearch('');
      onClose();
    }
  }, [onClose]);

  const periodLabel = month
    ? (() => { const [y, m] = month.split('-'); return `T${m}/${y}`; })()
    : '';

  return (
    <Sheet open={!!type} onOpenChange={handleOpenChange}>
      <SheetContent side={isMobile ? "bottom" : "right"} className="w-full sm:w-[440px] p-0 flex flex-col">
        {/* Header */}
        <SheetHeader className="px-4 py-3 border-b flex-shrink-0" style={{ paddingTop: "max(12px, calc(12px + env(safe-area-inset-top)))" }}>
          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0">
              <SheetTitle className="text-sm font-semibold truncate">
                {cfg?.title ?? ''}
              </SheetTitle>
              <div className="flex items-center gap-2 mt-0.5">
                {!isLoading && (
                  <span className="text-xs text-muted-foreground">
                    {search.trim() ? `${filteredEmployees.length}/${data?.total ?? 0}` : (data?.total ?? 0)} nhân viên
                    {periodLabel && ` · ${periodLabel}`}
                  </span>
                )}
              </div>
            </div>
            <SheetClose asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full text-muted-foreground shrink-0">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>

        {/* Search */}
        {employees.length > 0 && (
          <div className="px-4 py-1 flex-shrink-0">
            <div className="relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
              <Input
                value={search}
                onChange={handleSearchChange}
                placeholder="Tìm tên, SĐT, CCCD, dự án..."
                className="h-8 pl-8 text-xs rounded-lg"
              />
            </div>
          </div>
        )}

        {/* List */}
        <div className="flex-1 overflow-y-auto p-4 space-y-2">
          {isLoading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="rounded-xl border border-border/60 p-3.5 space-y-3">
                <div className="flex items-center gap-3">
                  <Skeleton className="w-9 h-9 rounded-full shrink-0" />
                  <div className="flex-1 space-y-1.5">
                    <Skeleton className="h-3.5 w-32" />
                    <Skeleton className="h-3 w-24" />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <Skeleton className="h-3 w-24" />
                  <Skeleton className="h-3 w-20" />
                </div>
              </div>
            ))
          ) : filteredEmployees.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-40 text-center gap-2">
              <div className="w-12 h-12 rounded-2xl bg-muted/60 flex items-center justify-center">
                {type === 'active'
                  ? <UserCheck className="w-6 h-6 text-muted-foreground/40" />
                  : type === 'dropped'
                    ? <UserX className="w-6 h-6 text-muted-foreground/40" />
                    : <DollarSign className="w-6 h-6 text-muted-foreground/40" />
                }
              </div>
              <p className="text-sm text-muted-foreground">
                {search.trim() ? 'Không tìm thấy nhân viên phù hợp' : cfg?.emptyText}
              </p>
            </div>
          ) : (
            filteredEmployees.map((item) => (
              <EmployeeCard key={item.employee_id} item={item} type={type!} />
            ))
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
