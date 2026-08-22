import { cn } from "@/lib/utils";

interface EmptyStateIllustrationProps {
  className?: string;
}

/** A single visual language for empty data, filter, and search states. */
export function EmptyStateIllustration({ className }: EmptyStateIllustrationProps) {
  return (
    <img
      src="/images/empty-states/payroll-empty-state-illustration.png"
      alt=""
      aria-hidden="true"
      className={cn("h-20 w-20 object-contain sm:h-24 sm:w-24", className)}
    />
  );
}
