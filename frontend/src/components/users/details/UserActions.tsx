import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { memo } from "react";
import { cn } from "@/lib/utils";
import { Pencil, Check, X, Trash2, KeyRound } from "lucide-react";

interface UserActionsProps {
  isEditing: boolean;
  onEdit: () => void;
  onSave: () => void;
  onCancel: () => void;
  onDelete: () => void;
  onResetPassword: () => void;
  loading?: boolean;
}

export const UserActions = memo(({
  isEditing,
  onEdit,
  onSave,
  onCancel,
  onDelete,
  onResetPassword,
  loading = false,
}: UserActionsProps) => {
  if (isEditing) {
    return (
      <>
        <Button
          variant="ghost"
          size="sm"
          onClick={onCancel}
          disabled={loading}
          className="flex-1 h-9 text-muted-foreground hover:text-foreground"
        >
          <X className="w-3.5 h-3.5 mr-1.5" />
          Hủy
        </Button>
        <Button
          variant="default"
          size="sm"
          onClick={onSave}
          disabled={loading}
          className="flex-1 h-9"
        >
          <Check className="w-3.5 h-3.5 mr-1.5" />
          Lưu thay đổi
        </Button>
      </>
    );
  }

  return (
    <>
      {/* Destructive actions — icon only, pushed left */}
      <div className="flex gap-1.5">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={onResetPassword}
              disabled={loading}
              className="h-9 w-9 text-muted-foreground hover:text-foreground hover:bg-muted"
              aria-label="Đổi mật khẩu"
            >
              <KeyRound className="w-4 h-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">Đổi mật khẩu</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              onClick={onDelete}
              disabled={loading}
              className="h-9 w-9 text-muted-foreground hover:text-destructive hover:bg-destructive/8"
              aria-label="Xóa người dùng"
            >
              <Trash2 className="w-4 h-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="top">Xóa người dùng</TooltipContent>
        </Tooltip>
      </div>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Primary action */}
      <Button
        variant="outline"
        size="sm"
        onClick={onEdit}
        disabled={loading}
        className="h-9 px-4"
      >
        <Pencil className="w-3.5 h-3.5 mr-1.5" />
        Chỉnh sửa
      </Button>
    </>
  );
});

UserActions.displayName = "UserActions";
