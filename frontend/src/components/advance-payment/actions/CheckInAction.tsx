import { ScanFace } from "lucide-react";

interface CheckInActionProps {
  onClick: () => void;
}

export function CheckInAction({ onClick }: CheckInActionProps) {
  return (
    <button
      onClick={onClick}
      aria-label="Cài đặt điểm danh"
      title="Cài đặt điểm danh"
      className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border bg-card px-3 text-sm font-medium text-foreground transition-all whitespace-nowrap hover:bg-accent"
    >
      <ScanFace className="h-4 w-4 shrink-0" />
      <span className="hidden md:inline">Điểm danh</span>
      <span className="md:hidden">ĐD</span>
    </button>
  );
}
