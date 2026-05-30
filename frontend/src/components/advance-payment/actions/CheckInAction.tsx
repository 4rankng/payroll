import { ScanFace } from "lucide-react";

interface CheckInActionProps {
  onClick: () => void;
}

export function CheckInAction({ onClick }: CheckInActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1.5 h-9 px-4 rounded-xl border border-border bg-card/90 backdrop-blur-sm shadow-sm hover:shadow-md text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all"
    >
      <ScanFace className="h-4 w-4 shrink-0" />
      <span className="hidden sm:inline">Điểm danh</span>
      <span className="sm:hidden">ĐD</span>
    </button>
  );
}
