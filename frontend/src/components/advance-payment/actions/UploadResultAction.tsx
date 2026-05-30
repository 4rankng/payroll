import { ArrowRightLeft } from "lucide-react";

interface UploadResultActionProps {
  onClick: () => void;
}

export function UploadResultAction({ onClick }: UploadResultActionProps) {
  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1.5 h-9 px-4 bg-card/90 backdrop-blur-sm text-foreground text-sm font-medium whitespace-nowrap hover:bg-accent hover:text-accent-foreground transition-all border-l border-border"
    >
      <ArrowRightLeft className="h-4 w-4 shrink-0" />
      Nhập KQ
    </button>
  );
}
