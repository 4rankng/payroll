import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useSearchParams } from 'react-router-dom';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { RotateCcw, X, AlertTriangle, Loader2 } from 'lucide-react';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useReverseTransaction } from '@/hooks/transactions/useReverseTransaction';
import { transactionService } from '@/services/api/transaction.service';
import { formatCurrency } from '@/utils/formatters';
import type { ModalConfig } from '@/types/modal-config.types';
import type { Transaction } from '@/services/api/transaction.service';

export const modalConfig: ModalConfig = {
  id: 'reverse-transaction',
  name: 'Reverse Transaction',
  description: 'Reverse a transaction by creating an opposite entry',
  category: 'ledger',
  permissions: {
    action: 'update',
    subject: 'Transaction',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id'],
    example: '?modal=reverse_transaction&id=123',
  },
  requiresAuth: true,
  encryptData: false,
};

interface ReverseTransactionModalProps {
  isOpen: boolean;
  onClose: () => void;
  transaction: Transaction;
}

interface FormData {
  reason: string;
}

function ReverseTransactionModalComponent({
  isOpen,
  onClose,
  transaction,
}: ReverseTransactionModalProps) {
  const [step, setStep] = useState<'form' | 'confirm'>('form');
  const [formData, setFormData] = useState<FormData | null>(null);

  const reverseTransaction = useReverseTransaction();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormData>({
    defaultValues: {
      reason: '',
    },
  });

  // Initialize form when opening
  useEffect(() => {
    if (isOpen) {
      reset({
        reason: '',
      });
      setStep('form');
      setFormData(null);
    }
  }, [isOpen, reset]);

  const onFormSubmit = async (data: FormData) => {
    // Store form data for confirmation step
    setFormData(data);
    setStep('confirm');
  };

  const handleConfirm = async () => {
    try {
      await reverseTransaction.mutateAsync({
        id: transaction.id,
        reason: formData?.reason || undefined,
      });

      onClose();
    } catch (error) {
      console.error('Reverse transaction error:', error);
      setStep('form');
    }
  };

  const handleBack = () => {
    setStep('form');
  };

  const isLoading = reverseTransaction.isPending;

  return (
    <AlertDialog open={isOpen} onOpenChange={onClose}>
      <AlertDialogContent className="max-w-xl">
        <AlertDialogHeader>
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-orange-100 flex items-center justify-center flex-shrink-0">
              <RotateCcw className="h-5 w-5 text-orange-600" />
            </div>
            <div>
              <AlertDialogTitle className="text-lg font-semibold">
                {step === 'form' ? 'Đảo ngược giao dịch' : 'Xác nhận đảo ngược'}
              </AlertDialogTitle>
              <AlertDialogDescription className="text-sm text-muted-foreground">
                {step === 'form' ? 'Tạo bút toán đảo ngược để hủy giao dịch này' : 'Kiểm tra lại thông tin trước khi xác nhận'}
              </AlertDialogDescription>
            </div>
          </div>
        </AlertDialogHeader>

        {step === 'form' && (
          <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-4 mt-4">
            {/* Transaction Info */}
            <div className="bg-muted/50 rounded-xl p-3 space-y-2">
              <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Thông tin giao dịch</div>
              <div className="space-y-1.5 typography-body-small">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Mã giao dịch:</span>
                  <span className="font-medium">{transaction.transaction_code}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Số tiền:</span>
                  <span className={`font-semibold ${
                    transaction.transaction_type === 'revenue' ? 'text-emerald-600' : 'text-red-600'
                  }`}>
                    {formatCurrency(transaction.amount)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Đối tượng:</span>
                  <span className="font-medium">{transaction.party}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Diễn giải:</span>
                  <span className="font-medium truncate max-w-xs" title={transaction.description}>
                    {transaction.description}
                  </span>
                </div>
              </div>
            </div>

            {/* Warning Alert */}
            <div className="bg-amber-50 border border-amber-200 rounded-xl p-3 flex gap-3">
              <AlertTriangle className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
              <div className="space-y-1">
                <p className="typography-body-small font-medium text-amber-900">
                  Lưu ý quan trọng
                </p>
                <p className="typography-body-small text-amber-800">
                  Hành động này sẽ tạo một giao dịch đảo ngược để hủy giao dịch gốc. Hai giao dịch sẽ triệt tiêu lẫn nhau trong sổ cái.
                </p>
              </div>
            </div>

            {/* Reason */}
            <div className="space-y-2">
              <Label htmlFor="reason" className="text-sm font-medium">
                Lý do đảo ngược (tùy chọn)
              </Label>
              <Textarea
                id="reason"
                {...register('reason')}
                placeholder="Ví dụ: Giao dịch bị trùng, sai thông tin..."
                className="text-sm resize-none"
                rows={3}
              />
              {errors.reason && (
                <p className="text-xs text-red-600">{errors.reason.message}</p>
              )}
            </div>

            <AlertDialogFooter className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={onClose}
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded border border-border bg-background text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                <X className="w-4 h-4" />
                Hủy
              </button>
              <button
                type="submit"
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded bg-destructive text-destructive-foreground text-sm font-medium whitespace-nowrap hover:bg-destructive/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                Tiếp tục
              </button>
            </AlertDialogFooter>
          </form>
        )}

        {step === 'confirm' && formData && (
          <div className="space-y-4 mt-4">
            <div className="bg-muted/50 rounded-xl p-3 space-y-3">
              <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                Xác nhận đảo ngược
              </div>

              <div className="space-y-2 typography-body-small">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Giao dịch gốc:</span>
                  <span className="font-semibold">{transaction.transaction_code}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Số tiền:</span>
                  <span className="font-semibold">{formatCurrency(transaction.amount)}</span>
                </div>
                {formData.reason && (
                  <div className="pt-2 border-t border-border">
                    <span className="text-muted-foreground">Lý do:</span>
                    <div className="mt-1">{formData.reason}</div>
                  </div>
                )}
              </div>
            </div>

            <div className="bg-red-50 border border-red-200 rounded-xl p-3">
              <p className="typography-body-small text-red-800">
                ⚠️ Hành động này sẽ tạo giao dịch đảo ngược và không thể hoàn thành.
              </p>
            </div>

            <AlertDialogFooter className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={handleBack}
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded border border-border bg-background text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                Quay lại
              </button>
              <button
                type="button"
                onClick={handleConfirm}
                disabled={isLoading}
                className="inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded bg-destructive text-destructive-foreground text-sm font-medium whitespace-nowrap hover:bg-destructive/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                {isLoading ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    Đang xử lý...
                  </>
                ) : (
                  <>
                    <RotateCcw className="w-4 h-4" />
                    Xác nhận đảo ngược
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

/**
 * Container component that provides URL synchronization
 */
export function ReverseTransactionModal({
  isOpen: propIsOpen,
  onClose: propOnClose,
  transaction,
}: ReverseTransactionModalProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();

  const modalId = searchParams.get('modal');
  const isOpenViaUrl = modalId === 'reverse_transaction';

  const isOpen = isOpenViaUrl || propIsOpen;

  const handleClose = () => {
    if (isOpenViaUrl) {
      closeModal();
    } else {
      propOnClose();
    }
  };

  return (
    <ReverseTransactionModalComponent
      isOpen={isOpen}
      onClose={handleClose}
      transaction={transaction}
    />
  );
}

export default ReverseTransactionModal;
