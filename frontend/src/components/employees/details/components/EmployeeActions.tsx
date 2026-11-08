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
 *   - View: destructive "Xóa" (admin-only) + neutral "Đổi mật khẩu"
 *     (admin/partner) on the left, primary "Đóng" on the right.
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
  // Employee deletion is admin-only — partners never see the "Xóa" button.
  const canDelete = userRole === 'admin';

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
  // Desktop: destructive "Xóa" sits left and isolated; "Đổi mật khẩu" and
  // "Đóng" right-align as quiet outline buttons — a dismiss never outshouts
  // the real actions. Phones keep the full-width stacked grid (tap targets).
  const actions = [
    canDelete && onDelete
      ? {
          key: "delete",
          label: "Xóa",
          icon: Trash2,
          variant: "ghost" as const,
          onClick: onDelete,
          className: cn(
            "border border-destructive/30 bg-destructive/10 text-destructive",
            "hover:bg-destructive/20 hover:text-destructive",
            "sm:mr-auto",
          ),
        }
      : null,
    canResetPassword && onResetPassword
      ? {
          key: "reset",
          label: "Đổi mật khẩu",
          icon: KeyRound,
          variant: "outline" as const,
          onClick: onResetPassword,
          className: "",
        }
      : null,
    onClose
      ? {
          key: "close",
          label: "Đóng",
          icon: X,
          variant: "outline" as const,
          onClick: onClose,
          className: "",
        }
      : null,
  ].filter((action): action is NonNullable<typeof action> => action !== null);

  if (actions.length === 0) return null;

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-end sm:gap-3">
      {actions.map((action) => {
        const Icon = action.icon;
        return (
          <Button
            key={action.key}
            type="button"
            variant={action.variant}
            onClick={action.onClick}
            className={cn(
              "h-11 min-h-11 w-full flex-row items-center justify-center gap-2 px-4 text-sm font-medium sm:w-auto sm:px-5 rounded-lg",
              action.className,
            )}
          >
            <Icon className="h-4 w-4" aria-hidden="true" />
            {action.label}
          </Button>
        );
      })}
    </div>
  );
});

EmployeeActions.displayName = "EmployeeActions";
