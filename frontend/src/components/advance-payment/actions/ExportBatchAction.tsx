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
      aria-label="Xuất lô chuyển tiền (Chuyển lô)"
      title="Xuất lô chuyển tiền"
      className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border bg-card px-3 text-sm font-medium text-foreground transition-all whitespace-nowrap hover:bg-accent disabled:pointer-events-none disabled:opacity-50"
    >
      {isLoading ? (
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
      ) : (
        <FileDown className="h-4 w-4 shrink-0" />
      )}
      <span className="hidden md:inline">Chuyển lô</span>
      <span className="md:hidden">Chuyển lô</span>
    </button>
  );
}
