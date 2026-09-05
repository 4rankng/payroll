import { memo } from "react";
import { formatCurrency } from "@/utils/formatters";
import { useCountUp } from "@/hooks/useCountUp";

/** Count-up currency readout shared by the treasury band panels. */
export const AnimatedCurrency = memo(function AnimatedCurrency({
  target,
}: {
  target: number;
}) {
  const animated = useCountUp(target, 700);
  return (
    <span className="font-financial tabular-nums tracking-normal">
      {formatCurrency(animated)}
    </span>
  );
});
