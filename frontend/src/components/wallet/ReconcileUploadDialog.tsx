import { useEffect, useMemo, useRef, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogNavyHeader,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Loader2,
  Upload,
  FileCheck,
  FileX2,
  Download,
  Calendar,
} from "lucide-react";
import { walletService } from "@/services/api/wallet.service";
import type { ReconciliationJob } from "@/types/api/wallet.types";
import { showErrorNotification, showSuccessNotification } from "@/utils/error-handler";
import { DateRangePicker } from "@/components/ui/date-range-picker";

const MAX_RANGE_DAYS = 30;

const toIsoDate = (d: Date): string => {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
};

const toDdMmYyyy = (iso: string): string => {
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
};

const defaultRange = (): { start: string; end: string } => {
  const today = new Date();
  const sevenDaysAgo = new Date(today);
  sevenDaysAgo.setDate(today.getDate() - 6);
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  return { start: toIsoDate(sevenDaysAgo), end: toIsoDate(yesterday) };
};

interface ReconcileUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCompleted?: () => void;
}

export default function ReconcileUploadDialog({
  open,
  onOpenChange,
  onCompleted,
}: ReconcileUploadDialogProps) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [tab, setTab] = useState<"auto" | "manual">("auto");
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [autoRunning, setAutoRunning] = useState(false);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [job, setJob] = useState<ReconciliationJob | null>(null);
  const [polling, setPolling] = useState(false);

  useEffect(() => {
    if (open) {
      setTab("auto");
      setFile(null);
      setJob(null);
      setUploading(false);
      setAutoRunning(false);
      setPolling(false);
      const { start, end } = defaultRange();
      setStartDate(start);
      setEndDate(end);
    }
  }, [open]);

  const dateError = useMemo<string | null>(() => {
    if (!startDate || !endDate) return "Vui lòng chọn khoảng ngày";
    if (startDate > endDate) return "Từ ngày phải nhỏ hơn hoặc bằng đến ngày";
    const fromMs = new Date(startDate + "T00:00:00").getTime();
    const toMs = new Date(endDate + "T00:00:00").getTime();
    const days = Math.round((toMs - fromMs) / (24 * 60 * 60 * 1000)) + 1;
    if (days > MAX_RANGE_DAYS) return `Khoảng ngày tối đa là ${MAX_RANGE_DAYS} ngày`;
    return null;
  }, [startDate, endDate]);

  const startPolling = (jobId: string) => {
    setPolling(true);
    const poll = async () => {
      try {
        const j = await walletService.getReconciliationJobStatus(jobId);
        setJob(j);
        if (j.status === "processing") {
          setTimeout(poll, 2000);
        } else {
          setPolling(false);
          onCompleted?.();
          if (j.status === "completed") {
            showSuccessNotification(
              `Đối soát hoàn tất: ${j.matched} khớp, ${j.unmatched} không khớp`,
            );
          }
        }
      } catch (err) {
        setPolling(false);
        showErrorNotification(err, "Lỗi truy vấn trạng thái đối soát");
      }
    };
    poll();
  };

  const handleAutoConfirm = async () => {
    if (dateError) return;
    setAutoRunning(true);
    setJob(null);
    try {
      const { job_id } = await walletService.autoReconcile(
        toDdMmYyyy(startDate),
        toDdMmYyyy(endDate),
      );
      setAutoRunning(false);
      if (!job_id) {
        showSuccessNotification("Đối soát hoàn tất. Vui lòng làm mới để xem kết quả.");
        onCompleted?.();
        return;
      }
      startPolling(job_id);
    } catch (err) {
      showErrorNotification(err, "Tải dữ liệu đối soát thất bại");
      setAutoRunning(false);
    }
  };

  const handleUpload = async () => {
    if (!file) return;
    setUploading(true);
    setJob(null);
    try {
      const { job_id } = await walletService.uploadReconciliation(file);
      startPolling(job_id);
    } catch (err) {
      showErrorNotification(err, "Tải file đối soát thất bại");
    } finally {
      setUploading(false);
    }
  };

  const busy = uploading || autoRunning || polling;
  const isTerminal = job?.status === "completed" || job?.status === "failed";

  const today = new Date();
  today.setHours(23, 59, 59, 999);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-md gap-0 overflow-hidden"
        contentPadding="none"
        onOpenAutoFocus={(e) => e.preventDefault()}
        onPointerDownOutside={(e) => {
          const target = e.target as HTMLElement | null;
          if (target?.closest?.("#datepicker-portal")) e.preventDefault();
        }}
        onInteractOutside={(e) => {
          const target = e.target as HTMLElement | null;
          if (target?.closest?.("#datepicker-portal")) e.preventDefault();
        }}
        hideCloseButton
      >
        <DialogNavyHeader
          title="Đối soát giao dịch"
          description="Chọn phương thức để lấy dữ liệu đối soát."
        />

        {/* Tab Switcher */}
        <div className="px-4 pt-4 pb-2">
          <div className="flex p-1 bg-slate-100 rounded-xl">
            <button
              onClick={() => !busy && setTab("auto")}
              className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-semibold rounded-lg transition-all ${
                tab === "auto"
                  ? "bg-white text-slate-900 shadow-sm"
                  : "text-slate-500 hover:text-slate-700"
              }`}
            >
              <Download size={13} />
              Tải dữ liệu
            </button>
            <button
              onClick={() => !busy && setTab("manual")}
              className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-semibold rounded-lg transition-all ${
                tab === "manual"
                  ? "bg-white text-slate-900 shadow-sm"
                  : "text-slate-500 hover:text-slate-700"
              }`}
            >
              <Upload size={13} />
              Tải lên file CSV
            </button>
          </div>
        </div>

        {/* Tab Content */}
        <div className="px-4 pb-4">
          {tab === "auto" ? (
            <div>
              <label className="text-[10px] font-bold uppercase tracking-tight text-slate-400 flex items-center gap-1 mb-1.5">
                <Calendar size={10} className="text-slate-400" />
                Khoảng thời gian
              </label>
              <div className="rounded-md border border-slate-200 bg-white px-2.5 py-1.5">
                <DateRangePicker
                  variant="default"
                  startDate={startDate}
                  endDate={endDate}
                  onStartDateChange={setStartDate}
                  onEndDateChange={setEndDate}
                  maxDate={today}
                  disabled={autoRunning}
                  usePortal
                />
              </div>
              {dateError && (
                <p className="text-xs text-destructive mt-1.5">{dateError}</p>
              )}
            </div>
          ) : (
            <div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,text/csv"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (f) {
                    setFile(f);
                    setJob(null);
                  }
                }}
                disabled={busy}
              />

              {file ? (
                <div className="flex items-center gap-2.5 rounded border border-slate-200 bg-white p-2.5">
                  <FileCheck size={14} className="text-emerald-600 shrink-0" />
                  <span className="text-xs text-slate-600 truncate flex-1 font-medium">
                    {file.name}
                  </span>
                  <button
                    type="button"
                    onClick={() => setFile(null)}
                    className="text-slate-400 hover:text-slate-600 transition-colors"
                  >
                    &times;
                  </button>
                </div>
              ) : (
                <div
                  onClick={() => fileInputRef.current?.click()}
                  className="border border-dashed border-slate-300 rounded bg-white py-4 flex flex-col items-center justify-center gap-1.5 hover:border-[#3b82f6] transition-colors cursor-pointer"
                >
                  <div className="w-7 h-7 bg-slate-50 text-slate-400 rounded-full flex items-center justify-center">
                    <Upload size={14} />
                  </div>
                  <div className="text-center">
                    <span className="text-xs font-bold text-[#3b82f6] block">
                      Nhấn để chọn file
                    </span>
                    <p className="text-[10px] text-slate-400">CSV (Max 10MB)</p>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Job result */}
        {(polling || job) && (
          <div className="mx-4 mb-4 rounded-xl border bg-slate-50 p-4 space-y-2">
            <div className="flex items-center gap-2">
              {polling && !isTerminal ? (
                <>
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-slate-500" />
                  <span className="text-xs text-slate-700">Đang xử lý...</span>
                </>
              ) : job?.status === "completed" ? (
                <>
                  <FileCheck className="h-3.5 w-3.5 text-emerald-600" />
                  <span className="text-xs font-medium text-emerald-700">
                    Đối soát hoàn tất
                  </span>
                </>
              ) : job?.status === "failed" ? (
                <>
                  <FileX2 className="h-3.5 w-3.5 text-rose-600" />
                  <span className="text-xs font-medium text-rose-700">
                    Đối soát thất bại
                  </span>
                </>
              ) : null}
            </div>

            {job && (
              <div className="grid grid-cols-3 gap-3 text-xs">
                <div>
                  <p className="text-[10px] text-slate-500">Tổng dòng</p>
                  <p className="font-semibold tabular-nums">{job.total_rows}</p>
                </div>
                <div>
                  <p className="text-[10px] text-slate-500">Khớp</p>
                  <p className="font-semibold text-emerald-700 tabular-nums">
                    {job.matched}
                  </p>
                </div>
                <div>
                  <p className="text-[10px] text-slate-500">Không khớp</p>
                  <p className="font-semibold text-rose-700 tabular-nums">
                    {job.unmatched}
                  </p>
                </div>
              </div>
            )}

            {job?.status === "completed" && (job?.unmatched ?? 0) > 0 && job?.raw_rows?.length && (
              <button
                onClick={() => walletService.downloadBreakTransactionsCsv(job)}
                className="flex items-center gap-1.5 text-xs font-semibold text-[#3b82f6] hover:text-[#2563eb] transition-colors mt-1"
              >
                <Download size={12} />
                Tải giao dịch không khớp
              </button>
            )}
          </div>
        )}

        {/* Footer */}
        <DialogFooter className="px-4 py-3 border-t flex justify-between">
          <Button
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={busy}
            className="px-3 py-1 h-auto text-xs font-bold text-slate-500 hover:text-slate-700 uppercase tracking-tight"
          >
            {isTerminal ? "Đóng" : "Hủy"}
          </Button>

          <Button
            onClick={tab === "auto" ? handleAutoConfirm : handleUpload}
            disabled={tab === "auto" ? !!dateError || busy : !file || busy}
            onMouseDown={(e) => e.stopPropagation()}
            className="px-3.5 py-1 h-auto text-xs font-bold bg-[#2a3b58] hover:bg-[#1e293b] text-white uppercase tracking-tight shadow-sm flex items-center gap-1.5"
          >
            {busy ? (
              <Loader2 size={12} className="animate-spin" />
            ) : tab === "auto" ? (
              <Download size={12} />
            ) : (
              <Upload size={12} />
            )}
            {tab === "auto" ? "Đối soát" : "Tải lên"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
