import { History } from "lucide-react";

interface HistoryActionProps {
  onClick: () => void;
}

export function HistoryAction({ onClick }: HistoryActionProps) {
  return (
    <button
      onClick={onClick}
      aria-label="Lịch sử import và kết quả"
      title="Lịch sử import và kết quả"
      className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border bg-card px-3 text-sm font-medium text-foreground transition-all whitespace-nowrap hover:bg-accent"
    >
      <History className="h-4 w-4 shrink-0" />
      <span className="hidden md:inline">Lịch sử file</span>
      <span className="md:hidden">LS file</span>
    </button>
  );
}
