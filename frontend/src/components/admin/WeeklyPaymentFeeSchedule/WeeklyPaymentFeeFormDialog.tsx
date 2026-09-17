import { useEffect, useMemo, useState } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

import {
  useCreateWeeklyPaymentFeeSchedule,
  useUpdateWeeklyPaymentFeeSchedule,
} from "@/hooks/api/useWeeklyPaymentFeeSchedules";

import {
  buildInitialFormState,
  toCreateRequest,
  todayISO,
  validateFormState,
  type WeeklyPaymentFeeFormMode,
  type WeeklyPaymentFeeFormState,
} from "./helpers";

interface Props {
  open: boolean;
  mode: WeeklyPaymentFeeFormMode;
  onOpenChange: (open: boolean) => void;
}

export const WeeklyPaymentFeeFormDialog = ({
  open,
  mode,
  onOpenChange,
}: Props) => {
  const [state, setState] = useState<WeeklyPaymentFeeFormState>(() =>
    buildInitialFormState(mode),
  );
  const createMutation = useCreateWeeklyPaymentFeeSchedule();
  const updateMutation = useUpdateWeeklyPaymentFeeSchedule();
  const isSaving = createMutation.isPending || updateMutation.isPending;

  // Re-seed when reopening so the form reflects the new mode/entry.
  useEffect(() => {
    if (open) setState(buildInitialFormState(mode));
  }, [open, mode]);

  const validationError = useMemo(() => validateFormState(state), [state]);
  const isEdit = mode.kind === "edit";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (validationError) return;
    try {
      if (mode.kind === "edit") {
        await updateMutation.mutateAsync({
          id: mode.entry.id,
          body: toCreateRequest(state),
        });
      } else {
        await createMutation.mutateAsync(toCreateRequest(state));
      }
      onOpenChange(false);
    } catch {
      // Error toast handled by hooks; keep dialog open so user can retry.
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEdit
              ? "Sửa cấu hình phí trả lương tuần"
              : "Thêm cấu hình phí trả lương tuần"}
          </DialogTitle>
          <DialogDescription>
            Cấu hình áp dụng từ ngày hiệu lực cho đến khi có cấu hình mới thay
            thế. Các giao dịch đã phát sinh giữ nguyên mức phí cũ.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="wpf-effective-date">Áp dụng từ ngày</Label>
            <Input
              id="wpf-effective-date"
              type="date"
              value={state.effectiveDate}
              min={todayISO()}
              onChange={(e) =>
                setState((s) => ({ ...s, effectiveDate: e.target.value }))
              }
              disabled={isSaving}
              required
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="wpf-percentage">Tỷ lệ phí (%)</Label>
            <Input
              id="wpf-percentage"
              type="number"
              min={0}
              max={100}
              step="0.1"
              value={state.percentage}
              onChange={(e) =>
                setState((s) => ({
                  ...s,
                  percentage: Number.parseFloat(e.target.value || "0"),
                }))
              }
              disabled={isSaving}
              required
            />
            <p className="text-xs text-muted-foreground">
              Tỷ lệ dịch vụ cộng vào tổng lương tuần khi đối soát với đối tác
              (công nợ phải thu = tiền chuyển × (1 + tỷ lệ)).
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="wpf-notes">Ghi chú (không bắt buộc)</Label>
            <Textarea
              id="wpf-notes"
              value={state.notes}
              onChange={(e) =>
                setState((s) => ({ ...s, notes: e.target.value }))
              }
              disabled={isSaving}
              rows={2}
              placeholder="Ví dụ: giảm phí dịch vụ từ 2% xuống 1,8% để các cấu hình placeholder"
            />
          </div>

          {validationError && (
            <p className="text-xs text-destructive">{validationError}</p>
          )}

          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
              disabled={isSaving}
            >
              Hủy
            </Button>
            <Button type="submit" disabled={!!validationError || isSaving}>
              {isSaving ? "Đang lưu..." : isEdit ? "Cập nhật" : "Thêm cấu hình"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
