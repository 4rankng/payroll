import { Button } from "@/components/ui/button";
import type { LucideIcon } from "lucide-react";

interface MobileOverflowActionProps {
  icon: LucideIcon;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  isLoading?: boolean;
}

export function MobileOverflowAction({
  icon: Icon,
  label,
  onClick,
  disabled = false,
  isLoading = false,
}: MobileOverflowActionProps) {
  return (
    <Button
      variant="ghost"
      className="w-full justify-start gap-3 h-12 px-2"
      onClick={onClick}
      disabled={disabled}
    >
      <Icon className={`h-5 w-5 text-muted-foreground shrink-0${isLoading ? " animate-spin" : ""}`} />
      <span className="text-sm font-medium">{label}</span>
    </Button>
  );
}

export function MobileOverflowDivider() {
  return <div className="h-px bg-border mx-2 my-1" />;
}
