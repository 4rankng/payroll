import { cn } from "@/lib/utils";

interface EmployeePayIllustrationProps {
  className?: string;
  size?: "sm" | "md";
}

export function EmployeePayIllustration({
  className,
  size = "md",
}: EmployeePayIllustrationProps) {
  const dimensions = size === "sm" ? "h-11 w-11" : "h-[72px] w-[72px]";

  return (
    <span
      className={cn(
        "employee-pay-art flex shrink-0 items-center justify-center",
        dimensions,
        className
      )}
      aria-hidden="true"
    >
      <img
        src="/employee-pay-wallet.png"
        alt=""
        width={64}
        height={64}
        className="h-full w-full object-contain"
        decoding="async"
      />
    </span>
  );
}
