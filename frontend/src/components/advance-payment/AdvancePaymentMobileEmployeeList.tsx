import { memo, useCallback } from "react";
import { Users } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";
import { cn } from "@/lib/utils";
import { AccentStripCard } from "@/components/shared/AccentStripCard";
import { EmptyState } from "@/components/shared/EmptyState";
import { MobilePagination, type PaginationInfo } from "@/components/shared/MobilePagination";
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
      onClick={() => onPress(emp)}
    >
      <div className="px-4 pt-3.5 pb-3">
        <div className="flex items-start justify-between gap-2 mb-2">
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold text-foreground truncate">
              {emp.fullname}
            </p>
            <p className="text-xs text-muted-foreground mt-0.5">{emp.cccd}</p>
          </div>
        </div>

        <div className="flex items-center gap-4 text-xs">
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

        {emp.project?.name && (
          <p className="text-xs text-muted-foreground mt-2 truncate">
            {emp.project.name}
          </p>
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
