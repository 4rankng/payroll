import { useState, memo, useCallback, useEffect, useMemo } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Calendar } from '@/components/ui/calendar';
import { Download, Loader2 } from 'lucide-react';
import { vi } from 'date-fns/locale';
import { formatDateForAPI } from '@/utils/formatters';
import type { Formatters } from 'react-day-picker';

export interface PayrollReportExportParams {
  atDate: string;
}

interface PayrollReportExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onExport: (params: PayrollReportExportParams) => void;
  isLoading?: boolean;
}

export const PayrollReportExportDialog = memo(function PayrollReportExportDialog({
  open,
  onOpenChange,
  onExport,
  isLoading = false
}: PayrollReportExportDialogProps) {
  const [reportDate, setReportDate] = useState<Date>();

  useEffect(() => {
    if (open) {
      setReportDate(new Date());
    } else {
      setReportDate(undefined);
    }
  }, [open]);

  const handleReportDateSelect = useCallback((date: Date | undefined) => {
    if (date) {
      setReportDate(date);
    }
  }, []);

  const formatters: Partial<Formatters> = useMemo(
    () => ({
      formatCaption: (date) => {
        return `Tháng ${date.getMonth() + 1}, ${date.getFullYear()}`;
      },
    }),
    []
  );

  const handleDialogClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const canExport = Boolean(reportDate);

  const handleExport = useCallback(() => {
    if (!reportDate) return;

    const params: PayrollReportExportParams = {
      atDate: formatDateForAPI(reportDate),
    };

    onExport(params);
  }, [reportDate, onExport]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-full sm:max-w-md"
        contentPadding="none"
        description="Chọn ngày để tải báo cáo sao kê"
      >
        <DialogHeader className="mx-0 mt-0 px-6 pr-16 pt-5 pb-4 sm:mx-0 sm:mt-0 sm:pr-20">
          <DialogTitle>Xuất sao kê thanh toán</DialogTitle>
        </DialogHeader>

        <div className="flex justify-center px-1 py-4 sm:px-6">
          <Calendar
            mode="single"
            selected={reportDate}
            onSelect={handleReportDateSelect}
            initialFocus
            locale={vi}
            formatters={formatters}
            className="rounded-xl p-0 sm:p-3"
          />
        </div>

        <DialogFooter className="flex flex-row gap-3 sm:gap-4 px-6 pb-6">
          <Button
            variant="outline"
            onClick={handleDialogClose}
            disabled={isLoading}
            className="w-full h-12 min-h-[44px] order-2 sm:order-1"
          >
            Đóng
          </Button>
          <Button
            onClick={handleExport}
            disabled={!canExport || isLoading}
            className="w-full h-12 min-h-[44px] order-1 sm:order-2"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang xuất...
              </>
            ) : (
              <>
                <Download className="w-4 h-4 mr-2" />
                Xuất sao kê
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
