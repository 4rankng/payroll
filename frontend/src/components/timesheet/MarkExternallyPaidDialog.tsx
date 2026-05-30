import { memo, useCallback, useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { Loader2, Banknote, AlertTriangle } from 'lucide-react';
import { toast } from '@/components/ui/sonner';
import { showErrorNotification } from '@/utils/error-handler';
import { bulkTransferService } from '@/services/api/bulk-transfer.service';

interface MarkExternallyPaidDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Timesheet IDs that should be marked as paid (typically the failed rows
   * from a provider batch). When the list is empty, the dialog still allows the
   * user to enter a reference but the submit button is disabled.
   */
  timesheetIds: number[];
  /** Optional human label for the source batch (e.g. "Batch #42"). */
  batchLabel?: string;
  onSuccess?: () => void;
}

export const MarkExternallyPaidDialog = memo(function MarkExternallyPaidDialog({
  open,
  onOpenChange,
  timesheetIds,
  batchLabel,
  onSuccess,
}: MarkExternallyPaidDialogProps) {
  const [reference, setReference] = useState('');
  const [note, setNote] = useState('');

  useEffect(() => {
    if (!open) {
      setReference('');
      setNote('');
    }
  }, [open]);

  const mutation = useMutation({
    mutationFn: () =>
      bulkTransferService.markAsPaidExternally({
        timesheet_ids: timesheetIds,
        reference: reference.trim(),
        note: note.trim() || undefined,
      }),
    onSuccess: () => {
      toast({
        title: 'Đã ghi nhận thanh toán ngoài app',
        description: `${timesheetIds.length} bảng công đã được đánh dấu là Đã thanh toán.`,
      });
      onSuccess?.();
      onOpenChange(false);
    },
    onError: (err) => {
      showErrorNotification(err);
    },
  });

  const handleSubmit = useCallback(() => {
    if (!reference.trim() || timesheetIds.length === 0) return;
    mutation.mutate();
  }, [reference, timesheetIds.length, mutation]);

  const canSubmit = reference.trim().length > 0 && timesheetIds.length > 0 && !mutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md shadow-none">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Banknote className="w-5 h-5 text-white/80" />
            Trả ngoài app
          </DialogTitle>
          <DialogDescription>
            Ghi nhận các bảng công này là <span className="font-semibold text-foreground">Đã thanh toán</span> bằng phương thức ngoài hệ thống
            (chuyển khoản tay, tiền mặt, app khác). Cung cấp số tham chiếu để truy vết về sau.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-1">
          <div className="rounded-lg border bg-muted/30 px-3 py-2 text-xs space-y-1">
            <div className="flex items-baseline justify-between">
              <span className="text-muted-foreground">Số bảng công</span>
              <span className="font-semibold tabular-nums text-foreground">{timesheetIds.length}</span>
            </div>
            {batchLabel && (
              <div className="flex items-baseline justify-between">
                <span className="text-muted-foreground">Nguồn</span>
                <span className="font-medium text-foreground truncate max-w-[60%] text-right">{batchLabel}</span>
              </div>
            )}
          </div>

          {timesheetIds.length === 0 && (
            <div className="flex items-start gap-2 rounded-md border border-amber-300/60 bg-amber-50 p-2 text-xs text-amber-800">
              <AlertTriangle className="h-3.5 w-3.5 shrink-0 mt-0.5" />
              <span>Chưa có bảng công nào được chọn để đánh dấu.</span>
            </div>
          )}

          <div className="space-y-1.5">
            <Label htmlFor="external-paid-ref" className="text-xs font-medium">
              Số tham chiếu <span className="text-red-500">*</span>
            </Label>
            <Input
              id="external-paid-ref"
              value={reference}
              onChange={(e) => setReference(e.target.value)}
              placeholder="Ví dụ: VCB-20260509-XYZ123"
              maxLength={120}
              autoFocus
            />
            <p className="text-xs text-muted-foreground">
              Có thể là mã giao dịch ngân hàng, số phiếu chi, hoặc bất kỳ định danh nào giúp đối soát.
            </p>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="external-paid-note" className="text-xs font-medium">
              Ghi chú (tuỳ chọn)
            </Label>
            <Textarea
              id="external-paid-note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Lý do trả ngoài, kênh sử dụng, v.v."
              rows={3}
              maxLength={500}
            />
          </div>
        </div>

        <DialogFooter className="flex-row gap-3">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={mutation.isPending} className="flex-1">
            Hủy
          </Button>
          <Button onClick={handleSubmit} disabled={!canSubmit} className="flex-1">
            {mutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Đang ghi nhận...
              </>
            ) : (
              <>
                <Banknote className="w-4 h-4 mr-2" />
                Đánh dấu đã trả
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
