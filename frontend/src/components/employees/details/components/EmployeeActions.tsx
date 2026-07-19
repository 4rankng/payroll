import { Button } from "@/components/ui/button";
import { memo } from "react";
import { authManager } from "@/lib/auth";
import { Save, X, Trash2, KeyRound } from "lucide-react";
import { cn } from "@/lib/utils";
import type { EmployeeActionsProps } from "../types";

/**
 * Footer action bar for EmployeeDetailsSheet.
 *
 * Tailkit-inspired (a-c-form-actions-02/03) button styling translated to
 * project semantic tokens. All buttons meet the 44px mobile tap-target
 * standard (min-h-11). Layout adapts to viewport:
 *   - Mobile (<sm): full-width stacked or grid-cols-2
 *   - Tablet/desktop (>=sm): space-between with grouped actions
 *
 * Two modes:
 *   - View: destructive "Xóa" + neutral "Đổi mật khẩu" on the left,
 *     primary "Đóng" on the right.
 *   - Edit: full-width row with ghost "Hủy" (left) and primary
 *     "Lưu thay đổi" (right).
 */
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

  // ─── Editing mode — save/cancel ──────────────────────────────────────────
  if (onSave && onCancel) {
    return (
      <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
        <Button
          type="button"
          variant="ghost"
          onClick={onCancel}
          className="min-h-11 w-full border border-border bg-card"
        >
          <X className="h-4 w-4" />
          Hủy
        </Button>
        <Button
          type="button"
          onClick={onSave}
          disabled={!isDirty}
          className="min-h-11 w-full"
        >
          <Save className="h-4 w-4" />
          Lưu thay đổi
        </Button>
      </div>
    );
  }

  // ─── View mode — delete/reset/close ──────────────────────────────────────
  // Layout: secondary actions (Xóa, Đổi mật khẩu) on the left grouped together,
  // primary "Đóng" action on the right. On very narrow screens the actions wrap
  // and the primary button takes full width below.
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      {/* Secondary actions group */}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="ghost"
          onClick={onDelete}
          className={cn(
            "min-h-11 border border-destructive/30 bg-destructive/10 text-destructive",
            "hover:bg-destructive/20 hover:text-destructive",
          )}
        >
          <Trash2 className="h-4 w-4" />
          Xóa
        </Button>

        {canResetPassword && onResetPassword && (
          <Button
            type="button"
            variant="ghost"
            onClick={onResetPassword}
            className="min-h-11 border border-border bg-card text-foreground hover:bg-muted"
          >
            <KeyRound className="h-4 w-4" />
            Đổi mật khẩu
          </Button>
        )}
      </div>

      {/* Primary action */}
      {onClose && (
        <Button
          type="button"
          onClick={onClose}
          className="min-h-11 w-full sm:w-auto"
        >
          Đóng
        </Button>
      )}
    </div>
  );
});

EmployeeActions.displayName = "EmployeeActions";
