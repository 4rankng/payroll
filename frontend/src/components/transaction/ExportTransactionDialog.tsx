import { useState, useMemo } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { Download, X, Calendar } from 'lucide-react';
import { dateToString } from '@/utils/dateHelpers';
import { useMetadata } from '@/contexts';
import type { TransactionType } from '@/services/api/transaction.service';

interface ExportTransactionDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onExport: (fromDate: string, toDate: string, transactionType?: TransactionType) => void;
  isExporting?: boolean;
}

export function ExportTransactionDialog({
  isOpen,
  onClose,
  onExport,
  isExporting = false,
}: ExportTransactionDialogProps) {
  const { transactionMetadata, isLoadingTransactionMetadata } = useMetadata();
  // Get default date range (last 3 months)
  const defaultDateRange = useMemo(() => {
    const now = new Date();
    const threeMonthsAgo = new Date(now.getFullYear(), now.getMonth() - 3, 1);
    const endOfCurrentMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0);
    return {
      fromDate: dateToString(threeMonthsAgo),
      toDate: dateToString(endOfCurrentMonth),
    };
  }, []);

  const [fromDate, setFromDate] = useState(defaultDateRange.fromDate);
  const [toDate, setToDate] = useState(defaultDateRange.toDate);
  const [transactionType, setTransactionType] = useState<string>('all');

  const handleExport = () => {
    if (!fromDate || !toDate) {
      return;
    }

    const type = transactionType === 'all' ? undefined : (transactionType as TransactionType);
    onExport(fromDate, toDate, type);
  };

  const handleClose = () => {
    if (!isExporting) {
      onClose();
    }
  };

  // Date presets
  const handlePresetClick = (preset: string) => {
    const now = new Date();
    let newFromDate = '';
    let newToDate = '';

    switch (preset) {
      case 'this-month':
        newFromDate = dateToString(new Date(now.getFullYear(), now.getMonth(), 1));
        newToDate = dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0));
        break;
      case 'last-month':
        newFromDate = dateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1));
        newToDate = dateToString(new Date(now.getFullYear(), now.getMonth(), 0));
        break;
      case 'this-quarter':
        const currentQuarter = Math.floor(now.getMonth() / 3);
        newFromDate = dateToString(new Date(now.getFullYear(), currentQuarter * 3, 1));
        newToDate = dateToString(new Date(now.getFullYear(), currentQuarter * 3 + 3, 0));
        break;
      case 'this-year':
        newFromDate = dateToString(new Date(now.getFullYear(), 0, 1));
        newToDate = dateToString(new Date(now.getFullYear(), 11, 31));
        break;
      case 'last-3-months':
        newFromDate = dateToString(new Date(now.getFullYear(), now.getMonth() - 3, 1));
        newToDate = dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0));
        break;
    }

    if (newFromDate && newToDate) {
      setFromDate(newFromDate);
      setToDate(newToDate);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Xuất danh sách giao dịch
          </DialogTitle>
          <DialogDescription>
            Chọn khoảng thời gian và loại giao dịch để xuất
          </DialogDescription>
        </DialogHeader>

        <div className="flex gap-8 py-4">
          {/* Left Pane - Date Presets */}
          <div className="space-y-3 flex-1">
            <Label className="text-sm font-medium">Khoảng thời gian nhanh</Label>
            <div className="flex flex-col gap-2">
              <Button
                type="button"
                variant="outline"
                size="default"
                onClick={() => handlePresetClick('this-month')}
                className="justify-start w-full"
              >
                <Calendar className="h-4 w-4 mr-2" />
                Tháng này
              </Button>
              <Button
                type="button"
                variant="outline"
                size="default"
                onClick={() => handlePresetClick('last-month')}
                className="justify-start w-full"
              >
                <Calendar className="h-4 w-4 mr-2" />
                Tháng trước
              </Button>
              <Button
                type="button"
                variant="outline"
                size="default"
                onClick={() => handlePresetClick('this-quarter')}
                className="justify-start w-full"
              >
                <Calendar className="h-4 w-4 mr-2" />
                Quý này
              </Button>
              <Button
                type="button"
                variant="outline"
                size="default"
                onClick={() => handlePresetClick('last-3-months')}
                className="justify-start w-full"
              >
                <Calendar className="h-4 w-4 mr-2" />
                3 tháng gần đây
              </Button>
              <Button
                type="button"
                variant="outline"
                size="default"
                onClick={() => handlePresetClick('this-year')}
                className="justify-start w-full"
              >
                <Calendar className="h-4 w-4 mr-2" />
                Năm này
              </Button>
            </div>
          </div>

          {/* Right Pane - Custom Date Range & Transaction Type */}
          <div className="flex-1 space-y-3">
            <Label className="text-sm font-medium">Hoặc chọn tùy chỉnh</Label>
            <div className="space-y-3">
              <div className="space-y-2">
                <Label htmlFor="fromDate" className="text-xs text-muted-foreground">
                  Từ ngày *
                </Label>
                <Input
                  id="fromDate"
                  type="date"
                  value={fromDate}
                  onChange={(e) => setFromDate(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="toDate" className="text-xs text-muted-foreground">
                  Đến ngày *
                </Label>
                <Input
                  id="toDate"
                  type="date"
                  value={toDate}
                  onChange={(e) => setToDate(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="transactionType" className="text-xs text-muted-foreground">
                  Loại giao dịch
                </Label>
                {isLoadingTransactionMetadata ? (
                  <Skeleton className="w-full h-10" />
                ) : (
                  <SearchableSelect
                    triggerId="transactionType"
                    value={transactionType}
                    onChange={setTransactionType}
                    searchPlaceholder="Tìm loại giao dịch..."
                    options={[
                      { value: 'all', label: 'Tất cả' },
                      ...(transactionMetadata?.transaction_types.map((type) => ({
                        value: type.type,
                        label: type.label,
                      })) ?? []),
                    ]}
                  />
                )}
              </div>
            </div>
          </div>
        </div>

        <DialogFooter className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <Button
            type="button"
            variant="outline"
            onClick={handleClose}
            disabled={isExporting}
            className="min-h-11 w-full"
          >
            <X className="h-4 w-4 mr-2" />
            Hủy
          </Button>
          <Button
            type="button"
            onClick={handleExport}
            disabled={isExporting || !fromDate || !toDate}
            className="min-h-11 w-full"
          >
            <Download className="h-4 w-4 mr-2" />
            {isExporting ? 'Đang xuất...' : 'Xuất file'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
