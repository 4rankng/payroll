import { FileSpreadsheet } from "lucide-react";

interface ImportActionProps {
  onClick: () => void;
}

export function ImportAction({ onClick }: ImportActionProps) {
  return (
    <div className="flex items-center gap-px rounded-xl border border-border overflow-hidden shadow-sm hover:shadow-md transition-shadow">
      <button
        onClick={onClick}
        className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all"
      >
        <FileSpreadsheet className="h-4 w-4 shrink-0" />
        <span className="hidden sm:inline">Nhập bảng lương</span>
        <span className="sm:hidden">Nhập BL</span>
      </button>
    </div>
  );
}
