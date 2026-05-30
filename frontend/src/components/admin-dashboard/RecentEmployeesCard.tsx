import React, { useCallback, useMemo, useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Loader2 } from 'lucide-react';
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
}

/** Shimmer skeleton card for a single employee entry */
const EmployeeSkeleton = () => (
  <div className="flex items-center gap-2.5 rounded-xl px-2.5 py-2">
    <div className="w-8 h-8 rounded-full flex-shrink-0 bg-muted animate-shimmer bg-[length:200%_100%]
      [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
    <div className="min-w-0 flex-1 space-y-1.5">
      <div className="h-3 w-24 rounded bg-muted animate-shimmer bg-[length:200%_100%]
        [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
      <div className="h-2.5 w-16 rounded bg-muted animate-shimmer bg-[length:200%_100%]
        [background-image:linear-gradient(90deg,hsl(var(--muted))_25%,hsl(var(--muted-foreground)/0.08)_50%,hsl(var(--muted))_75%)]" />
    </div>
  </div>
);

export const RecentEmployeesCard: React.FC<RecentEmployeesCardProps> = ({
  employees,
  isLoading = false,
  isLoadingMore = false,
  onEmployeeClick,
  onLoadMore,
  totalEmployees = 0,
  weeks = 1,
}) => {
  const [showAll, setShowAll] = useState(false);

  const employeesArray = useMemo<NewEmployee[]>(() => {
    if (!employees || !Array.isArray(employees)) return [];
    return employees;
  }, [employees]);

  const visibleEmployees = useMemo(
    () => showAll ? employeesArray : employeesArray.slice(0, 5),
    [employeesArray, showAll]
  );

  const hasMore = employeesArray.length > 5;

  const handleEmployeeClick = useCallback(
    (employee: NewEmployee) => onEmployeeClick(employee),
    [onEmployeeClick]
  );

  const weekLabel = weeks === 1 ? '1 tuần gần đây' : `${weeks} tuần gần đây`;

  return (
    <Card className="hover:shadow-elevated transition-all">
      <CardContent className="p-3 sm:p-4">
        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {!isLoading && employeesArray.length === 0 && (
          <p className="text-center text-muted-foreground py-8 text-sm">
            Không có nhân viên mới trong {weekLabel}
          </p>
        )}

        {!isLoading && employeesArray.length > 0 && (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-1">
              {visibleEmployees.map((employee) => (
                <button
                  key={employee.id}
                  type="button"
                  onClick={() => handleEmployeeClick(employee)}
                  className="group flex items-center gap-2.5 rounded-xl px-2.5 py-2 text-left
                    transition-all duration-150 ease-out
                    hover:bg-accent/50 hover:scale-[1.02] hover:shadow-sm
                    focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <UserAvatar
                    name={employee.fullname}
                    email={employee.email}
                    size="sm"
                    className="flex-shrink-0 ring-2 ring-border/60 transition-transform duration-150 group-hover:scale-105"
                  />
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-foreground truncate leading-tight">
                      {employee.fullname}
                    </p>
                    <p className="text-xs text-muted-foreground/70 mt-0.5 truncate">
                      {formatVietnameseRelativeTime(employee.created_at)}
                    </p>
                    <p className="text-xs text-muted-foreground/50 truncate">
                      {employee.created_by_name}
                    </p>
                  </div>
                </button>
              ))}

              {/* Skeleton placeholders while loading more */}
              {isLoadingMore && Array.from({ length: 4 }).map((_, i) => (
                <EmployeeSkeleton key={`skeleton-${i}`} />
              ))}
            </div>

            <div className="mt-4 flex items-center justify-between border-t pt-3">
              <p className="text-xs text-muted-foreground">
                {employeesArray.length} nhân viên
              </p>
              <div className="flex items-center gap-2">
                {hasMore && (
                  <button
                    onClick={() => setShowAll(prev => !prev)}
                    className="text-primary text-xs font-semibold active:opacity-70"
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
                    className="text-xs transition-all duration-150 hover:scale-[1.02]"
                  >
                    {isLoadingMore ? (
                      <>
                        <Loader2 className="h-3.5 w-3.5 animate-spin mr-1.5" />
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
      </CardContent>
    </Card>
  );
};
