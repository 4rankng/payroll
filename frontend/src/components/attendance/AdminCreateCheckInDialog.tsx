import { useEffect, useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmployeeSelector } from "@/components/ui/employee-selector";
import { ProjectSelector } from "@/components/ui/project-selector";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAdminCheckInShifts, useAdminCreateCheckIn } from "@/hooks/api/useAdminAttendance";
import { AlertCircle, Loader2 } from "lucide-react";
import type { Employee } from "@/types/api/employee.types";
import type { Project } from "@/types/api/project.types";

interface AdminCreateCheckInDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function vietnamBusinessDate(): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Ho_Chi_Minh",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
}

/**
 * Admin entry point for a missed check-in. The form intentionally has no
 * checkout control: selecting a shift only creates an open check-in, leaving
 * the employee's normal GPS checkout as the completion/earning authority.
 */
export function AdminCreateCheckInDialog({ open, onOpenChange }: AdminCreateCheckInDialogProps) {
  const [employee, setEmployee] = useState<Employee | null>(null);
  const [project, setProject] = useState<Project | null>(null);
  const [shiftIndex, setShiftIndex] = useState<string>("");
  const date = vietnamBusinessDate();
  const createCheckIn = useAdminCreateCheckIn();
  const shiftsQuery = useAdminCheckInShifts(employee?.id ?? null, project?.id ?? null, date, { enabled: open });

  useEffect(() => {
    setShiftIndex("");
  }, [employee?.id, project?.id]);

  useEffect(() => {
    if (!open) {
      setEmployee(null);
      setProject(null);
      setShiftIndex("");
    }
  }, [open]);

  const shifts = shiftsQuery.data?.shifts ?? [];
  const canSubmit = employee !== null && project !== null && shiftIndex !== "" && !createCheckIn.isPending;
  const errorMessage = shiftsQuery.error instanceof Error ? shiftsQuery.error.message : null;

  const handleCreate = () => {
    if (!employee || !project || shiftIndex === "") return;
    createCheckIn.mutate(
      {
        employee_id: employee.id,
        project_id: project.id,
        date,
        shift_index: Number(shiftIndex),
      },
      { onSuccess: () => onOpenChange(false) },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[92dvh] w-[calc(100%-2rem)] max-w-lg flex-col overflow-hidden p-0 sm:w-full">
        <DialogHeader className="border-b px-4 pb-3 pt-4 sm:px-6">
          <DialogTitle>Tạo check-in cho nhân viên</DialogTitle>
          <DialogDescription>
            Chỉ tạo check-in cho hôm nay. Nhân viên vẫn phải tự tan ca tại khu vực chấm công để hoàn tất ca và ghi nhận thu nhập.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto px-4 py-4 sm:px-6">
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <p className="text-sm font-medium">Nhân viên</p>
              <EmployeeSelector
                value={employee}
                onSelect={setEmployee}
                showAllOption={false}
                className="min-h-11"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <p className="text-sm font-medium">Dự án</p>
              <ProjectSelector
                value={project}
                onSelect={setProject}
                activeOnly
                flexibleOnly
                className="min-h-11"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-medium" htmlFor="admin-check-in-shift">Ca làm việc hôm nay</label>
              <Select value={shiftIndex} onValueChange={setShiftIndex} disabled={!employee || !project || shiftsQuery.isLoading}>
                <SelectTrigger id="admin-check-in-shift" className="min-h-11">
                  <SelectValue placeholder={shiftsQuery.isLoading ? "Đang lấy ca làm việc..." : "Chọn ca làm việc"} />
                </SelectTrigger>
                <SelectContent>
                  {shifts.map((shift) => (
                    <SelectItem key={shift.index} value={String(shift.index)}>
                      {shift.label}{shift.position ? ` · ${shift.position}` : ""}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {employee && project && shifts.length === 0 && !shiftsQuery.isLoading && (
                <p className="text-xs text-muted-foreground">Không có ca làm việc hợp lệ cho phân công này.</p>
              )}
            </div>

            <Alert>
              <AlertCircle className="size-4" />
              <AlertDescription>
                Check-in do quản trị viên tạo không thay cho checkout GPS của nhân viên và không tự tạo thu nhập.
              </AlertDescription>
            </Alert>
            {errorMessage && (
              <Alert variant="destructive">
                <AlertDescription>{errorMessage}</AlertDescription>
              </Alert>
            )}
          </div>
        </div>

        <DialogFooter className="border-t px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))] sm:px-6 sm:pb-3">
          <Button type="button" variant="outline" className="min-h-11" onClick={() => onOpenChange(false)} disabled={createCheckIn.isPending}>
            Hủy
          </Button>
          <Button type="button" className="min-h-11" disabled={!canSubmit} onClick={handleCreate}>
            {createCheckIn.isPending && <Loader2 className="animate-spin" data-icon="inline-start" />}
            Tạo check-in
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
