import { memo } from "react";
import { cn } from "@/lib/utils";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { AdvancePaymentRequestStatus } from "@/types/api/advance-payment.types";

interface StatusOption {
  value: AdvancePaymentRequestStatus | "all";
  label: string;
  dotClass: string;
}

const STATUS_OPTIONS: StatusOption[] = [
  { value: "all",       label: "Tất cả",    dotClass: "" },
  { value: "pending",   label: "Chờ xử lý", dotClass: "bg-amber-400" },
  { value: "completed", label: "Hoàn tất",  dotClass: "bg-emerald-500" },
  { value: "failed",    label: "Thất bại",  dotClass: "bg-red-400" },
  { value: "cancelled", label: "Đã hủy",    dotClass: "bg-gray-400" },
];

export interface StatusCounts {
  all?: number;
  pending?: number;
  completed?: number;
  failed?: number;
  cancelled?: number;
}

interface StatusFilterBarProps {
  value: AdvancePaymentRequestStatus | "all";
  onChange: (value: string) => void;
  counts?: StatusCounts;
}

export const StatusFilterBar = memo(function StatusFilterBar({
  value,
  onChange,
  counts,
}: StatusFilterBarProps) {
  const selected = STATUS_OPTIONS.find((o) => o.value === value) ?? STATUS_OPTIONS[0];
  const selectedCount = counts?.[selected.value as keyof StatusCounts];

  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="h-8 text-sm shrink-0 w-auto min-w-36 gap-1.5">
        {selected.dotClass && (
          <span className={cn("h-1.5 w-1.5 rounded-full shrink-0", selected.dotClass)} />
        )}
        <SelectValue>
          {selected.label}
          {selectedCount !== undefined && selectedCount > 0 && (
            <span className="ml-1.5 inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full text-[10px] tabular-nums font-semibold bg-foreground/10">
              {selectedCount}
            </span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {STATUS_OPTIONS.map((opt) => {
          const count = counts?.[opt.value as keyof StatusCounts];
          return (
            <SelectItem key={opt.value} value={opt.value} className="text-sm">
              <div className="flex items-center gap-2">
                {opt.dotClass ? (
                  <span className={cn("h-1.5 w-1.5 rounded-full shrink-0", opt.dotClass)} />
                ) : (
                  <span className="h-1.5 w-1.5 shrink-0" />
                )}
                <span>{opt.label}</span>
                {count !== undefined && count > 0 && (
                  <span className="ml-auto pl-3 text-[11px] tabular-nums text-muted-foreground font-medium">
                    {count}
                  </span>
                )}
              </div>
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
});
