import { useEffect, useState } from "react";
import { AlertTriangle } from "lucide-react";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { formatVietnameseDate } from "./helpers";
import type { WeeklyPaymentFeeScheduleEntry } from "@/types/api/weekly-payment-fee-schedule.types";

interface Props {
  entry: WeeklyPaymentFeeScheduleEntry | null;
  isDeleting: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

// Type-to-confirm delete: user must type the schedule's effective date
// (DD/MM/YYYY) verbatim to enable the destructive action.
export const WeeklyPaymentFeeDeleteDialog = ({
  entry,
  isDeleting,
  onCancel,
  onConfirm,
}: Props) => {
  const [typed, setTyped] = useState("");
  const expected = entry ? formatVietnameseDate(entry.effectiveDate) : "";
  const matches = typed.trim() === expected;

  useEffect(() => {
    if (entry) setTyped("");
  }, [entry]);

  return (
    <AlertDialog open={!!entry} onOpenChange={(o) => !o && onCancel()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-destructive/10">
              <AlertTriangle className="h-4 w-4 text-destructive" />
            </div>
            <AlertDialogTitle>Xóa cấu hình phí?</AlertDialogTitle>
          </div>
          <AlertDialogDescription className="pt-2">
            Cấu hình phí trả lương tuần hiệu lực từ ngày{" "}
            <strong className="font-semibold text-foreground">
              {expected}
            </strong>{" "}
            sẽ bị xóa vĩnh viễn. Hành động này không thể hoàn tác.
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className="space-y-2 px-6 mt-4 pb-2">
          <Label htmlFor="wpf-delete-confirm" className="text-xs">
            Nhập <strong className="font-semibold">{expected}</strong> để xác
            nhận xóa
          </Label>
          <Input
            id="wpf-delete-confirm"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            placeholder={expected}
            autoComplete="off"
            disabled={isDeleting}
          />
        </div>

        <AlertDialogFooter>
          <AlertDialogCancel disabled={isDeleting}>Hủy</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={onConfirm}
            disabled={!matches || isDeleting}
          >
            {isDeleting ? "Đang xóa..." : "Xóa cấu hình"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
};
