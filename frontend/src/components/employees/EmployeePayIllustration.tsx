import { cn } from "@/lib/utils";

interface EmployeePayIllustrationProps {
  className?: string;
  size?: "sm" | "md";
}

export function EmployeePayIllustration({
  className,
  size = "md",
}: EmployeePayIllustrationProps) {
  const dimensions = size === "sm" ? "h-11 w-11" : "h-12 w-12";

  return (
    <div
      className={cn(
        "flex shrink-0 items-center justify-center rounded-2xl bg-employee/10 text-employee",
        dimensions,
        "employee-pay-illustration",
        className
      )}
      aria-hidden="true"
    >
      <svg
        viewBox="0 0 56 56"
        fill="none"
        className="h-full w-full overflow-visible"
      >
        <path
          d="M16 19.5h23.5a5 5 0 0 1 5 5v13a5 5 0 0 1-5 5H16a5 5 0 0 1-5-5v-18Z"
          fill="currentColor"
          opacity="0.16"
        />
        <path
          d="M12 20.5h27.5a5 5 0 0 1 5 5v11.5a5 5 0 0 1-5 5H16.5a5 5 0 0 1-5-5V18.5a4.5 4.5 0 0 1 4.5-4.5h21"
          stroke="currentColor"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M38 27.5h8v8h-8a4 4 0 0 1 0-8Z"
          fill="white"
          stroke="currentColor"
          strokeWidth="3"
          strokeLinejoin="round"
        />
        <circle
          cx="39.5"
          cy="31.5"
          r="1.6"
          fill="currentColor"
          className="motion-safe:animate-pulse"
        />
        <g className="origin-center motion-safe:animate-pulse">
          <circle cx="21" cy="17" r="7" fill="white" />
          <circle cx="21" cy="17" r="7" fill="currentColor" opacity="0.16" />
          <path
            d="M18.2 17.2 20 19l3.8-4.2"
            stroke="currentColor"
            strokeWidth="2.4"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </g>
      </svg>
    </div>
  );
}
