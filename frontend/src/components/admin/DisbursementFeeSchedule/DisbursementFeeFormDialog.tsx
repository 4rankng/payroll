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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import {
  useCreateDisbursementFeeSchedule,
  useUpdateDisbursementFeeSchedule,
} from "@/hooks/api/useDisbursementFeeSchedules";

import {
  buildInitialFormState,
  toCreateRequest,
  todayISO,
  validateFormState,
  type DisbursementFeeFormMode,
  type DisbursementFeeFormState,
} from "./helpers";
import {
  DISBURSEMENT_PROVIDERS,
  PROVIDER_LABELS,
  type DisbursementProvider,
} from "@/types/api/disbursement-fee-schedule.types";

interface Props {
  open: boolean;
  mode: DisbursementFeeFormMode;
  onOpenChange: (open: boolean) => void;
}

export const DisbursementFeeFormDialog = ({
  open,
  mode,
  onOpenChange,
}: Props) => {
  const [state, setState] = useState<DisbursementFeeFormState>(() =>
    buildInitialFormState(mode),
  );
  const createMutation = useCreateDisbursementFeeSchedule();
  const updateMutation = useUpdateDisbursementFeeSchedule();
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
              ? "Sửa cấu hình phí giao dịch chi hộ"
              : "Thêm cấu hình phí giao dịch chi hộ"}
          </DialogTitle>
          <DialogDescription>
            Mỗi cấu hình áp dụng từ ngày hiệu lực cho đến khi có cấu hình mới
            thay thế. Các giao dịch đã phát sinh giữ nguyên mức phí cũ.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="dfs-provider">Nhà cung cấp</Label>
            <Select
              value={state.provider}
              onValueChange={(v) =>
                setState((s) => ({
                  ...s,
                  provider: v as DisbursementProvider,
                }))
              }
              disabled={isSaving || isEdit}
            >
              <SelectTrigger id="dfs-provider">
                <SelectValue placeholder="Chọn nhà cung cấp" />
              </SelectTrigger>
              <SelectContent>
                {DISBURSEMENT_PROVIDERS.map((p) => (
                  <SelectItem key={p} value={p}>
                    {PROVIDER_LABELS[p]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              Chọn nhà cung cấp dịch vụ chi hộ áp dụng mức phí này.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="dfs-effective-date">Áp dụng từ ngày</Label>
            <Input
              id="dfs-effective-date"
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
            <Label htmlFor="dfs-fee-vnd">Mức phí (₫)</Label>
            <Input
              id="dfs-fee-vnd"
              type="number"
              min={0}
              max={1_000_000}
              step={1}
              value={state.feeVnd}
              onChange={(e) =>
                setState((s) => ({
                  ...s,
                  feeVnd: Number.parseInt(e.target.value || "0", 10),
                }))
              }
              disabled={isSaving}
              required
            />
            <p className="text-xs text-muted-foreground">
              Mức phí cố định cho mỗi giao dịch chi hộ qua nhà cung cấp đã chọn.
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="dfs-notes">Ghi chú (không bắt buộc)</Label>
            <Textarea
              id="dfs-notes"
              value={state.notes}
              onChange={(e) =>
                setState((s) => ({ ...s, notes: e.target.value }))
              }
              disabled={isSaving}
              rows={2}
              placeholder="Ví dụ: 9pay điều chỉnh giá từ 01/06"
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
