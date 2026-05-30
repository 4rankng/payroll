import { FileDown, Loader2 } from "lucide-react";

interface ExportBatchActionProps {
  onClick: () => void;
  isLoading?: boolean;
}

export function ExportBatchAction({ onClick, isLoading = false }: ExportBatchActionProps) {
  return (
    <button
      onClick={onClick}
      disabled={isLoading}
      className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all disabled:opacity-50 disabled:pointer-events-none"
    >
      {isLoading ? (
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
      ) : (
        <FileDown className="h-4 w-4 shrink-0" />
      )}
      Chuyển lô
    </button>
  );
}
