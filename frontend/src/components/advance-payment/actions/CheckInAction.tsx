import { ScanFace } from "lucide-react";

interface CheckInActionProps {
  onClick: () => void;
}

export function CheckInAction({ onClick }: CheckInActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex min-h-11 items-center gap-1.5 rounded-xl border border-border bg-card/90 px-4 text-sm font-medium text-foreground shadow-sm backdrop-blur-sm transition-all whitespace-nowrap hover:bg-accent hover:text-accent-foreground hover:shadow-md"
    >
      <ScanFace className="h-4 w-4 shrink-0" />
      <span className="hidden sm:inline">Điểm danh</span>
      <span className="sm:hidden">ĐD</span>
    </button>
  );
}
