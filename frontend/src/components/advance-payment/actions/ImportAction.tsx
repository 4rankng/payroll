import { FileSpreadsheet } from "lucide-react";

interface ImportActionProps {
  onClick: () => void;
}

/** Primary page action - the only solid button in the toolbar. */
export function ImportAction({ onClick }: ImportActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-primary px-3 text-sm font-medium text-primary-foreground transition-all whitespace-nowrap hover:bg-primary/90"
    >
      <FileSpreadsheet className="h-4 w-4 shrink-0" />
      <span className="hidden sm:inline">Nhập bảng lương</span>
      <span className="sm:hidden">Nhập BL</span>
    </button>
  );
}
