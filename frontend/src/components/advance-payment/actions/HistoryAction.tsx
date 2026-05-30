import { History } from "lucide-react";

interface HistoryActionProps {
  onClick: () => void;
}

export function HistoryAction({ onClick }: HistoryActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1.5 h-9 px-4 rounded-xl border border-border bg-card/90 backdrop-blur-sm shadow-sm hover:shadow-md text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all"
    >
      <History className="h-4 w-4 shrink-0" />
      <span className="hidden sm:inline">Lịch sử file</span>
      <span className="sm:hidden">Lịch sử</span>
    </button>
  );
}
