import { format } from "date-fns";
import { useState, useEffect, useCallback, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Calendar, DollarSign, FileText, CheckCircle2, AlertCircle } from 'lucide-react';
import { useRepaySchedule } from '@/hooks/api/useLoans';
import { formatVND } from '@/utils/loanHelpers';
import { cn } from '@/lib/utils';
import type { Loan, CustomScheduleItem } from '@/types/api/loan.types';

interface RepayScheduleModalProps {
  isOpen: boolean;
  onClose: () => void;
  loan: Loan | null;
}

type FormData = {
  schedule_id: string;
  payment_date: string;
  payment_reference: string;
  notes: string;
};

export function RepayScheduleModal({ isOpen, onClose, loan }: RepayScheduleModalProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [form, setForm] = useState<FormData>({
    schedule_id: '',
    payment_date: new Date().toISOString().split('T')[0],
    payment_reference: '',
    notes: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const repaySchedule = useRepaySchedule();

  // Get pending schedules
  const pendingSchedules = useMemo(() => {
    if (!loan || !loan.schedules) return [];
    return loan.schedules.filter(s => s.status === 'pending');
  }, [loan]);

  // Get selected schedule
  const selectedSchedule = useMemo(() => {
    if (!form.schedule_id || !loan?.schedules) return null;
    return loan.schedules.find(s => s.id === Number(form.schedule_id)) || null;
  }, [form.schedule_id, loan]);

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setStep(1);
      setForm({
        schedule_id: pendingSchedules.length > 0 ? pendingSchedules[0].id.toString() : '',
        payment_date: new Date().toISOString().split('T')[0],
        payment_reference: '',
        notes: '',
      });
      setErrors({});
    }
  }, [isOpen, pendingSchedules]);

  const updateField = useCallback((key: keyof FormData, value: string) => {
    setForm(prev => ({ ...prev, [key]: value }));
    if (errors[key]) {
      setErrors(prev => ({ ...prev, [key]: '' }));
    }
  }, [errors]);

  const validate = useCallback(() => {
    const nextErrors: Record<string, string> = {};

    if (!form.schedule_id) {
      nextErrors.schedule_id = 'Vui lòng chọn kỳ thanh toán';
    }
    if (!form.payment_date) {
      nextErrors.payment_date = 'Vui lòng chọn ngày thanh toán';
    }

    setErrors(nextErrors);
    return Object.keys(nextErrors).length === 0;
  }, [form]);

  const handleNext = useCallback(() => {
    if (validate()) {
      setStep(2);
    }
  }, [validate]);

  const handleBack = useCallback(() => {
    setStep(1);
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!loan) return;

    try {
      await repaySchedule.mutateAsync({
        id: loan.id,
        data: {
          schedule_id: Number(form.schedule_id),
          payment_date: form.payment_date,
          payment_reference: form.payment_reference.trim() || undefined,
          notes: form.notes.trim() || undefined,
        },
      });
      onClose();
    } catch (error) {
      // Error handled by mutation hook
      setStep(1); // Go back to step 1 to allow correction
    }
  }, [loan, form, repaySchedule, onClose]);

  if (!loan) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-h-[92dvh] overflow-y-auto sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Thanh toán theo lịch</DialogTitle>
          <DialogDescription>
            {step === 1 ? 'Nhập thông tin thanh toán' : 'Xác nhận thông tin thanh toán'}
          </DialogDescription>
        </DialogHeader>

        {step === 1 && (
          <div className="space-y-4 py-4">
            {/* Schedule Selection */}
            <div className="space-y-2">
              <Label className="typography-label-medium">Kỳ thanh toán *</Label>
              <Select
                value={form.schedule_id}
                onValueChange={(v) => updateField('schedule_id', v)}
              >
                <SelectTrigger className={cn("h-11", errors.schedule_id && 'border-red-500')}>
                  <SelectValue placeholder="Chọn kỳ thanh toán" />
                </SelectTrigger>
                <SelectContent>
                  {pendingSchedules.length === 0 ? (
                    <div className="p-2 text-center text-sm text-muted-foreground">
                      Không có kỳ thanh toán nào chờ xử lý
                    </div>
                  ) : (
                    pendingSchedules.map((schedule) => (
                      <SelectItem key={schedule.id} value={schedule.id.toString()}>
                        Kỳ {schedule.period} - {format(new Date(schedule.due_date), 'dd/MM/yyyy')} - {formatVND(schedule.amount)}
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
              {errors.schedule_id && (
                <p className="typography-body-small text-financial-negative">{errors.schedule_id}</p>
              )}
              {selectedSchedule && (
                <div className="mt-2 rounded-xl border border-blue-200 bg-blue-50 p-3">
                  <div className="flex items-start gap-2 typography-body-small font-semibold text-blue-800">
                    <DollarSign className="h-4 w-4" />
                    <span className="break-words">Số tiền: {formatVND(selectedSchedule.amount)}</span>
                  </div>
                </div>
              )}
            </div>

            {/* Payment Date */}
            <div className="space-y-2">
              <Label className="typography-label-medium">Ngày thanh toán *</Label>
              <Input
                type="date"
                value={form.payment_date}
                onChange={(e) => updateField('payment_date', e.target.value)}
                className={cn("h-11", errors.payment_date && 'border-red-500')}
              />
              {errors.payment_date && (
                <p className="typography-body-small text-financial-negative">{errors.payment_date}</p>
              )}
            </div>

            {/* Payment Reference */}
            <div className="space-y-2">
              <Label className="typography-label-medium">Mã tham chiếu</Label>
              <Input
                value={form.payment_reference}
                onChange={(e) => updateField('payment_reference', e.target.value)}
                placeholder="Mã giao dịch hoặc số chứng từ (nếu có)"
                className="h-11"
              />
            </div>

            {/* Notes */}
            <div className="space-y-2">
              <Label className="typography-label-medium">Ghi chú</Label>
              <Textarea
                value={form.notes}
                onChange={(e) => updateField('notes', e.target.value)}
                placeholder="Ghi chú thêm (nếu có)"
                className="min-h-24"
              />
            </div>
          </div>
        )}

        {step === 2 && selectedSchedule && (
          <div className="space-y-4 py-4">
            <div className="bg-amber-50 border border-amber-200 rounded-xl p-4 flex items-start gap-3">
              <AlertCircle className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
              <div className="space-y-1">
                <p className="typography-body-medium font-semibold text-amber-800">
                  Xác nhận thanh toán
                </p>
                <p className="typography-body-small text-amber-700">
                  Sau khi thanh toán, bạn không thể hoàn thành. Vui lòng kiểm tra kỹ thông tin trước khi xác nhận.
                </p>
              </div>
            </div>

            <div className="space-y-3 bg-muted/50 rounded-xl p-3">
              <div className="typography-label-medium text-muted-foreground uppercase font-semibold">
                Thông tin thanh toán
              </div>

              <div className="space-y-3">
                <div className="flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="typography-body-small text-muted-foreground">Kỳ thanh toán:</span>
                  <span className="typography-body-medium font-semibold min-[380px]:text-right">Kỳ {selectedSchedule.period}</span>
                </div>

                <div className="flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="typography-body-small text-muted-foreground">Ngày đáo hạn:</span>
                  <span className="typography-body-medium min-[380px]:text-right">{format(new Date(selectedSchedule.due_date), 'dd/MM/yyyy')}</span>
                </div>

                <div className="flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="typography-body-small text-muted-foreground">Số tiền:</span>
                  <span className="typography-body-medium break-words font-semibold text-blue-600 min-[380px]:text-right">
                    {formatVND(selectedSchedule.amount)}
                  </span>
                </div>

                <div className="flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                  <span className="typography-body-small text-muted-foreground">Ngày thanh toán:</span>
                  <span className="typography-body-medium min-[380px]:text-right">{form.payment_date}</span>
                </div>

                {form.payment_reference && (
                  <div className="flex flex-col gap-1 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
                    <span className="typography-body-small text-muted-foreground">Mã tham chiếu:</span>
                    <span className="typography-body-medium break-all min-[380px]:text-right">{form.payment_reference}</span>
                  </div>
                )}

                {form.notes && (
                  <div className="space-y-1">
                    <span className="typography-body-small text-muted-foreground">Ghi chú:</span>
                    <p className="typography-body-small break-words text-foreground">{form.notes}</p>
                  </div>
                )}
              </div>
            </div>

            <div className="flex items-start gap-2 rounded-xl border border-blue-200 bg-blue-50 p-3">
              <Calendar className="h-4 w-4 text-blue-600 flex-shrink-0" />
              <p className="typography-body-small text-blue-800">
                Dư nợ sau thanh toán: <span className="font-semibold">
                  {formatVND(loan.outstanding_principal - selectedSchedule.amount)}
                </span>
              </p>
            </div>
          </div>
        )}

        <DialogFooter className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {step === 1 && (
            <>
              <button
                onClick={onClose}
                className="inline-flex min-h-11 items-center justify-center gap-1.5 rounded border border-border bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted"
              >
                Hủy
              </button>
              <button
                onClick={handleNext}
                disabled={pendingSchedules.length === 0}
                className="inline-flex min-h-11 items-center justify-center gap-1.5 rounded bg-primary px-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
              >
                Tiếp tục
              </button>
            </>
          )}

          {step === 2 && (
            <>
              <button
                onClick={handleBack}
                className="inline-flex min-h-11 items-center justify-center gap-1.5 rounded border border-border bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted"
              >
                Quay lại
              </button>
              <button
                onClick={handleSubmit}
                disabled={repaySchedule.isPending}
                className="inline-flex min-h-11 items-center justify-center gap-1.5 rounded bg-primary px-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
              >
                {repaySchedule.isPending ? (
                  'Đang xử lý...'
                ) : (
                  <>
                    <CheckCircle2 className="w-4 h-4" />
                    Xác nhận thanh toán
                  </>
                )}
              </button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default RepayScheduleModal;
