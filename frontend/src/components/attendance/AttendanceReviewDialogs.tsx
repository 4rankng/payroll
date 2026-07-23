import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { formatCurrency } from "@/utils/formatters";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";

export type ReviewMode = "approve" | "reject" | null;

export interface AttendanceReviewDialogsProps {
  mode: ReviewMode;
  row: AdminAttendanceResponse | null;
  onClose: () => void;
  onApprove: (row: AdminAttendanceResponse, note: string) => void;
  onReject: (row: AdminAttendanceResponse, note: string) => void;
  approveLoading?: boolean;
  rejectLoading?: boolean;
}

/**
 * Renders BOTH the approve and reject flows for an attendance row.
 *  - Approve uses ConfirmDialog (note optional; backend recomputes earning).
 *  - Reject uses a Dialog with a required reason Textarea.
 * Which dialog is open is controlled by `mode` (null = both closed).
 */
export function AttendanceReviewDialogs({
  mode,
  row,
  onClose,
  onApprove,
  onReject,
  approveLoading = false,
  rejectLoading = false,
}: AttendanceReviewDialogsProps) {
  const [rejectNote, setRejectNote] = useState("");
  const [approveNote, setApproveNote] = useState("");

  // Reset the reason inputs whenever a new row/mode opens so stale notes don't
  // carry over between successive reviews.
  useEffect(() => {
    if (mode !== null) {
      setRejectNote("");
      setApproveNote("");
    }
  }, [mode, row?.id]);

  const employeeLabel = row ? `${row.employee_name} · ${row.project_name || ""}`.trim() : "";
  const currentEarning = row?.earning_amount ?? 0;

  return (
    <>
      <ConfirmDialog
        open={mode === "approve" && row !== null}
        onOpenChange={(open) => {
          if (!open) onClose();
        }}
        title="Duyệt chấm công?"
        description={
          <div className="space-y-2">
            <p className="text-sm text-muted-foreground">
              Lương sẽ được tính lại theo ca làm việc đã cấu hình. Mọi lý do tự động từ chối trước đó sẽ bị xoá.
            </p>
            {employeeLabel && (
              <p className="text-xs font-medium text-foreground">{employeeLabel}</p>
            )}
            <p className="text-xs text-muted-foreground">
              Thu nhập hiện tại: <span className="font-semibold text-foreground">{formatCurrency(currentEarning)}</span>
            </p>
            <div className="space-y-1.5 pt-1">
              <Label htmlFor="approve-note" className="text-xs text-muted-foreground">
                Ghi chú duyệt (không bắt buộc)
              </Label>
              <Textarea
                id="approve-note"
                value={approveNote}
                onChange={(e) => setApproveNote(e.target.value)}
                placeholder="VD: Khách xác nhận làm đủ ca"
                className="min-h-[88px] text-sm"
              />
            </div>
          </div>
        }
        confirmText="Duyệt"
        confirmVariant="default"
        loading={approveLoading}
        onConfirm={() => {
          if (!row) return;
          onApprove(row, approveNote.trim());
        }}
      />

      <Dialog
        open={mode === "reject" && row !== null}
        onOpenChange={(open) => {
          if (!open) onClose();
        }}
      >
        <DialogContent className="max-h-[92dvh] overflow-y-auto pb-[calc(1rem+env(safe-area-inset-bottom))] sm:max-w-[440px]">
          <DialogHeader>
            <DialogTitle>Từ chối chấm công</DialogTitle>
            <DialogDescription>
              Thu nhập sẽ bị huỷ (0 ₫) và ghi lại lý do. Hành động này có thể đảo ngược bằng "Duyệt" sau đó.
            </DialogDescription>
          </DialogHeader>
          {employeeLabel && (
            <p className="text-xs font-medium text-foreground -mt-1">{employeeLabel}</p>
          )}
          <div className="space-y-1.5">
            <Label htmlFor="reject-note" className="text-sm">
              Lý do từ chối <span className="text-destructive">*</span>
            </Label>
            <Textarea
              id="reject-note"
              autoFocus
              value={rejectNote}
              onChange={(e) => setRejectNote(e.target.value)}
              placeholder="VD: Nhân viên rời dự án sớm"
              className="min-h-[112px]"
            />
            <p className="text-[11px] text-muted-foreground">
              Lý do này sẽ được lưu vào ghi chú từ chối lương của bản ghi.
            </p>
          </div>
          <DialogFooter className="grid grid-cols-1 gap-2 sm:flex sm:gap-2">
            <Button className="min-h-11" variant="outline" onClick={onClose} disabled={rejectLoading}>
              Hủy
            </Button>
            <Button
              variant="destructive"
              className="min-h-11"
              disabled={rejectNote.trim().length === 0 || rejectLoading}
              onClick={() => {
                if (!row) return;
                onReject(row, rejectNote.trim());
              }}
            >
              {rejectLoading ? "Đang xử lý..." : "Từ chối"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
