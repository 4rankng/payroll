import { Download, Loader2 } from "lucide-react";

interface ExportListActionProps {
  onClick: () => void;
  isLoading?: boolean;
}

export function ExportListAction({ onClick, isLoading = false }: ExportListActionProps) {
  return (
    <button
      onClick={onClick}
      disabled={isLoading}
      aria-label="Xuất danh sách nhân viên ứng lương"
      title="Xuất danh sách nhân viên ứng lương"
      className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border bg-card px-3 text-sm font-medium text-foreground transition-all whitespace-nowrap hover:bg-accent disabled:pointer-events-none disabled:opacity-50"
    >
      {isLoading ? (
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
      ) : (
        <Download className="h-4 w-4 shrink-0" />
      )}
      <span className="hidden md:inline">Xuất danh sách</span>
      <span className="md:hidden">Xuất DS</span>
    </button>
  );
}
