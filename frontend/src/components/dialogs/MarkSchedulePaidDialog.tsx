import { format } from "date-fns";
import { useState, useEffect, useCallback, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogNavyHeader, DialogFooter } from '@/components/ui/dialog';
import { Calendar, CheckCircle2, AlertTriangle, Loader2, Banknote } from 'lucide-react';
import { useRepaySchedule } from '@/hooks/api/useLoans';
import { formatVND } from '@/utils/loanHelpers';
import type { Loan, CustomScheduleItem } from '@/types/api/loan.types';

interface MarkSchedulePaidDialogProps {
  isOpen: boolean;
  onClose: () => void;
  loan: Loan | null;
  schedule: CustomScheduleItem | null;
}

type FormData = {
  payment_date: string;
  payment_reference: string;
};

export function MarkSchedulePaidDialog({ isOpen, onClose, loan, schedule }: MarkSchedulePaidDialogProps) {
  const [form, setForm] = useState<FormData>({
    payment_date: new Date().toISOString().split('T')[0],
    payment_reference: '',
  });

  const repaySchedule = useRepaySchedule();

  useEffect(() => {
    if (isOpen) {
      setForm({
        payment_date: new Date().toISOString().split('T')[0],
        payment_reference: '',
      });
    }
  }, [isOpen]);

  const updateField = useCallback((key: keyof FormData, value: string) => {
    setForm(prev => ({ ...prev, [key]: value }));
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!loan || !schedule) return;
    try {
      await repaySchedule.mutateAsync({
        id: loan.id,
        data: {
          schedule_id: schedule.id,
          payment_date: form.payment_date,
          payment_reference: form.payment_reference.trim() || undefined,
        },
      });
      onClose();
    } catch {
      // handled in hook
    }
  }, [loan, schedule, form, repaySchedule, onClose]);

  const isValid = useMemo(() => form.payment_date !== '', [form.payment_date]);

  if (!loan || !schedule) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-md gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
        <DialogNavyHeader
          title="Xác nhận thanh toán"
          description={`Kỳ ${schedule.period} — không thể hoàn tác sau khi xác nhận`}
        />

        <div className="px-5 py-4 space-y-4">
          {/* Warning */}
          <div className="flex items-start gap-2.5 rounded-xl border border-warning/30 bg-warning/5 px-3 py-2.5">
            <AlertTriangle className="h-4 w-4 text-warning flex-shrink-0 mt-0.5" />
            <p className="text-xs text-muted-foreground">
              Vui lòng kiểm tra kỹ thông tin trước khi xác nhận. Hành động này không thể hoàn tác.
            </p>
          </div>

          {/* Schedule summary */}
          <div className="rounded-xl border bg-muted/20 divide-y divide-border/60">
            <div className="flex items-center justify-between px-3 py-2.5">
              <span className="text-xs text-muted-foreground">Kỳ thanh toán</span>
              <span className="text-sm font-medium">Kỳ {schedule.period}</span>
            </div>
            <div className="flex items-center justify-between px-3 py-2.5">
              <span className="text-xs text-muted-foreground">Ngày đáo hạn</span>
              <span className="text-sm font-medium">{format(new Date(schedule.due_date), 'dd/MM/yyyy')}</span>
            </div>
            <div className="flex items-center justify-between px-3 py-2.5">
              <span className="text-xs text-muted-foreground">Số tiền</span>
              <span className="text-sm font-semibold text-primary">{formatVND(schedule.amount)}</span>
            </div>
            <div className="flex items-center justify-between px-3 py-2.5">
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Banknote className="w-3.5 h-3.5" />
                Dư nợ sau thanh toán
              </div>
              <span className="text-sm font-medium">{formatVND(Math.max(loan.outstanding_principal - schedule.principal_amount, 0))}</span>
            </div>
          </div>

          {/* Form fields */}
          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium flex items-center gap-1.5">
                <Calendar className="w-3.5 h-3.5 text-muted-foreground" />
                Ngày thanh toán <span className="text-destructive">*</span>
              </Label>
              <Input
                type="date"
                value={form.payment_date}
                onChange={(e) => updateField('payment_date', e.target.value)}
                className="h-9"
              />
            </div>

            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Mã tham chiếu</Label>
              <Input
                value={form.payment_reference}
                onChange={(e) => updateField('payment_reference', e.target.value)}
                placeholder="Mã giao dịch hoặc số chứng từ (nếu có)"
                className="h-9"
              />
            </div>
          </div>
        </div>

        <DialogFooter className="px-5 py-4 border-t flex flex-row gap-2">
          <Button variant="outline" onClick={onClose} className="min-w-[72px]">
            Hủy
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={!isValid || repaySchedule.isPending}
            className="flex-1 gap-1.5"
          >
            {repaySchedule.isPending ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Đang xử lý...
              </>
            ) : (
              <>
                <CheckCircle2 className="w-4 h-4" />
                Xác nhận thanh toán
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default MarkSchedulePaidDialog;
