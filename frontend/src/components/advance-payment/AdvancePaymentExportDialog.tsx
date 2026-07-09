import { useState, memo, useCallback, useMemo, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Download, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatMonthDisplay } from '@/utils/advancePaymentHelpers';
import { apiClient } from '@/services/api/client';
import { API_ENDPOINTS } from '@/config/api.config';
import { showErrorNotification } from '@/utils/error-handler';

interface AdvancePaymentExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function generateMonthOptions(count: number) {
  const options: { value: string; label: string }[] = [];
  const now = new Date();
  for (let i = 0; i < count; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
    options.push({ value, label: formatMonthDisplay(value) });
  }
  return options;
}

function getDefaultMonth(monthOptions: { value: string }[]) {
  const currentDay = new Date().getDate();
  const defaultIndex = currentDay <= 8 ? 1 : 0;
  return monthOptions[defaultIndex]?.value || '';
}

export const AdvancePaymentExportDialog = memo(function AdvancePaymentExportDialog({
  open,
  onOpenChange,
}: AdvancePaymentExportDialogProps) {
  const monthOptions = useMemo(() => generateMonthOptions(12), []);
  const [selectedMonth, setSelectedMonth] = useState('');
  const [isDownloading, setIsDownloading] = useState(false);

  useEffect(() => {
    if (open) {
      setSelectedMonth(getDefaultMonth(monthOptions));
    } else {
      setSelectedMonth('');
    }
  }, [open, monthOptions]);

  const handleClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  const handleExport = useCallback(async () => {
    if (!selectedMonth) return;
    setIsDownloading(true);
    try {
      await apiClient.download(
        `${API_ENDPOINTS.advancePayments.reconciliation.export}?forMonth=${selectedMonth}`,
        `sao_ke_ung_luong_${selectedMonth}.xlsx`,
      );
    } catch (err) {
      showErrorNotification(err);
    } finally {
      setIsDownloading(false);
    }
  }, [selectedMonth]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-[95vw] sm:max-w-sm"
        contentPadding="none"
        title="Xuất sao kê ứng lương"
        description="Chọn tháng để tải xuống sao kê ứng lương"
      >
        <DialogHeader className="mx-0 mt-0 px-6 pt-5 pb-4">
          <DialogTitle>Xuất sao kê ứng lương</DialogTitle>
        </DialogHeader>

        <div className="px-6 pb-4 space-y-3">
          <p className="text-sm text-muted-foreground">Chọn tháng cần xuất sao kê</p>
          <div className="flex flex-wrap gap-2">
            {monthOptions.slice(0, 6).map((month) => (
              <button
                key={month.value}
                type="button"
                onClick={() => setSelectedMonth(month.value)}
                className={cn(
                  'h-8 px-3 rounded-xl text-sm font-medium border transition-colors',
                  selectedMonth === month.value
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'bg-background text-foreground border-border hover:bg-muted',
                )}
              >
                {month.label}
              </button>
            ))}
          </div>
        </div>

        <DialogFooter className="flex flex-row gap-3 sm:gap-4 px-6 pb-6">
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={isDownloading}
            className="w-full h-12 min-h-[44px] order-2 sm:order-1"
          >
            Đóng
          </Button>
          <Button
            onClick={handleExport}
            disabled={!selectedMonth || isDownloading}
            className="w-full h-12 min-h-[44px] order-1 sm:order-2"
          >
            {isDownloading ? (
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
