import { Button } from "@/components/ui/button";
import { memo } from "react";
import { authManager } from "@/lib/auth";
import { Save, X, Trash2, KeyRound } from "lucide-react";
import type { EmployeeActionsProps } from "../types";

export const EmployeeActions = memo(({
  onSave,
  onCancel,
  onDelete,
  onResetPassword,
  onClose,
  isDirty,
}: EmployeeActionsProps) => {
  const userRole = authManager.getUserRole();
  const canResetPassword = userRole === 'admin' || userRole === 'partner';

  // Editing mode — save/cancel
  if (onSave && onCancel) {
    return (
      <div className="flex items-center justify-between w-full">
        <Button
          variant="ghost"
          size="sm"
          onClick={onCancel}
          className="h-7 px-2.5 text-xs text-muted-foreground hover:text-foreground gap-1.5"
        >
          <X className="h-3 w-3" />
          Hủy
        </Button>
        <Button
          size="sm"
          onClick={onSave}
          disabled={!isDirty}
          className="h-7 px-4 gap-1.5 text-xs"
        >
          <Save className="h-3 w-3" />
          Lưu thay đổi
        </Button>
      </div>
    );
  }

  // View mode — delete/reset/close
  return (
    <div className="flex items-center justify-between w-full">
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={onDelete}
          className="h-7 px-2.5 text-xs text-destructive hover:text-destructive hover:bg-destructive/10 gap-1.5"
        >
          <Trash2 className="h-3 w-3" />
          Xóa
        </Button>

        {canResetPassword && onResetPassword && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onResetPassword}
            className="h-7 px-2.5 text-xs text-muted-foreground hover:text-foreground gap-1.5"
          >
            <KeyRound className="h-3 w-3" />
            Đổi MK
          </Button>
        )}
      </div>

      {onClose && (
        <Button
          size="sm"
          onClick={onClose}
          className="h-7 px-4 text-xs"
        >
          Đóng
        </Button>
      )}
    </div>
  );
});

EmployeeActions.displayName = "EmployeeActions";
