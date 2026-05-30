import { Skeleton } from "@/components/ui/skeleton";
import { useSlowestEndpoints } from "@/hooks/api/useSystemHealth";
import { EndpointLatencyCard } from "./EndpointLatencyCard";
import { AllClear } from "./AllClear";

interface Props {
  days?: number;
  limit?: number;
}

export function LatencyGrid({ days = 7, limit = 12 }: Props) {
  const { data: slowest, isLoading } = useSlowestEndpoints(days);

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-20 rounded-xl w-full" />
        ))}
      </div>
    );
  }

  if (!slowest || slowest.length === 0) {
    return <AllClear text="Không có dữ liệu độ trễ" />;
  }

  return (
    <div className="columns-1 sm:columns-2 gap-2">
      {slowest.slice(0, limit).map((ep) => (
        <div key={ep.endpoint} className="break-inside-avoid mb-2">
          <EndpointLatencyCard endpoint={ep} days={days} />
        </div>
      ))}
    </div>
  );
}
