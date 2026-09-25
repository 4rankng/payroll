import { useCallback, useMemo, useState } from "react";
import {
  CalendarClock,
  CheckCircle2,
  Pencil,
  Plus,
  Sparkles,
  Trash2,
} from "lucide-react";

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

import {
  useDeleteWeeklyPaymentFeeSchedule,
  useWeeklyPaymentFeeSchedules,
} from "@/hooks/api/useWeeklyPaymentFeeSchedules";
import type { WeeklyPaymentFeeScheduleEntry } from "@/types/api/weekly-payment-fee-schedule.types";

import { WeeklyPaymentFeeDeleteDialog } from "./WeeklyPaymentFeeDeleteDialog";
import { WeeklyPaymentFeeFormDialog } from "./WeeklyPaymentFeeFormDialog";
import {
  daysFromTodayISO,
  formatDaysFromNow,
  formatPercentageVi,
  formatVietnameseDate,
  isEntryEditable,
  type WeeklyPaymentFeeFormMode,
} from "./helpers";

export const WeeklyPaymentFeeScheduleSection = () => {
  const { data: entries = [], isLoading } = useWeeklyPaymentFeeSchedules();
  const deleteMutation = useDeleteWeeklyPaymentFeeSchedule();

  const [formMode, setFormMode] = useState<WeeklyPaymentFeeFormMode | null>(
    null,
  );
  const [pendingDelete, setPendingDelete] =
    useState<WeeklyPaymentFeeScheduleEntry | null>(null);

  const activeEntry = useMemo(
    () => entries.find((e) => e.isCurrentlyActive) ?? null,
    [entries],
  );
  const upcoming = useMemo(
    () =>
      [...entries]
        .filter((e) => e.isPending)
        .sort((a, b) => (a.effectiveDate < b.effectiveDate ? -1 : 1)),
    [entries],
  );
  const nextEntry = upcoming[0] ?? null;

  const sorted = useMemo(
    () =>
      [...entries].sort((a, b) =>
        a.effectiveDate < b.effectiveDate ? 1 : -1,
      ),
    [entries],
  );

  const handleOpenCreate = useCallback(() => {
    setFormMode({ kind: "create" });
  }, []);

  const handleEdit = useCallback((entry: WeeklyPaymentFeeScheduleEntry) => {
    setFormMode({ kind: "edit", entry });
  }, []);

  const handleDelete = useCallback((entry: WeeklyPaymentFeeScheduleEntry) => {
    setPendingDelete(entry);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!pendingDelete) return;
    try {
      await deleteMutation.mutateAsync(pendingDelete.id);
      setPendingDelete(null);
    } catch {
      // Error toast handled by hook; keep dialog open for retry.
    }
  }, [pendingDelete, deleteMutation]);

  const handleFormOpenChange = useCallback((open: boolean) => {
    if (!open) setFormMode(null);
  }, []);

  return (
    <section className="space-y-5">
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div>
          <h2 className="text-base font-semibold tracking-tight">
            Phí trả lương tuần
          </h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Tỷ lệ dịch vụ cộng vào tổng lương tuần khi chuyển lương tuần và đối
            soát với đối tác. Mỗi cấu hình áp dụng từ ngày hiệu lực cho đến khi
            có cấu hình mới thay thế.
          </p>
        </div>
        <Button
          type="button"
          size="sm"
          onClick={handleOpenCreate}
          className="shrink-0"
        >
          <Plus className="h-4 w-4 mr-1.5" />
          Thêm cấu hình mới
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
        <ActiveCard active={activeEntry} />
        {nextEntry ? (
          <UpcomingCard
            entry={nextEntry}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
        ) : (
          <NoUpcomingCard />
        )}
      </div>

      <div className="space-y-2">
        <h3 className="text-sm font-semibold">Lịch sử cấu hình</h3>
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : sorted.length === 0 ? (
          <EmptyState
            icon={CalendarClock}
            title="Chưa có cấu hình phí"
            description="Thêm cấu hình đầu tiên để bắt đầu áp dụng cho các kỳ trả lương tuần"
          />
        ) : (
          <div className="rounded-lg border overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Ngày hiệu lực</TableHead>
                  <TableHead>Trạng thái</TableHead>
                  <TableHead>Tỷ lệ phí</TableHead>
                  <TableHead>Ghi chú</TableHead>
                  {sorted.some((e) => e.isPending) && (
                    <TableHead className="w-[80px] text-right">
                      Hành động
                    </TableHead>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {sorted.map((entry) => (
                  <TableRow key={entry.id}>
                    <TableCell className="font-medium whitespace-nowrap">
                      {formatVietnameseDate(entry.effectiveDate)}
                    </TableCell>
                    <TableCell>
                      {entry.isCurrentlyActive && <Badge>Đang dùng</Badge>}
                      {entry.isPending && (
                        <Badge variant="secondary">Đang chờ</Badge>
                      )}
                      {!entry.isCurrentlyActive && !entry.isPending && (
                        <Badge variant="outline">Đã hết hạn</Badge>
                      )}
                    </TableCell>
                    <TableCell className="whitespace-nowrap">
                      {formatPercentageVi(entry.percentage)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[280px] truncate">
                      {entry.notes ?? ""}
                    </TableCell>
                    {sorted.some((e) => e.isPending) && (
                      <TableCell className="text-right">
                        {entry.isPending && (
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              aria-label="Sửa"
                              onClick={() => handleEdit(entry)}
                            >
                              <Pencil className="size-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              aria-label="Xóa"
                              onClick={() => handleDelete(entry)}
                            >
                              <Trash2 className="size-4" />
                            </Button>
                          </div>
                        )}
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>

      {formMode && (
        <WeeklyPaymentFeeFormDialog
          open
          mode={formMode}
          onOpenChange={handleFormOpenChange}
        />
      )}

      <WeeklyPaymentFeeDeleteDialog
        entry={pendingDelete}
        isDeleting={deleteMutation.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={handleConfirmDelete}
      />
    </section>
  );
};

const ActiveCard = ({
  active,
}: {
  active: WeeklyPaymentFeeScheduleEntry | null;
}) => {
  if (!active) {
    return (
      <div className="rounded-xl border border-dashed bg-muted/30 p-5 flex items-center justify-center text-center">
        <p className="text-sm text-muted-foreground">
          Chưa có cấu hình nào đang hoạt động
        </p>
      </div>
    );
  }

  return (
    <div className="rounded-xl border-2 border-primary/30 bg-primary/[0.04] p-4 sm:p-5 relative overflow-hidden">
      <div className="absolute inset-x-0 top-0 h-1 bg-primary/60" />
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/15">
            <CheckCircle2 className="h-4 w-4 text-primary" />
          </div>
          <span className="text-xs font-semibold uppercase tracking-wider text-primary">
            Đang áp dụng
          </span>
        </div>
      </div>

      <p className="text-base sm:text-lg leading-snug text-foreground">
        <strong className="font-semibold">
          {formatPercentageVi(active.percentage)}
        </strong>
      </p>

      <p className="text-xs text-muted-foreground mt-2">
        Có hiệu lực từ{" "}
        <strong className="font-medium text-foreground/80">
          {formatVietnameseDate(active.effectiveDate)}
        </strong>
      </p>
    </div>
  );
};

const UpcomingCard = ({
  entry,
  onEdit,
  onDelete,
}: {
  entry: WeeklyPaymentFeeScheduleEntry;
  onEdit: (entry: WeeklyPaymentFeeScheduleEntry) => void;
  onDelete: (entry: WeeklyPaymentFeeScheduleEntry) => void;
}) => {
  const days = daysFromTodayISO(entry.effectiveDate);
  const editable = isEntryEditable(entry);

  return (
    <div className="rounded-xl border-2 border-amber-400/40 bg-amber-50/50 dark:bg-amber-500/[0.06] p-4 sm:p-5 relative overflow-hidden">
      <div className="absolute inset-x-0 top-0 h-1 bg-amber-400/70" />
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/15">
            <CalendarClock className="h-4 w-4 text-amber-700" />
          </div>
          <span className="text-xs font-semibold uppercase tracking-wider text-amber-700 dark:text-amber-400">
            Đang chờ
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            aria-label="Sửa cấu hình sắp có hiệu lực"
            disabled={!editable}
            onClick={() => onEdit(entry)}
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            aria-label="Xóa cấu hình sắp có hiệu lực"
            disabled={!editable}
            onClick={() => onDelete(entry)}
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <p className="text-base sm:text-lg leading-snug text-foreground">
        <strong className="font-semibold">
          {formatPercentageVi(entry.percentage)}
        </strong>
      </p>

      <p className="text-xs text-muted-foreground mt-2">
        Sẽ thay thế cấu hình hiện tại từ{" "}
        <strong className="font-medium text-foreground/80">
          {formatVietnameseDate(entry.effectiveDate)}
        </strong>{" "}
        ({formatDaysFromNow(days)})
      </p>
    </div>
  );
};

const NoUpcomingCard = () => (
  <div className="rounded-xl border border-dashed bg-muted/20 p-4 sm:p-5 flex flex-col justify-center text-center gap-2">
    <Sparkles className="h-5 w-5 text-muted-foreground mx-auto" />
    <p className="text-sm text-muted-foreground leading-relaxed max-w-sm mx-auto">
      Không có cấu hình sắp có hiệu lực. Thêm cấu hình mới để thay đổi phí từ
      một ngày trong tương lai.
    </p>
  </div>
);
