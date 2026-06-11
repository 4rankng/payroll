import { format } from "date-fns";
import { History, Calendar } from "lucide-react";
import { formatDate } from "@/utils/formatters";
import { useAttendanceHistory } from "@/hooks/api/useAttendance";
import { Skeleton } from "@/components/ui/skeleton";

function formatTime(time: string | undefined | null, fallback = "--:--"): string {
  if (!time) return fallback;
  try {
    return format(new Date(time), "HH:mm");
  } catch {
    return fallback;
  }
}

interface EmployeeAttendanceHistoryCardProps {
  className?: string;
  style?: React.CSSProperties;
}

export function EmployeeAttendanceHistoryCard({ className, style }: EmployeeAttendanceHistoryCardProps) {
  const { data: historyResponse, isLoading } = useAttendanceHistory();
  const history = historyResponse?.data || [];

  return (
    <div className={className ?? "bg-white rounded-2xl p-4"} style={style}>
      <div className="flex items-center gap-1.5 mb-4">
        <History className="h-4 w-4 text-employee" />
        <h3 className="font-semibold text-gray-800 uppercase tracking-wide text-sm">
          Lịch sử chấm công gần đây
        </h3>
      </div>

      <div className="space-y-3">
        {isLoading ? (
          Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="flex gap-3 items-center">
              <Skeleton className="h-10 w-10 rounded-full" />
              <div className="space-y-2 flex-1">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-24" />
              </div>
            </div>
          ))
        ) : history.length === 0 ? (
          <div className="text-center py-6 text-gray-400">
            <Calendar className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">Chưa có dữ liệu chấm công</p>
          </div>
        ) : (
          history.slice(0, 5).map((att) => (
            <div
              key={att.id}
              className="flex justify-between items-center p-3 rounded-xl bg-gray-50 border border-gray-100"
            >
              <div className="flex-1">
                <p className="font-medium text-sm text-gray-800">
                  Ngày {formatDate(att.date)}
                </p>
                <div className="text-xs text-gray-500 mt-1 flex gap-2">
                  <span>Vào: {formatTime(att.check_in_time)}</span>
                  {att.check_out_time && (
                    <span>Ra: {formatTime(att.check_out_time)}</span>
                  )}
                </div>
              </div>
              <div className="text-right">
                {att.status === "completed" ? (
                  <span className="text-xs font-medium text-green-600 bg-green-50 px-2 py-1 rounded-full">
                    Hoàn thành
                  </span>
                ) : att.status === "orphaned" ? (
                  <span className="text-xs font-medium text-red-600 bg-red-50 px-2 py-1 rounded-full">
                    Thiếu ra
                  </span>
                ) : (
                  <span className="text-xs font-medium text-blue-600 bg-blue-50 px-2 py-1 rounded-full">
                    Đang làm
                  </span>
                )}
                {att.earning_amount != null && att.earning_amount > 0 && (
                  <p className="text-xs font-semibold text-gray-700 mt-1.5">
                    +{att.earning_amount.toLocaleString("vi-VN")}đ
                  </p>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
