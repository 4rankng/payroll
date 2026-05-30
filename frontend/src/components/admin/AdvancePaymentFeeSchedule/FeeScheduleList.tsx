import { useMemo } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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

import { isEntryEditable } from "./form-helpers";

interface Props {
  entries: FeeScheduleEntry[];
  isLoading: boolean;
  onEdit: (entry: FeeScheduleEntry) => void;
  onDelete: (entry: FeeScheduleEntry) => void;
}

export const FeeScheduleList = ({
  entries,
  isLoading,
  onEdit,
  onDelete,
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
            <TableHead className="w-[120px] text-right">Hành động</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.map((entry) => {
            const editable = isEntryEditable(entry);
            return (
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
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label="Sửa"
                      disabled={!editable}
                      onClick={() => onEdit(entry)}
                    >
                      <Pencil className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label="Xóa"
                      disabled={!editable}
                      onClick={() => onDelete(entry)}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
};
