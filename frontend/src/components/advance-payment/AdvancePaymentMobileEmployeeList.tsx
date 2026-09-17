import { memo, useCallback } from "react";
import { ChevronRight, Users } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";
import { cn } from "@/lib/utils";
import { AccentStripCard } from "@/components/shared/AccentStripCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { MobilePagination, type PaginationInfo } from "@/components/shared/MobilePagination";
import { renderAdvanceToggle } from "@/components/advance-payment/table-config";
import type { FlexPayEmployeeListItem } from "@/types/api/advance-payment.types";

const EmployeeCard = memo(function EmployeeCard({
  emp,
  onPress,
}: {
  emp: FlexPayEmployeeListItem;
  onPress: (emp: FlexPayEmployeeListItem) => void;
}) {
  const usedPercentage =
    emp.maxAdvanceAmount > 0
      ? (emp.utilizedAmount / emp.maxAdvanceAmount) * 100
      : 0;

  return (
    <AccentStripCard
      accentColor={emp.availableAmount > 0 ? "green" : "orange"}
    >
      <div className="px-4 pt-3.5 pb-3">
        <button type="button" onClick={() => onPress(emp)} aria-label={`Xem chi tiết ứng lương của ${emp.fullname}`} className="mb-2 flex min-h-11 w-full items-center justify-between gap-2 rounded-md text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          <div className="min-w-0 flex-1">
            <p className="break-words text-sm font-semibold text-foreground">
              {emp.fullname}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">{emp.cccd}</p>
          </div>
          <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden />
        </button>

        <div className="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs">
          <div>
            <span className="text-muted-foreground">Hạn mức: </span>
            <span className="font-semibold text-foreground tabular-nums">
              {formatCurrency(emp.maxAdvanceAmount)}
            </span>
          </div>
          <div>
            <span className="text-muted-foreground">Còn lại: </span>
            <span className="font-semibold text-emerald-700 tabular-nums">
              {formatCurrency(emp.availableAmount)}
            </span>
          </div>
        </div>

        {usedPercentage > 0 && (
          <div className="mt-2">
            <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
              <span>Đã sử dụng</span>
              <span className="tabular-nums">
                {Math.round(usedPercentage)}%
              </span>
            </div>
            <div className="h-1.5 bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  "h-full rounded-full transition-all",
                  usedPercentage > 80 ? "bg-orange-400" : "bg-emerald-400",
                )}
                style={{ width: `${Math.min(100, usedPercentage)}%` }}
              />
            </div>
          </div>
        )}

        {emp.project && (
          <>
            {emp.project.name && (
              <p className="mt-2 break-words text-xs text-muted-foreground">
                {emp.project.name}
              </p>
            )}
            <div className="mt-2.5 pt-2.5 border-t border-border/60 flex items-center justify-between">
              <span className="text-xs text-muted-foreground">Ứng lương</span>
              {renderAdvanceToggle(emp)}
            </div>
          </>
        )}
      </div>
    </AccentStripCard>
  );
});

interface AdvancePaymentMobileEmployeeListProps {
  employees: FlexPayEmployeeListItem[];
  onEmployeePress: (emp: FlexPayEmployeeListItem) => void;
  pagination?: PaginationInfo | null;
  onPageChange?: (page: number) => void;
}

export function AdvancePaymentMobileEmployeeList({
  employees,
  onEmployeePress,
  pagination,
  onPageChange,
}: AdvancePaymentMobileEmployeeListProps) {
  const handlePress = useCallback(
    (emp: FlexPayEmployeeListItem) => { onEmployeePress(emp); },
    [onEmployeePress],
  );

  if (employees.length === 0) {
    return (
      <EmptyState
        icon={Users}
        title="Chưa có dữ liệu"
        description="Vui lòng nhập danh sách nhân viên"
      />
    );
  }

  return (
    <>
      <div className="space-y-2.5">
        {employees.map((emp) => (
          <EmployeeCard
            key={`${emp.employeeId}-${emp.project?.id}`}
            emp={emp}
            onPress={handlePress}
          />
        ))}
      </div>

      {pagination && onPageChange && (
        <MobilePagination pagination={pagination} onPageChange={onPageChange} />
      )}
    </>
  );
}
