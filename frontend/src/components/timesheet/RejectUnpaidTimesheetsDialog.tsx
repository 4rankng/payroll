import { useEffect, useRef } from 'react';
import { AlertTriangle, ArrowLeft, Loader2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { SearchableDropdown } from '@/components/ui/searchable-dropdown';
import { Textarea } from '@/components/ui/textarea';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useRejectUnpaidTimesheetsDialog } from './useRejectUnpaidTimesheetsDialog';

interface RejectUnpaidTimesheetsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialProjectId?: number;
}

export function RejectUnpaidTimesheetsDialog({
  open,
  onOpenChange,
  initialProjectId,
}: RejectUnpaidTimesheetsDialogProps) {
  const isMobile = useIsMobile();
  const form = useRejectUnpaidTimesheetsDialog({
    open,
    initialProjectId,
    onSuccess: () => onOpenChange(false),
  });
  const confirmationHeadingRef = useRef<HTMLHeadingElement>(null);
  const reasonRef = useRef<HTMLTextAreaElement>(null);
  const previousStepRef = useRef(form.step);

  useEffect(() => {
    if (previousStepRef.current === form.step) return;
    if (form.step === 'confirm') {
      confirmationHeadingRef.current?.focus();
    } else {
      reasonRef.current?.focus();
    }
    previousStepRef.current = form.step;
  }, [form.step]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (!form.isPending) onOpenChange(nextOpen);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent
          className="max-h-[92dvh] w-full max-w-[calc(100vw-1rem)] overflow-y-auto overflow-x-hidden p-4 shadow-none sm:max-w-lg sm:p-6"
          hideCloseButton={form.isPending}
        >
          <DialogHeader>
            <DialogTitle>Loại bảng công chưa thanh toán</DialogTitle>
            <DialogDescription>
              {form.step === 'form'
                ? 'Chọn một dự án, khoảng ngày và lý do áp dụng cho toàn bộ đợt.'
                : 'Kiểm tra phạm vi trước khi xác nhận thao tác không thể hoàn tác.'}
            </DialogDescription>
          </DialogHeader>

          {form.step === 'form' ? (
            <div className="min-w-0 space-y-5">
              <div className="min-w-0 space-y-2">
                <Label>Dự án</Label>
                <SearchableDropdown
                  value={form.projectId}
                  onValueChange={form.setProjectId}
                  options={form.projectOptions}
                  placeholder={form.isProjectsLoading ? 'Đang tải dự án...' : 'Chọn dự án'}
                  searchPlaceholder="Tìm dự án..."
                  emptyMessage="Không tìm thấy dự án"
                  mobileTitle="Chọn dự án"
                  className="w-full min-w-0"
                />
                {form.errors.project && (
                  <p className="text-sm text-destructive" role="alert">{form.errors.project}</p>
                )}
              </div>

              <div className="min-w-0 space-y-2">
                <Label>Khoảng ngày làm việc</Label>
                <DateRangePicker
                  startDate={form.fromDate}
                  endDate={form.toDate}
                  onStartDateChange={form.setFromDate}
                  onEndDateChange={form.setToDate}
                  variant={isMobile ? 'mobile' : 'default'}
                  className="min-h-11 w-full min-w-0 justify-between overflow-hidden rounded-lg border border-input bg-background px-3"
                  disabled={form.isPending}
                  usePortal
                />
                <p className="text-xs text-muted-foreground">Bao gồm cả ngày bắt đầu và ngày kết thúc.</p>
                {form.errors.dateRange && (
                  <p className="text-sm text-destructive" role="alert">{form.errors.dateRange}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="reject-unpaid-reason">Lý do loại</Label>
                <Textarea
                  ref={reasonRef}
                  id="reject-unpaid-reason"
                  value={form.rejectionReason}
                  onChange={(event) => form.setRejectionReason(event.target.value)}
                  placeholder="Nhập lý do áp dụng cho tất cả bảng công..."
                  className="min-h-24 resize-y"
                  maxLength={500}
                  disabled={form.isPending}
                />
                {form.errors.reason && (
                  <p className="text-sm text-destructive" role="alert">{form.errors.reason}</p>
                )}
              </div>
            </div>
          ) : (
            <div className="min-w-0 space-y-4">
              <h3
                ref={confirmationHeadingRef}
                tabIndex={-1}
                className="text-base font-semibold outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                Xác nhận phạm vi loại
              </h3>
              <div className="flex items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/5 p-3 text-sm">
                <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-destructive" aria-hidden="true" />
                <p className="min-w-0 text-foreground">
                  Tất cả bảng công chưa được thanh toán trong phạm vi này sẽ chuyển sang trạng thái bị loại.
                  Bảng công đã thanh toán luôn được giữ nguyên.
                </p>
              </div>
              <dl className="min-w-0 divide-y divide-border rounded-xl border px-3">
                <div className="grid min-w-0 grid-cols-[5.5rem_minmax(0,1fr)] gap-2 py-3 text-sm">
                  <dt className="text-muted-foreground">Dự án</dt>
                  <dd className="break-words text-right font-medium">{form.selectedProject?.name}</dd>
                </div>
                <div className="grid min-w-0 grid-cols-[5.5rem_minmax(0,1fr)] gap-2 py-3 text-sm">
                  <dt className="text-muted-foreground">Khoảng ngày</dt>
                  <dd className="break-words text-right font-medium tabular-nums">{form.fromDate} – {form.toDate}</dd>
                </div>
                <div className="grid min-w-0 grid-cols-[5.5rem_minmax(0,1fr)] gap-2 py-3 text-sm">
                  <dt className="text-muted-foreground">Lý do</dt>
                  <dd className="whitespace-pre-wrap break-words text-right font-medium">{form.rejectionReason.trim()}</dd>
                </div>
              </dl>
            </div>
          )}

          <DialogFooter className="pt-1">
            {form.step === 'form' ? (
              <>
                <Button
                  type="button"
                  variant="outline"
                  className="min-h-11 sm:min-h-10"
                  onClick={() => handleOpenChange(false)}
                >
                  Hủy
                </Button>
                <Button type="button" className="min-h-11 sm:min-h-10" onClick={form.handleReview}>
                  Kiểm tra phạm vi
                </Button>
              </>
            ) : (
              <>
                <Button
                  type="button"
                  variant="outline"
                  className="min-h-11 sm:min-h-10"
                  onClick={() => form.setStep('form')}
                  disabled={form.isPending}
                >
                  <ArrowLeft className="h-4 w-4" aria-hidden="true" />
                  Quay lại
                </Button>
                <Button
                  type="button"
                  variant="destructive"
                  className="min-h-11 sm:min-h-10"
                  onClick={() => void form.handleSubmit()}
                  disabled={form.isPending}
                >
                  {form.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                  ) : (
                    <Trash2 className="h-4 w-4" aria-hidden="true" />
                  )}
                  {form.isPending ? 'Đang loại...' : 'Xác nhận loại'}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <div id="datepicker-portal" />
    </>
  );
}
