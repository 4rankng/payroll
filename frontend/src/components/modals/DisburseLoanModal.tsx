import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { CheckCircle, X, Calendar, FileText, Banknote, Building, Loader2 } from 'lucide-react';
import { useDisburseLoan } from '@/hooks/api/useLoans';
import { formatVND } from '@/utils/loanHelpers';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import type { Loan } from '@/types/api/loan.types';

interface DisburseLoanModalProps {
  isOpen: boolean;
  onClose: () => void;
  loan: Loan;
}

interface FormData {
  disbursement_date: string;
  disbursement_reference: string;
  notes: string;
}

export function DisburseLoanModal({
  isOpen,
  onClose,
  loan,
}: DisburseLoanModalProps) {
  const [step, setStep] = useState<'form' | 'confirm'>('form');
  const [formData, setFormData] = useState<FormData | null>(null);

  const disburseLoan = useDisburseLoan();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormData>({
    defaultValues: {
      disbursement_date: new Date().toISOString().split('T')[0],
      disbursement_reference: '',
      notes: '',
    },
  });

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      reset({
        disbursement_date: new Date().toISOString().split('T')[0],
        disbursement_reference: '',
        notes: '',
      });
      setStep('form');
      setFormData(null);
    }
  }, [isOpen, reset]);

  const onFormSubmit = async (data: FormData) => {
    setFormData(data);
    setStep('confirm');
  };

  const handleConfirm = async () => {
    if (!formData) return;

    try {
      await disburseLoan.mutateAsync({
        id: loan.id,
        data: {
          disbursement_date: formData.disbursement_date,
          disbursement_reference: formData.disbursement_reference || undefined,
        },
      });

      onClose();
    } catch (error) {
      console.error('Disbursement error:', error);
      setStep('form');
    }
  };

  const handleBack = () => {
    setStep('form');
  };

  const formatDate = (dateString: string) => {
    try {
      const date = new Date(dateString);
      return format(date, 'dd MMMM yyyy', { locale: vi });
    } catch {
      return dateString;
    }
  };

  const isLoading = disburseLoan.isPending;

  return (
    <AlertDialog open={isOpen} onOpenChange={onClose}>
      <AlertDialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <AlertDialogHeader>
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-blue-100 flex items-center justify-center flex-shrink-0">
              <Banknote className="h-5 w-5 text-blue-600" />
            </div>
            <div>
              <AlertDialogTitle className="text-lg font-semibold">
                {step === 'form' ? 'Giải ngân khoản vay' : 'Xác nhận giải ngân'}
              </AlertDialogTitle>
              <AlertDialogDescription className="text-sm text-muted-foreground">
                {step === 'form' ? 'Nhập thông tin giải ngân' : 'Kiểm tra lại thông tin trước khi xác nhận'}
              </AlertDialogDescription>
            </div>
          </div>
        </AlertDialogHeader>

        {step === 'form' && (
          <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-4 mt-4">
            {/* Loan Info Summary */}
            <div className="bg-muted/50 rounded-xl p-3 space-y-3">
              <div className="typography-label-small text-muted-foreground">Thông tin khoản vay</div>
              <div className="space-y-2 typography-body-small">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Mã khoản vay:</span>
                  <span className="font-medium">{loan.loan_code}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Chủ nợ:</span>
                  <span className="font-medium">{loan.lender.name}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Số tiền vay:</span>
                  <span className="font-semibold text-blue-600">{formatVND(loan.principal_amount)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Lãi suất:</span>
                  <span className="font-medium">{loan.interest_rate_bps / 100}% /năm</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Kỳ hạn:</span>
                  <span className="font-medium">{loan.term_months} tháng</span>
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4">
              {/* Disbursement Date */}
              <div className="space-y-2">
                <Label htmlFor="disbursement_date" className="text-sm font-medium flex items-center gap-2">
                  <Calendar className="h-4 w-4" />
                  Ngày giải ngân *
                </Label>
                <Input
                  id="disbursement_date"
                  type="date"
                  {...register('disbursement_date', {
                    required: 'Ngày giải ngân là bắt buộc'
                  })}
                  className={errors.disbursement_date ? 'border-red-500' : ''}
                />
                {errors.disbursement_date && (
                  <p className="text-xs text-red-600">{errors.disbursement_date.message}</p>
                )}
              </div>

              {/* Disbursement Reference */}
              <div className="space-y-2">
                <Label htmlFor="disbursement_reference" className="text-sm font-medium flex items-center gap-2">
                  <Building className="h-4 w-4" />
                  Mã tham chiếu ngân hàng
                </Label>
                <Input
                  id="disbursement_reference"
                  {...register('disbursement_reference')}
                  placeholder="VD: BANK-TXN-20250105-001"
                  className="text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  Mã giao dịch từ ngân hàng (tùy chọn)
                </p>
              </div>

              {/* Notes */}
              <div className="space-y-2">
                <Label htmlFor="notes" className="text-sm font-medium flex items-center gap-2">
                  <FileText className="h-4 w-4" />
                  Ghi chú
                </Label>
                <Textarea
                  id="notes"
                  {...register('notes')}
                  placeholder="Ghi chú về khoản giải ngân này..."
                  className="text-sm resize-none"
                  rows={3}
                />
              </div>
            </div>

            <AlertDialogFooter className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={onClose}
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-11 px-3 sm:h-8 rounded border border-border bg-card text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                <X className="w-4 h-4" />
                Hủy
              </button>
              <button
                type="submit"
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-11 px-3 sm:h-8 rounded bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                Tiếp tục
              </button>
            </AlertDialogFooter>
          </form>
        )}

        {step === 'confirm' && formData && (
          <div className="space-y-4 mt-4">
            {/* Loan Summary */}
            <div className="bg-muted/50 rounded-xl p-3 space-y-2">
              <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Khoản vay
              </div>
              <div className="space-y-1 typography-body-small">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Mã khoản vay:</span>
                  <span className="font-medium">{loan.loan_code}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Số tiền:</span>
                  <span className="font-semibold text-blue-600">{formatVND(loan.principal_amount)}</span>
                </div>
              </div>
            </div>

            {/* Disbursement Details */}
            <div className="bg-muted/50 rounded-xl p-3 space-y-3">
              <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Thông tin giải ngân
              </div>

              <div className="space-y-2 typography-body-small">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Ngày giải ngân:</span>
                  <span className="font-semibold">
                    {formatDate(formData.disbursement_date)}
                  </span>
                </div>

                {formData.disbursement_reference && (
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Mã tham chiếu:</span>
                    <span className="font-medium">
                      {formData.disbursement_reference}
                    </span>
                  </div>
                )}

                {formData.notes && (
                  <div className="pt-2 border-t border-border">
                    <span className="text-muted-foreground">Ghi chú:</span>
                    <div className="mt-1">{formData.notes}</div>
                  </div>
                )}
              </div>
            </div>

            <div className="bg-amber-50 border border-amber-200 rounded-xl p-3">
              <p className="typography-body-small text-amber-800">
                ⚠️ Hành động này sẽ ghi nhận khoản vay đã được giải ngân và không thể hoàn thành.
              </p>
            </div>

            <AlertDialogFooter className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={handleBack}
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-11 px-3 sm:h-8 rounded border border-border bg-card text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                Quay lại
              </button>
              <button
                type="button"
                onClick={handleConfirm}
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-11 px-3 sm:h-8 rounded bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                {isLoading ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    Đang xử lý...
                  </>
                ) : (
                  <>
                    <CheckCircle className="w-4 h-4" />
                    Xác nhận giải ngân
                  </>
                )}
              </button>
            </AlertDialogFooter>
          </div>
        )}
      </AlertDialogContent>
    </AlertDialog>
  );
}

export default DisburseLoanModal;
