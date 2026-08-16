import { useCallback, useMemo, useState, useEffect } from "react";
import { Plus, Trash2, AlertCircle } from "lucide-react";

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
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import {
  useCreateFeeSchedule,
  useUpdateFeeSchedule,
} from "@/hooks/api/useAdvancePaymentFeeSchedules";
import {
  buildInitialFormState,
  setStructure,
  addTier,
  removeTier,
  updateTier,
  validateFormState,
  toCreateRequest,
  todayISO,
  computeFieldErrors,
  computeFieldWarnings,
  hasFieldErrors,
  summarizeBlockingIssues,
  loadDraft,
  saveDraft,
  clearDraft,
} from "./form-helpers";
import { detectUnsafePreview } from "./compare-helpers";
import { FEE_SCHEDULE_PRESETS } from "./presets";
import type {
  FeeScheduleFormMode,
  FeeScheduleFormState,
} from "./types";
import { FeeScheduleLivePreview } from "./FeeScheduleLivePreview";
import { FeeScheduleConfirmDialog } from "./FeeScheduleConfirmDialog";
import type { FeeScheduleEntry } from "@/types/api/advance-payment-fee-schedule.types";

interface Props {
  open: boolean;
  mode: FeeScheduleFormMode;
  onOpenChange: (open: boolean) => void;
  /** The currently active schedule, used by the confirmation diff. */
  currentActive: FeeScheduleEntry | null;
}

