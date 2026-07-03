import { FileSpreadsheet } from "lucide-react";

interface ImportActionProps {
  onClick: () => void;
}

export function ImportAction({ onClick }: ImportActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex min-h-11 items-center gap-1.5 bg-card/90 px-4 text-sm font-medium text-foreground backdrop-blur-sm transition-all whitespace-nowrap hover:bg-accent hover:text-accent-foreground"
    >
      <FileSpreadsheet className="h-4 w-4 shrink-0" />
      <span className="hidden sm:inline">Nhập bảng lương</span>
      <span className="sm:hidden">Nhập BL</span>
    </button>
  );
}
