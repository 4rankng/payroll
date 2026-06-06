import { useCallback, useMemo, useState } from "react";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  useDeleteFeeSchedule,
  useFeeSchedules,
} from "@/hooks/api/useAdvancePaymentFeeSchedules";
import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

import { FeeScheduleCurrentBanner } from "./FeeScheduleCurrentBanner";
import { FeeScheduleCalculator } from "./FeeScheduleCalculator";
import { FeeScheduleList } from "./FeeScheduleList";
import { FeeScheduleFormDialog } from "./FeeScheduleFormDialog";
import { FeeScheduleDeleteDialog } from "./FeeScheduleDeleteDialog";
import type { FeeScheduleFormMode } from "./types";

export const FeeScheduleSection = () => {
  const { data: entries = [], isLoading } = useFeeSchedules();
  const deleteMutation = useDeleteFeeSchedule();

  const [formMode, setFormMode] = useState<FeeScheduleFormMode | null>(null);
  const [pendingDelete, setPendingDelete] = useState<FeeScheduleEntry | null>(
    null,
  );

  const activeEntry = useMemo(
    () => entries.find((e) => e.isCurrentlyActive) ?? null,
    [entries],
  );
  const upcoming = useMemo(
    () => entries.filter((e) => e.isPending),
    [entries],
  );

  const handleOpenCreate = useCallback(() => {
    setFormMode({ kind: "create" });
  }, []);

  const handleEdit = useCallback((entry: FeeScheduleEntry) => {
    setFormMode({ kind: "edit", entry });
  }, []);

  const handleDelete = useCallback((entry: FeeScheduleEntry) => {
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
            Cấu hình phí ứng lương
          </h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Mỗi cấu hình áp dụng từ ngày hiệu lực cho đến khi có cấu hình mới
            thay thế.
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

      <FeeScheduleCurrentBanner
        active={activeEntry}
        upcoming={upcoming}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />

      <FeeScheduleCalculator entries={entries} />

      <div className="space-y-2">
        <h3 className="text-sm font-semibold">Lịch sử cấu hình</h3>
        <FeeScheduleList
          entries={entries}
          isLoading={isLoading}
        />
      </div>

      {formMode && (
        <FeeScheduleFormDialog
          open
          mode={formMode}
          onOpenChange={handleFormOpenChange}
          currentActive={activeEntry}
        />
      )}

      <FeeScheduleDeleteDialog
        entry={pendingDelete}
        isDeleting={deleteMutation.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={handleConfirmDelete}
      />
    </section>
  );
};
