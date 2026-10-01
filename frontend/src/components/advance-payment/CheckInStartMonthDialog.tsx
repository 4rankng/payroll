import { useState } from "react";
import { addMonths, format, startOfMonth } from "date-fns";
import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import type { CheckInStartMonth } from "@/types/api/project-employee.types";

/** The day-1 date a choice lands on, shown so the admin sees the real effect. */
export function getCheckInStartDate(startMonth: CheckInStartMonth, now = new Date()): Date {
  return startMonth === "this_month" ? startOfMonth(now) : startOfMonth(addMonths(now, 1));
}

const choices: { value: CheckInStartMonth; label: string; description: string }[] = [
  {
    value: "this_month",
    label: "Tháng này",
    description: "Dịch vụ tự chấm công dùng được ngay, tính từ ngày 1 tháng này.",
  },
  {
    value: "next_month",
    label: "Tháng sau",
    description: "Hệ thống sẽ tự kích hoạt vào ngày 1 của tháng sau.",
  },
];

interface CheckInStartMonthDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Employee the choice applies to, named in the dialog. */
  employeeName: string;
  /**
   * True when the row already waits for activation: the same dialog then moves
   * that queued activation to the other month instead of creating a new one.
   */
  isReschedule?: boolean;
  isSubmitting?: boolean;
  onConfirm: (startMonth: CheckInStartMonth) => void;
}

export function CheckInStartMonthDialog({
  open,
  onOpenChange,
  employeeName,
  isReschedule = false,
  isSubmitting = false,
  onConfirm,
}: CheckInStartMonthDialogProps) {
  const [startMonth, setStartMonth] = useState<CheckInStartMonth>("next_month");
  const now = new Date();
  const effectiveDate = format(getCheckInStartDate(startMonth, now), "dd/MM/yyyy");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isReschedule ? "Đổi tháng kích hoạt" : "Bật tự chấm công"}
          </DialogTitle>
          <DialogDescription>
            Chọn tháng bắt đầu cho <span className="font-medium text-foreground">{employeeName}</span>.
            Ngày bắt đầu luôn là ngày 1 của tháng đã chọn.
          </DialogDescription>
        </DialogHeader>

        <RadioGroup
          value={startMonth}
          onValueChange={(value) => setStartMonth(value as CheckInStartMonth)}
          className="gap-2"
        >
          {choices.map((choice) => {
            const choiceDate = format(getCheckInStartDate(choice.value, now), "dd/MM/yyyy");
            const id = `check-in-start-month-${choice.value}`;
            return (
              <Label
                key={choice.value}
                htmlFor={id}
                className="flex cursor-pointer items-start gap-3 rounded-lg border border-border p-3 has-[[data-state=checked]]:border-primary has-[[data-state=checked]]:bg-primary/5"
              >
                <RadioGroupItem id={id} value={choice.value} className="mt-0.5" />
                <span className="min-w-0 space-y-0.5">
                  <span className="block text-sm font-medium text-foreground">
                    {choice.label} — từ {choiceDate}
                  </span>
                  <span className="block text-xs text-muted-foreground">
                    {choice.description}
                  </span>
                </span>
              </Label>
            );
          })}
        </RadioGroup>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            className="min-h-11"
            disabled={isSubmitting}
            onClick={() => onOpenChange(false)}
          >
            Hủy
          </Button>
          <Button
            type="button"
            className="min-h-11 gap-1.5"
            disabled={isSubmitting}
            onClick={() => onConfirm(startMonth)}
          >
            {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
            {isReschedule ? "Xác nhận đổi tháng" : `Xác nhận từ ${effectiveDate}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
