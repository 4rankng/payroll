import { useMemo } from "react";

import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { EmptyState } from "@/components/shared/EmptyState";

import { formatCurrency, formatDate } from "@/utils/formatters";
import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

interface Props {
  entries: FeeScheduleEntry[];
  isLoading: boolean;
}

export const FeeScheduleList = ({
  entries,
  isLoading,
}: Props) => {
  const sorted = useMemo(
    () =>
      [...entries].sort((a, b) =>
        a.effectiveDate < b.effectiveDate ? 1 : -1,
      ),
    [entries],
  );

  if (isLoading) {
    return (
      <div className="space-y-2">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (sorted.length === 0) {
    return (
      <EmptyState
        title="Chưa có cấu hình phí"
        description="Thêm cấu hình đầu tiên để bắt đầu áp dụng cho các giao dịch ứng lương"
      />
    );
  }

  return (
    <div className="rounded-lg border overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Ngày hiệu lực</TableHead>
            <TableHead>Trạng thái</TableHead>
            <TableHead>Cấu trúc phí</TableHead>
            <TableHead>Phí tối thiểu</TableHead>
            <TableHead>Ghi chú</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.map((entry) => (
              <TableRow key={entry.id}>
                <TableCell className="font-medium whitespace-nowrap">
                  {formatDate(entry.effectiveDate)}
                </TableCell>
                <TableCell>
                  {entry.isCurrentlyActive && (
                    <Badge>Đang dùng</Badge>
                  )}
                  {entry.isPending && (
                    <Badge variant="secondary">Đang chờ</Badge>
                  )}
                  {!entry.isCurrentlyActive && !entry.isPending && (
                    <Badge variant="outline">Đã hết hạn</Badge>
                  )}
                </TableCell>
                <TableCell className="text-sm">{entry.summary}</TableCell>
                <TableCell className="whitespace-nowrap">
                  {formatCurrency(entry.minFeeVnd)}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground max-w-[280px] truncate">
                  {entry.notes ?? ""}
                </TableCell>
              </TableRow>
            ))}

        </TableBody>
      </Table>
    </div>
  );
};
