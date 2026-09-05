import { useState } from "react";
import { Switch } from "@/components/ui/switch";
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
import { useToggleAdvanceRequestEnabled } from "@/hooks/api/useAdvancePayments";

interface AdvanceRequestToggleProps {
  projectId: number;
  employeeId: number;
  /** false = "Đang tạm ngừng" — new advance requests blocked for this employee */
  enabled: boolean;
}

/**
 * Per-employee advance request kill switch ("tạm ngừng ứng lương").
 * Pausing asks for confirmation — it blocks the employee's access to new
 * advance payments; re-enabling is immediate. stopPropagation keeps the
 * switch safe inside tappable table rows and cards.
 */
export function AdvanceRequestToggle({
  projectId,
  employeeId,
  enabled,
}: AdvanceRequestToggleProps) {
  const toggleMutation = useToggleAdvanceRequestEnabled();
  const [confirmOpen, setConfirmOpen] = useState(false);

  const fire = (value: boolean) => {
    if (toggleMutation.isPending) return; // guard double-fire during a slow PATCH
    toggleMutation.mutate({ projectId, employeeId, enabled: value });
  };

  const handleToggle = (checked: boolean) => {
    if (!checked) {
      setConfirmOpen(true);
      return;
    }
    fire(true);
  };

  return (
    <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
      <Switch
        checked={enabled}
        onCheckedChange={handleToggle}
        disabled={toggleMutation.isPending}
      />
      {enabled ? (
        <span className="text-xs text-muted-foreground whitespace-nowrap">Bật</span>
      ) : (
        <span className="whitespace-nowrap rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">
          Đang tạm ngừng
        </span>
      )}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent onClick={(e) => e.stopPropagation()}>
          <AlertDialogHeader>
            <AlertDialogTitle>Tạm ngừng ứng lương?</AlertDialogTitle>
            <AlertDialogDescription>
              Nhân viên sẽ không thể tạo yêu cầu ứng lương mới. Các yêu cầu đang
              chờ hoặc đã duyệt vẫn được giữ nguyên.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Giữ lại</AlertDialogCancel>
            <AlertDialogAction
              className="bg-amber-600 hover:bg-amber-700"
              disabled={toggleMutation.isPending}
              onClick={() => fire(false)}
            >
              Tạm ngừng
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
