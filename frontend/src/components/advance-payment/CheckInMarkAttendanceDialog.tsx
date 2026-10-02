import { useEffect, useState } from "react";
import { AlertCircle, Loader2 } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { SearchableSelect } from "@/components/ui/searchable-select";
import {
  useAdminAttendances,
  useAdminCheckInShifts,
  useAdminCreateCheckIn,
  useApproveAttendance,
} from "@/hooks/api/useAdminAttendance";
import type { AdminAttendanceResponse, AdminCheckInShift } from "@/types/api/attendance.types";
import { formatVnTime } from "@/utils/vn-time";

interface CheckInMarkAttendanceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  employee: { id: number; name: string } | null;
  projectId: number | null;
  projectName?: string;
}

function vietnamBusinessDate(): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Ho_Chi_Minh",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}

const attendanceStatusLabel: Record<string, string> = {
  checked_in: "Đã điểm danh lúc",
  orphaned: "Quá hạn tan ca (vào lúc)",
  rejected: "Bị tự động từ chối (vào lúc)",
};

export function CheckInMarkAttendanceDialog({ open, onOpenChange, employee, projectId, projectName }: CheckInMarkAttendanceDialogProps) {
  const [shiftIndex, setShiftIndex] = useState<string>("");
  const [withCheckout, setWithCheckout] = useState(false);

  const today = vietnamBusinessDate();
  const createCheckIn = useAdminCreateCheckIn();
  const approve = useApproveAttendance({ successMessage: "Đã ghi nhận tan ca cho nhân viên." });
  const recordQuery = useAdminAttendances(
    {
      employee_id: employee?.id,
      project_id: projectId ?? undefined,
      from_date: today,
      to_date: today,
      pageSize: 10,
    },
    { enabled: open && employee !== null && projectId !== null },
  );
  const record: AdminAttendanceResponse | undefined = recordQuery.data?.data?.[0];
  const shiftsQuery = useAdminCheckInShifts(
    employee?.id ?? null,
    projectId ?? null,
    today,
    { enabled: open && employee !== null && projectId !== null && !record },
  );
  const shifts = shiftsQuery.data?.shifts ?? [];
  const selectedShift: AdminCheckInShift | undefined = shifts.find((s) => String(s.index) === shiftIndex);
  const shiftEnded = selectedShift ? new Date(selectedShift.end) <= new Date() : false;

  useEffect(() => {
    if (!open) {
      setShiftIndex("");
      setWithCheckout(false);
      return;
    }
  }, [open]);

  const canCreate = employee !== null && projectId !== null && shiftIndex !== "" && !createCheckIn.isPending;
  const errorMessage = shiftsQuery.error instanceof Error ? shiftsQuery.error.message : null;

  const handleCreate = () => {
    if (!employee || !projectId || shiftIndex === "") return;
    createCheckIn.mutate(
      {
        employee_id: employee.id,
        project_id: projectId,
        date: today,
        shift_index: Number(shiftIndex),
        with_checkout: withCheckout && shiftEnded,
      },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[92dvh] w-[calc(100%-2rem)] max-w-lg flex-col overflow-hidden p-0 sm:w-full">
        <DialogHeader className="border-b px-4 pb-3 pt-4 sm:px-6">
          <DialogTitle>Điểm danh hộ nhân viên</DialogTitle>
          <DialogDescription>
            Chốt chấm công hôm nay{projectName ? ` tại dự án ${projectName}` : ""} cho nhân viên chưa thể tự chấm công.
          </DialogDescription>
        </DialogHeader>
        {recordQuery.isLoading ? (
          <div className="flex flex-1 items-center justify-center py-10">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          </div>
        ) : record && record.status === "completed" ? (
          <div className="flex flex-1 flex-col justify-center gap-2 px-4 py-10 text-center sm:px-6">
            <p className="text-sm font-medium text-foreground">Nhân viên đã hoàn thành chấm công hôm nay.</p>
            <p className="text-sm text-muted-foreground">
              Vào {formatVnTime(record.check_in_time)}
              {record.check_out_time ? ` · Tan ca ${formatVnTime(record.check_out_time)}` : ""}
            </p>
          </div>
        ) : record ? (
          <div className="flex flex-1 flex-col gap-3 px-4 py-5 sm:px-6">
            <div className="rounded-lg border bg-muted/30 px-3 py-2.5">
              <p className="text-sm font-medium text-foreground">
                {attendanceStatusLabel[record.status] ?? "Vào lúc"} {formatVnTime(record.check_in_time)}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Xác nhận sẽ chốt tan ca theo giờ kết thúc ca cấu hình và tính lương nguyên ca.
              </p>
            </div>
          </div>
        ) : shiftsQuery.isLoading ? (
          <div className="flex flex-1 items-center justify-center py-10">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          </div>
        ) : (
          <div className="flex-1 space-y-4 overflow-y-auto px-4 py-4 sm:px-6">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="mark-attendance-shift" className="text-sm font-medium">Ca làm việc hôm nay</Label>
              <SearchableSelect
                triggerId="mark-attendance-shift"
                value={shiftIndex}
                onChange={setShiftIndex}
                placeholder="Chọn ca làm việc"
                searchPlaceholder="Tìm ca làm việc..."
                triggerClassName="min-h-11"
                options={shifts.map((shift) => ({
                  value: String(shift.index),
                  label: `${shift.label}${shift.position ? ` · ${shift.position}` : ""}`,
                }))}
              />
              {employee && projectId && shifts.length === 0 && !shiftsQuery.isLoading ? (
                <p className="text-xs text-muted-foreground">Không có ca làm việc hợp lệ cho phân công này.</p>
              ) : null}
            </div>

            {shiftEnded && shiftIndex !== "" ? (
              <div className="flex items-start gap-2.5 rounded-lg border px-3 py-2.5">
                <Checkbox
                  id="mark-attendance-checkout"
                  checked={withCheckout}
                  onCheckedChange={(checked) => setWithCheckout(checked === true)}
                  className="mt-0.5"
                />
                <div className="min-w-0 space-y-0.5">
                  <Label htmlFor="mark-attendance-checkout" className="text-sm font-medium leading-5">
                    Nhân viên đã tan ca
                  </Label>
                  <p className="text-xs text-muted-foreground">
                    Tạo bản ghi hoàn chỉnh: vào {selectedShift ? formatShiftTime(selectedShift.start) : ""} — tan {selectedShift ? formatShiftTime(selectedShift.end) : ""}, thu nhập tính theo cấu hình ca.
                  </p>
                </div>
              </div>
            ) : null}

            {withCheckout && shiftEnded ? null : (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  Check-in do quản trị viên tạo không thay cho checkout GPS của nhân viên.
                </AlertDescription>
              </Alert>
            )}
            {errorMessage ? (
              <Alert variant="destructive">
                <AlertDescription>{errorMessage}</AlertDescription>
              </Alert>
            ) : null}
          </div>
        )}
        <DialogFooter className="border-t px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-6 sm:pb-3">
          <Button
            type="button"
            variant="outline"
            className="min-h-11"
            onClick={() => onOpenChange(false)}
            disabled={createCheckIn.isPending || approve.isPending}
          >
            Đóng
          </Button>
          {record && record.status !== "completed" ? (
            <Button
              type="button"
              className="min-h-11"
              disabled={approve.isPending}
              onClick={() => approve.mutate({ id: record.id })}
            >
              {approve.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
              Xác nhận đã tan ca
            </Button>
          ) : !record && !recordQuery.isLoading ? (
            <Button type="button" className="min-h-11" disabled={!canCreate} onClick={handleCreate}>
              {createCheckIn.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
              {withCheckout && shiftEnded ? "Tạo check-in & tan ca" : "Tạo check-in"}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function formatShiftTime(iso: string): string {
  return new Intl.DateTimeFormat("vi-VN", { timeZone: "Asia/Ho_Chi_Minh", hour: "2-digit", minute: "2-digit" }).format(new Date(iso));
}