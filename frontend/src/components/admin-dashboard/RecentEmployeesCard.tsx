import { useCallback, useMemo, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/ui/user-avatar';
import { ConsolePanel } from '@/components/shared/console-panel';
import type { NewEmployee } from '@/types/api/dashboard.types';
import { formatVietnameseRelativeTime } from '@/utils/vietnamese';

interface RecentEmployeesCardProps {
  employees: NewEmployee[];
  isLoading?: boolean;
  isLoadingMore?: boolean;
  onEmployeeClick: (employee: NewEmployee) => void;
  onLoadMore?: () => void;
  totalEmployees?: number;
  weeks?: number;
  className?: string;
}

/** Shimmer skeleton row for one employee entry. */
const EmployeeSkeleton = () => (
  <div className="flex items-center gap-2.5 px-3 py-2">
    <div className="h-8 w-8 shrink-0 animate-shimmer rounded-full bg-muted bg-[length:200%_100%] [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
    <div className="min-w-0 flex-1 space-y-1.5">
      <div className="h-3 w-24 animate-shimmer rounded bg-muted bg-[length:200%_100%] [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
      <div className="h-2.5 w-16 animate-shimmer rounded bg-muted bg-[length:200%_100%] [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
    </div>
  </div>
);

export function RecentEmployeesCard({
  employees,
  isLoading = false,
  isLoadingMore = false,
  onEmployeeClick,
  onLoadMore,
  totalEmployees = 0,
  weeks = 1,
  className,
}: RecentEmployeesCardProps) {
  const [showAll, setShowAll] = useState(false);

  const employeesArray = useMemo<NewEmployee[]>(() => {
    if (!employees || !Array.isArray(employees)) return [];
    return employees;
  }, [employees]);

  const visibleEmployees = useMemo(
    () => (showAll ? employeesArray : employeesArray.slice(0, 5)),
    [employeesArray, showAll],
  );

  const hasMore = employeesArray.length > 5;
  const weekLabel = weeks === 1 ? '1 tuần gần đây' : `${weeks} tuần gần đây`;

  const handleEmployeeClick = useCallback(
    (employee: NewEmployee) => onEmployeeClick(employee),
    [onEmployeeClick],
  );

  return (
    <ConsolePanel
      kicker="Nhân sự"
      title="Nhân viên mới"
      description={`${weekLabel} · ${totalEmployees.toLocaleString('vi-VN')} tổng`}
      isFlush
      className={className}
    >
      {isLoading && (
        <div className="flex items-center justify-center py-6">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" aria-hidden="true" />
        </div>
      )}

      {!isLoading && employeesArray.length === 0 && (
        <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">
          Không có nhân viên mới trong {weekLabel}
        </p>
      )}

      {!isLoading && employeesArray.length > 0 && (
        <>
          <div className="divide-y divide-border/60 px-3">
            {visibleEmployees.map((employee) => (
              <button
                key={employee.id}
                type="button"
                onClick={() => handleEmployeeClick(employee)}
                className="group flex w-full items-center gap-2.5 py-2 text-left transition-colors duration-100 ease-linear hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <UserAvatar
                  name={employee.fullname}
                  email={employee.email}
                  size="sm"
                  className="shrink-0"
                />
                <div className="min-w-0">
                  <p className="truncate text-[13px] font-medium leading-tight text-foreground">
                    {employee.fullname}
                  </p>
                  <p className="mt-0.5 truncate text-[11px] text-muted-foreground">
                    {formatVietnameseRelativeTime(employee.created_at)} · {employee.created_by_name}
                  </p>
                </div>
              </button>
            ))}

            {isLoadingMore &&
              Array.from({ length: 4 }).map((_, i) => <EmployeeSkeleton key={`skeleton-${i}`} />)}
          </div>

          <div className="mt-1 flex flex-wrap items-center justify-between gap-2 border-t border-border/70 px-3 py-2">
            <p className="console-value text-[11px] text-muted-foreground">
              {employeesArray.length} nhân viên
            </p>
            <div className="flex flex-wrap items-center gap-2">
              {hasMore && (
                <button
                  type="button"
                  onClick={() => setShowAll((prev) => !prev)}
                  className="min-h-11 rounded-lg text-[11px] font-semibold text-primary active:opacity-70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 sm:min-h-9"
                >
                  {showAll ? 'Thu gọn' : `Xem tất cả (${employeesArray.length})`}
                </button>
              )}
              {onLoadMore && (
                <Button
                  onClick={onLoadMore}
                  disabled={isLoadingMore}
                  variant="outline"
                  size="sm"
                  className="text-[11px]"
                >
                  {isLoadingMore ? (
                    <>
                      <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                      Đang tải...
                    </>
                  ) : (
                    `Xem ${weeks + 1} tuần gần đây`
                  )}
                </Button>
              )}
            </div>
          </div>
        </>
      )}
    </ConsolePanel>
  );
}
