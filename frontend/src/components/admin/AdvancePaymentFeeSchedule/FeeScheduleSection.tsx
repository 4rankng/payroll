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

import { SettingsSectionHeader } from '@/components/settings/SettingsSection';

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
    <section className="space-y-3">
      <SettingsSectionHeader
        kicker="Phí"
        title="Phí ứng lương tháng"
        meta="Cấu hình hiệu lực theo ngày"
        actions={
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={handleOpenCreate}
            className="shrink-0"
          >
            <Plus className="mr-1.5 h-4 w-4" />
            Thêm cấu hình mới
          </Button>
        }
      />

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
