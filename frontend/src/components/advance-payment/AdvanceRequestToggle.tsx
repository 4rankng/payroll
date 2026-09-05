import { useState } from "react";
import { Switch } from "@/components/ui/switch";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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
    return toggleMutation
      .mutateAsync({ projectId, employeeId, enabled: value })
      .then(() => undefined);
  };

  const handleToggle = (checked: boolean) => {
    if (!checked) {
      setConfirmOpen(true);
      return;
    }
    fire(true);
  };

  return (
    <div className="flex items-center" onClick={(e) => e.stopPropagation()}>
      <Switch
        checked={enabled}
        onCheckedChange={handleToggle}
        disabled={toggleMutation.isPending}
      />
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Tạm ngừng ứng lương?"
        description="Nhân viên sẽ không thể tạo yêu cầu ứng lương mới. Các yêu cầu đang chờ hoặc đã duyệt vẫn được giữ nguyên."
        confirmText="Tạm ngừng"
        cancelText="Giữ lại"
        confirmVariant="destructive"
        onConfirm={() => fire(false)}
      />
    </div>
  );
}
