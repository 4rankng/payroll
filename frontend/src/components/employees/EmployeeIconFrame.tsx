import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

type EmployeeIconFrameSize = "section" | "row";
type EmployeeIconFrameTone = "employee" | "slate";

interface EmployeeIconFrameProps {
  icon: LucideIcon;
  size?: EmployeeIconFrameSize;
  tone?: EmployeeIconFrameTone;
  className?: string;
}

const sizeClasses: Record<EmployeeIconFrameSize, string> = {
  section: "h-11 w-11 rounded-2xl [&_svg]:h-5.5 [&_svg]:w-5.5",
  row: "h-9 w-9 rounded-xl [&_svg]:h-4 [&_svg]:w-4",
};

const toneClasses: Record<EmployeeIconFrameTone, string> = {
  employee: "bg-employee/10 text-employee",
  slate: "bg-slate-50 text-slate-500",
};

export function EmployeeIconFrame({
  icon: Icon,
  size = "section",
  tone = "employee",
  className,
}: EmployeeIconFrameProps) {
  return (
    <div
      className={cn(
        "flex shrink-0 items-center justify-center",
        sizeClasses[size],
        toneClasses[tone],
        className
      )}
      aria-hidden="true"
    >
      <Icon strokeWidth={2.2} />
    </div>
  );
}
