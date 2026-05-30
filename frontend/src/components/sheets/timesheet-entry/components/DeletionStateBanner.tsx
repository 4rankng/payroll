import { memo } from "react";
import { Trash2, Undo2 } from "lucide-react";

interface DeletionStateBannerProps {
  onUndo: () => void;
}

export const DeletionStateBanner = memo(
  ({ onUndo }: DeletionStateBannerProps) => (
    <div className="flex items-center justify-between gap-2 px-3 py-2 bg-red-100 rounded-xl border border-red-200">
      <div className="flex items-center gap-2">
        <Trash2 className="h-3.5 w-3.5 text-red-500 shrink-0" />
        <p className="text-xs font-medium text-red-700">Sẽ xóa khi lưu</p>
      </div>
      <button
        onClick={onUndo}
        className="flex items-center gap-1 text-xs font-semibold text-red-600 hover:text-red-800 active:opacity-70 transition-opacity"
      >
        <Undo2 className="h-3 w-3" />
        Hoàn tác
      </button>
    </div>
  ),
);

DeletionStateBanner.displayName = "DeletionStateBanner";
