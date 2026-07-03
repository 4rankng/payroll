import { ArrowRightLeft } from "lucide-react";

interface UploadResultActionProps {
  onClick: () => void;
}

export function UploadResultAction({ onClick }: UploadResultActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex min-h-11 items-center gap-1.5 border-l border-border bg-card/90 px-4 text-sm font-medium text-foreground backdrop-blur-sm transition-all whitespace-nowrap hover:bg-accent hover:text-accent-foreground"
    >
      <ArrowRightLeft className="h-4 w-4 shrink-0" />
      Nhập KQ
    </button>
  );
}
