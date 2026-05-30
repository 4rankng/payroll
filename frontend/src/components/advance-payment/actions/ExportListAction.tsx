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
      className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all border-l border-border disabled:opacity-50 disabled:pointer-events-none"
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
