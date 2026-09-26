import { ArrowRightLeft } from "lucide-react";

interface UploadResultActionProps {
  onClick: () => void;
}

export function UploadResultAction({ onClick }: UploadResultActionProps) {
  return (
    <button
      onClick={onClick}
      aria-label="Nhập kết quả thanh toán từ ngân hàng"
      title="Nhập kết quả thanh toán từ ngân hàng"
      className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border bg-card px-3 text-sm font-medium text-foreground transition-all whitespace-nowrap hover:bg-accent"
    >
      <ArrowRightLeft className="h-4 w-4 shrink-0" />
      <span className="hidden md:inline">Nhập kết quả</span>
      <span className="md:hidden">Nhập KQ</span>
    </button>
  );
}
