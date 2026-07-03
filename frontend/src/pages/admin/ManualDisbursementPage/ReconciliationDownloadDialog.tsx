import { useEffect, useMemo, useState } from "react";
import { Loader2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { DateRangePicker } from "@/components/ui/date-range-picker";
import { manualDisbursementService } from "@/services/api/manual-disbursement.service";
import { toast } from "@/components/ui/sonner";
import { getErrorMessage } from "@/utils/error-handler";

const MAX_RANGE_DAYS = 31;

const toIsoDate = (d: Date): string => {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
};

const toDdMmYyyy = (iso: string): string => {
  // iso is YYYY-MM-DD; convert to DD/MM/YYYY for backend.
  const [y, m, d] = iso.split("-");
  return `${d}/${m}/${y}`;
};

const defaultRange = (): { from: string; to: string } => {
  const today = new Date();
  const sevenDaysAgo = new Date(today);
  sevenDaysAgo.setDate(today.getDate() - 6); // last 7 days inclusive
  return { from: toIsoDate(sevenDaysAgo), to: toIsoDate(today) };
};

interface Props {
  open: boolean;
  onClose: () => void;
}

export function ReconciliationDownloadDialog({ open, onClose }: Props) {
  const [from, setFrom] = useState<string>("");
  const [to, setTo] = useState<string>("");
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (open) {
      const { from: f, to: t } = defaultRange();
      setFrom(f);
      setTo(t);
      setPending(false);
    }
  }, [open]);

  const error = useMemo<string | null>(() => {
    if (!from || !to) return "Vui lòng chọn khoảng ngày";
    if (from > to) return "Từ ngày phải nhỏ hơn hoặc bằng đến ngày";
    const fromMs = new Date(from + "T00:00:00").getTime();
    const toMs = new Date(to + "T00:00:00").getTime();
    const days = Math.round((toMs - fromMs) / (24 * 60 * 60 * 1000)) + 1;
    if (days > MAX_RANGE_DAYS) return `Khoảng ngày tối đa là ${MAX_RANGE_DAYS} ngày`;
    return null;
  }, [from, to]);

  const handleDownload = async () => {
    if (error || pending) return;
    setPending(true);
    try {
      await manualDisbursementService.downloadReconciliation(
        toDdMmYyyy(from),
        toDdMmYyyy(to),
      );
      toast({
        title: "Tải báo cáo thành công",
        description: "File đối soát đã được tải về máy.",
      });
      onClose();
    } catch (err) {
      toast({
        variant: "destructive",
        title: "Không thể tải báo cáo đối soát",
        description: getErrorMessage(err),
      });
    } finally {
      setPending(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o && !pending) onClose(); }}>
      <DialogContent className="max-h-[92dvh] w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] overflow-y-auto sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Tải file đối soát</DialogTitle>
          <DialogDescription>
            Chọn khoảng ngày để tải báo cáo giao dịch chi hộ. Quá trình
            có thể mất vài giây.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="rounded-md border bg-muted/30 px-3 py-2">
            <DateRangePicker
              variant="mobile"
              startDate={from}
              endDate={to}
              onStartDateChange={setFrom}
              onEndDateChange={setTo}
              maxDate={new Date()}
              disabled={pending}
              usePortal
            />
          </div>
          {error && !pending && (
            <p className="text-xs text-destructive">{error}</p>
          )}
          {pending && (
            <p className="text-xs text-muted-foreground flex items-center gap-2">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Đang tải báo cáo...
            </p>
          )}
        </div>

        <DialogFooter className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={pending}
            className="min-h-11 w-full sm:w-auto"
          >
            Hủy
          </Button>
          <Button
            type="button"
            onClick={handleDownload}
            disabled={!!error || pending}
            className="min-h-11 w-full sm:w-auto"
          >
            {pending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Đang tải...
              </>
            ) : (
              "Tải xuống"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
