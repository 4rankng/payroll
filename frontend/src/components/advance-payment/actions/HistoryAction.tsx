import { History } from "lucide-react";

interface HistoryActionProps {
  onClick: () => void;
}

export function HistoryAction({ onClick }: HistoryActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex min-h-11 items-center gap-1.5 rounded-xl border border-border bg-card/90 px-4 text-sm font-medium text-foreground shadow-sm backdrop-blur-sm transition-all whitespace-nowrap hover:bg-accent hover:text-accent-foreground hover:shadow-md"
    >
      <History className="h-4 w-4 shrink-0" />
      <span className="hidden sm:inline">Lịch sử file</span>
      <span className="sm:hidden">Lịch sử</span>
    </button>
  );
}
