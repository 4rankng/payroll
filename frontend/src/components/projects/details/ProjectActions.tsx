import { Button } from "@/components/ui/button";
import { Edit3, Trash2, Save, X } from "lucide-react";
import { memo } from "react";

interface ProjectActionsProps {
  isEditing: boolean;
  onEdit: () => void;
  onSave: () => void;
  onCancel: () => void;
  onDelete: () => void;
}

export const ProjectActions = memo(({
  isEditing,
  onEdit,
  onSave,
  onCancel,
  onDelete,
}: ProjectActionsProps) => {
  if (isEditing) {
    return (
      <div className="flex gap-3">
        <Button
          variant="outline"
          onClick={onCancel}
          className="flex-1"
        >
          <X className="w-4 h-4 mr-2" />
          Đóng
        </Button>
        <Button
          onClick={onSave}
          className="flex-1"
        >
          <Save className="w-4 h-4 mr-2" />
          Lưu thay đổi
        </Button>
      </div>
    );
  }

  return (
    <div className="flex gap-3">
      {/* Primary actions (left) */}
      <Button
        variant="outline"
        onClick={onEdit}
        className="flex-1"
      >
        <Edit3 className="w-4 h-4 mr-2" />
        Chỉnh sửa
      </Button>

      {/* Destructive actions (right) */}
      <Button
        variant="destructive"
        onClick={onDelete}
        className="flex-1"
      >
        <Trash2 className="w-4 h-4 mr-2" />
        Xóa dự án
      </Button>
    </div>
  );
});

ProjectActions.displayName = "ProjectActions";