export const FeeScheduleFormDialog = ({
  open,
  mode,
  onOpenChange,
  currentActive,
}: Props) => {
  const isEdit = mode.kind === "edit";

  const [formState, setFormState] = useState<FeeScheduleFormState>(() =>
    buildInitialFormState(mode),
  );
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const today = useMemo(() => todayISO(), []);

  const create = useCreateFeeSchedule();
  const update = useUpdateFeeSchedule();
  const isSubmitting = create.isPending || update.isPending;

  const fieldErrors = useMemo(() => computeFieldErrors(formState), [formState]);
  const fieldWarnings = useMemo(
    () => computeFieldWarnings(formState),
    [formState],
  );
  const blockingValidation = useMemo(
    () => validateFormState(formState),
    [formState],
  );
  const blockingIssues = useMemo(
    () => summarizeBlockingIssues(formState),
    [formState],
  );
  const unsafe = useMemo(() => detectUnsafePreview(formState), [formState]);

  const isInvalid =
    !!blockingValidation || hasFieldErrors(fieldErrors) || !!unsafe;

  // Reset state whenever the dialog re-opens or the entry being edited changes.
  // For "create" mode, also restore from sessionStorage draft if present.
  useEffect(() => {
    if (!open) return;
    if (mode.kind === "edit") {
      setFormState(buildInitialFormState(mode));
    } else {
      const draft = loadDraft();
      setFormState(draft ?? buildInitialFormState(mode));
    }
    setSubmitError(null);
    setShowConfirm(false);
  }, [open, mode]);

  // Persist draft only in create mode while the dialog is open.
  useEffect(() => {
    if (!open || mode.kind !== "create") return;
    saveDraft(formState);
  }, [open, mode, formState]);

  const handleStructureChange = useCallback((value: string) => {
    setFormState((s) => setStructure(s, value === "tiered" ? "tiered" : "flat"));
  }, []);

  const handleAddTier = useCallback(() => {
    setFormState((s) => addTier(s));
  }, []);

  const handleRemoveTier = useCallback((index: number) => {
    setFormState((s) => removeTier(s, index));
  }, []);

  const handlePreset = useCallback(
    (key: string) => {
      const preset = FEE_SCHEDULE_PRESETS.find((p) => p.key === key);
      if (!preset) return;
      const next = preset.build();
      // Preserve the user's selected effective date if they already chose one.
      setFormState((s) => ({
        ...next,
        effectiveDate: s.effectiveDate || next.effectiveDate,
        notes: s.notes || next.notes,
      }));
    },
    [],
  );

  const handleAttemptSubmit = useCallback(() => {
    setSubmitError(blockingValidation);
    if (isInvalid) return;
    setShowConfirm(true);
  }, [blockingValidation, isInvalid]);

  const handleConfirm = useCallback(async () => {
    const body = toCreateRequest(formState);
    try {
      if (mode.kind === "edit") {
        await update.mutateAsync({ id: mode.entry.id, body });
      } else {
        await create.mutateAsync(body);
      }
      clearDraft();
      setShowConfirm(false);
      onOpenChange(false);
    } catch {
      // Error toast already shown by hook. Keep dialog open for retry.
      setShowConfirm(false);
    }
  }, [formState, mode, create, update, onOpenChange]);

  return (
    <>
      <Dialog open={open && !showConfirm} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {isEdit ? "Sửa cấu hình phí" : "Thêm cấu hình phí"}
            </DialogTitle>
            <DialogDescription>
              Cấu hình sẽ áp dụng cho các giao dịch ứng lương từ ngày hiệu
              lực. Cấu hình quá khứ và đang hoạt động không thể sửa hoặc xóa.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-6 md:grid-cols-[1fr_280px]">
            <div className="space-y-5">
              {/* Presets — create mode only */}
              {!isEdit && (
                <div className="space-y-2">
                  <Label className="text-xs text-muted-foreground">
                    Mẫu nhanh
                  </Label>
                  <div className="flex flex-wrap gap-1.5">
                    {FEE_SCHEDULE_PRESETS.map((p) => (
                      <Tooltip key={p.key}>
                        <TooltipTrigger asChild>
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="h-7 text-xs font-normal"
                            onClick={() => handlePreset(p.key)}
                          >
                            {p.label}
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent side="bottom">
                          {p.description}
                        </TooltipContent>
                      </Tooltip>
                    ))}
                  </div>
                </div>
              )}

              {/* Effective date */}
              <div className="space-y-1.5">
                <Label htmlFor="effectiveDate">Ngày hiệu lực</Label>
                <Input
                  id="effectiveDate"
                  type="date"
                  value={formState.effectiveDate}
                  min={today}
                  className={cn(
                    fieldErrors.effectiveDate &&
                      "border-destructive focus-visible:ring-destructive",
                  )}
                  onChange={(e) =>
                    setFormState((s) => ({
                      ...s,
                      effectiveDate: e.target.value,
                    }))
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Có hiệu lực từ ngày này. Phải là hôm nay hoặc tương lai.
                </p>
                {fieldErrors.effectiveDate && (
                  <p className="text-xs text-destructive">
                    {fieldErrors.effectiveDate}
                  </p>
                )}
              </div>

              {/* Structure */}
              <div className="space-y-2">
                <Label>Cấu trúc phí</Label>
                <RadioGroup
                  value={formState.structure}
                  onValueChange={handleStructureChange}
                  className="flex gap-6"
                >
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="flat" id="structure-flat" />
                    <Label htmlFor="structure-flat" className="font-normal">
                      Phí cố định
                    </Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="tiered" id="structure-tiered" />
                    <Label htmlFor="structure-tiered" className="font-normal">
                      Phí phân tầng
                    </Label>
                  </div>
                </RadioGroup>
              </div>

              <Separator />

              {/* Tiers */}
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label>Bậc phí</Label>
                  {formState.structure === "tiered" && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={handleAddTier}
                    >
                      <Plus className="mr-1.5 size-4" /> Thêm bậc
                    </Button>
                  )}
                </div>
                <p className="text-xs text-muted-foreground -mt-1">
                  Bậc 1 áp dụng cho mọi khoản từ 0 ₫ trở lên (mặc định). Thêm
                  bậc nếu muốn % phí khác cho khoản lớn hơn.
                </p>
                <div className="space-y-2">
                  {formState.tiers.map((tier, idx) => {
                    const pctErr = fieldErrors.tierPercentage[idx];
                    const minErr = fieldErrors.tierMinAmount[idx];
                    const pctWarn = fieldWarnings.tierPercentage[idx];
                    return (
                      <div
                        key={idx}
                        className="grid grid-cols-[1fr_1fr_auto] items-end gap-2"
                      >
                        <div className="space-y-1">
                          <Label className="text-xs text-muted-foreground">
                            {idx === 0 ? "Từ (cố định: 0)" : "Từ số tiền (₫)"}
                          </Label>
                          <Input
                            type="number"
                            min={0}
                            value={tier.minAmount}
                            disabled={idx === 0}
                            className={cn(
                              minErr &&
                                "border-destructive focus-visible:ring-destructive",
                            )}
                            onChange={(e) =>
                              setFormState((s) =>
                                updateTier(s, idx, {
                                  minAmount: Number(e.target.value) || 0,
                                }),
                              )
                            }
                          />
                          {minErr && (
                            <p className="text-xs text-destructive">
                              {minErr}
                            </p>
                          )}
                        </div>
                        <div className="space-y-1">
                          <Label className="text-xs text-muted-foreground">
                            Phần trăm (%)
                          </Label>
                          <Input
                            type="number"
                            min={0}
                            max={100}
                            step={0.1}
                            value={tier.percentage}
                            className={cn(
                              pctErr &&
                                "border-destructive focus-visible:ring-destructive",
                              !pctErr &&
                                pctWarn &&
                                "border-amber-400 focus-visible:ring-amber-400",
                            )}
                            onChange={(e) =>
                              setFormState((s) =>
                                updateTier(s, idx, {
                                  percentage: Number(e.target.value) || 0,
                                }),
                              )
                            }
                          />
                          {pctErr ? (
                            <p className="text-xs text-destructive">
                              {pctErr}
                            </p>
                          ) : pctWarn ? (
                            <p className="text-xs text-amber-600 dark:text-amber-400">
                              {pctWarn}
                            </p>
                          ) : null}
                        </div>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          disabled={
                            idx === 0 || formState.structure !== "tiered"
                          }
                          aria-label="Xóa bậc"
                          onClick={() => handleRemoveTier(idx)}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    );
                  })}
                </div>
              </div>

              <Separator />

              {/* Min fee */}
              <div className="space-y-1.5">
                <Label htmlFor="minFee">Phí tối thiểu (₫)</Label>
                <Input
                  id="minFee"
                  type="number"
                  min={1}
                  value={formState.minFeeVnd}
                  className={cn(
                    fieldErrors.minFee &&
                      "border-destructive focus-visible:ring-destructive",
                    !fieldErrors.minFee &&
                      fieldWarnings.minFee &&
                      "border-amber-400 focus-visible:ring-amber-400",
                  )}
                  onChange={(e) =>
                    setFormState((s) => ({
                      ...s,
                      minFeeVnd: Number(e.target.value) || 0,
                    }))
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Mức phí tối thiểu, kể cả khi phần trăm nhân ra số nhỏ hơn.
                </p>
                {fieldErrors.minFee ? (
                  <p className="text-xs text-destructive">
                    {fieldErrors.minFee}
                  </p>
                ) : fieldWarnings.minFee ? (
                  <p className="text-xs text-amber-600 dark:text-amber-400">
                    {fieldWarnings.minFee}
                  </p>
                ) : null}
              </div>

              {/* Notes */}
              <div className="space-y-1.5">
                <Label htmlFor="notes">Ghi chú (tùy chọn)</Label>
                <Textarea
                  id="notes"
                  rows={2}
                  value={formState.notes}
                  onChange={(e) =>
                    setFormState((s) => ({ ...s, notes: e.target.value }))
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Tùy chọn — ghi lý do thay đổi để dễ kiểm tra sau.
                </p>
              </div>

              {(submitError || unsafe) && (
                <div
                  role="alert"
                  className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/5 p-3"
                >
                  <AlertCircle className="h-4 w-4 text-destructive mt-0.5 shrink-0" />
                  <p className="text-xs text-destructive">
                    {unsafe ?? submitError}
                  </p>
                </div>
              )}
            </div>

            <FeeScheduleLivePreview
              tiers={formState.tiers}
              minFeeVnd={formState.minFeeVnd}
            />
          </div>

          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Hủy
            </Button>
            <Tooltip>
              <TooltipTrigger asChild>
                <span tabIndex={isInvalid ? 0 : -1}>
                  <Button
                    type="button"
                    onClick={handleAttemptSubmit}
                    disabled={isInvalid || isSubmitting}
                  >
                    {isEdit ? "Xem lại & lưu" : "Xem lại & tạo"}
                  </Button>
                </span>
              </TooltipTrigger>
              {isInvalid && (
                <TooltipContent side="top" align="end" className="max-w-xs">
                  <p className="font-medium mb-1">Vui lòng sửa các lỗi sau:</p>
                  <ul className="list-disc list-inside space-y-0.5 text-xs">
                    {(unsafe ? [unsafe] : blockingIssues).map((issue, i) => (
                      <li key={i}>{issue}</li>
                    ))}
                  </ul>
                </TooltipContent>
              )}
            </Tooltip>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <FeeScheduleConfirmDialog
        open={showConfirm}
        onOpenChange={setShowConfirm}
        pending={formState}
        current={currentActive}
        isSubmitting={isSubmitting}
        isEdit={isEdit}
        onConfirm={handleConfirm}
      />
    </>
  );
};
