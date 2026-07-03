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
      className="inline-flex min-h-11 items-center gap-1.5 border-l border-border bg-card/90 px-4 text-sm font-medium text-foreground backdrop-blur-sm transition-all whitespace-nowrap hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
    >
      {isLoading ? (
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
      ) : (
        <Download className="h-4 w-4 shrink-0" />
      )}
      <span className="hidden sm:inline">Xuất DS</span>
      <span className="sm:hidden">Xuất</span>
    </button>
  );
}
