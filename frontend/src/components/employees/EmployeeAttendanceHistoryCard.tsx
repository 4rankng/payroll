import { format } from "date-fns";
import { AlertTriangle, Calendar, History, WalletCards } from "lucide-react";
import { formatDate } from "@/utils/formatters";
import { isSalaryRecorded, isSalaryMissing } from "@/utils/attendanceHelpers";
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
      <div className="p-4">
        <div className="mb-4 flex items-center gap-2.5">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-50 text-employee">
            <History className="h-4 w-4" />
          </div>
          <h3 className="text-[18px] font-bold leading-6 text-slate-900">
            Lịch sử chấm công
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
            <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50/70 px-4 py-8 text-center text-gray-400">
              <Calendar className="h-8 w-8 mx-auto mb-2 opacity-50" />
              <p className="text-[16px] font-semibold">Chưa có ca làm nào</p>
              <p className="mt-1 text-[15px] leading-5 text-slate-400">Sau khi bạn vào làm, ca sẽ hiện ở đây.</p>
            </div>
          ) : (
            history.slice(0, 5).map((att) => {
              const salaryRecorded = isSalaryRecorded(att);
              const salaryMissing = isSalaryMissing(att);

              return (
              <div
                key={att.id}
                className="rounded-xl border border-slate-200 bg-white p-3.5 shadow-sm"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <p className="text-[16px] font-bold leading-6 text-slate-900">
                      Ngày {formatDate(att.date)}
                    </p>
                    <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[15px] font-medium leading-5 text-slate-600">
                      <span>Vào {formatTime(att.check_in_time)}</span>
                      {att.check_out_time && (
                        <span>Tan {formatTime(att.check_out_time)}</span>
                      )}
                    </div>
                  </div>
                  <div className="shrink-0 text-right">
                    {att.status === "completed" ? (
                      <span className="rounded-lg bg-emerald-50 px-2.5 py-1.5 text-[15px] font-bold text-emerald-700">
                        Hoàn thành
                      </span>
                    ) : att.status === "orphaned" ? (
                      <span className="rounded-lg bg-red-50 px-2.5 py-1.5 text-[15px] font-bold text-red-700">
                        Thiếu tan ca
                      </span>
                    ) : (
                      <span className="rounded-lg bg-sky-50 px-2.5 py-1.5 text-[15px] font-bold text-sky-700">
                        Đang làm
                      </span>
                    )}
                  </div>
                </div>
                {salaryRecorded && (
                  <div className="mt-3 flex items-center gap-2 rounded-lg bg-emerald-50 px-3 py-2.5 text-[15px] font-bold text-emerald-700">
                    <WalletCards className="h-4 w-4" />
                    <span>Đã ghi lương +{att.earning_amount?.toLocaleString("vi-VN")}đ</span>
                  </div>
                )}
                {salaryMissing && (
                  <div className="mt-3 flex items-start gap-2 rounded-lg bg-amber-50 px-3 py-2.5 text-[15px] font-semibold leading-5 text-amber-800">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                    <span>{att.salary_reject_reason || att.salary_message || "Chưa ghi nhận lương cho ca này."}</span>
                  </div>
                )}
              </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
