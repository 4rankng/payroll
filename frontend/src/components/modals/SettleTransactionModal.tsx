import { format } from "date-fns";
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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ButtonGroup } from '@/components/ui/button-group';
import { FileUpload } from '@/components/ui/FileUpload';
import { UrlInput } from '@/components/ui/UrlInput';
import { CheckCircle, X, Receipt, Calendar, DollarSign, FileText, Loader2 } from 'lucide-react';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useUploadAsset } from '@/hooks/api/useAssets';
import { useSettleTransaction } from '@/hooks/transactions/useSettleTransaction';
import { transactionService, getTransactionRemainingAmount } from '@/services/api/transaction.service';
import type { ModalConfig } from '@/types/modal-config.types';
import type { Transaction } from '@/services/api/transaction.service';

export const modalConfig: ModalConfig = {
  id: 'settle-transaction',
  name: 'Settle Transaction',
  description: 'Settle a pending transaction with payment details',
  category: 'ledger',
  permissions: {
    action: 'update',
    subject: 'Transaction',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id'],
    example: '?modal=settle_transaction&id=123',
  },
  requiresAuth: true,
  encryptData: false,
};

interface SettleTransactionModalProps {
  isOpen: boolean;
  onClose: () => void;
  transaction: Transaction;
}

interface FormData {
  amount: string;
  settlement_date: string;
  payment_method: string;
  notes: string;
  evidenceUrl?: string;
}

function SettleTransactionModalComponent({
  isOpen,
  onClose,
  transaction,
}: SettleTransactionModalProps) {
  const [step, setStep] = useState<'form' | 'confirm'>('form');
  const [evidenceType, setEvidenceType] = useState<'url' | 'file'>('url');
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [fileUploadError, setFileUploadError] = useState<string | null>(null);
  const [formData, setFormData] = useState<FormData | null>(null);

  const settleTransaction = useSettleTransaction();
  const uploadAsset = useUploadAsset();

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormData>({
    defaultValues: {
      amount: '',
      settlement_date: new Date().toISOString().split('T')[0],
      payment_method: '',
      notes: '',
      evidenceUrl: '',
    },
  });

  const watchedAmount = watch('amount');

  // Initialize form when opening
  useEffect(() => {
    if (isOpen) {
      // For partial settlements, use remaining amount if available, otherwise full amount
      const amountToSettle = getTransactionRemainingAmount(transaction);
      const formattedAmount = amountToSettle.toLocaleString('vi-VN');
      reset({
        amount: formattedAmount,
        settlement_date: new Date().toISOString().split('T')[0],
        payment_method: '',
        notes: '',
        evidenceUrl: '',
      });
      setStep('form');
      setEvidenceType('url');
      setEvidenceFile(null);
      setFileUploadError(null);
      setFormData(null);
    }
  }, [isOpen, reset, transaction]);

  const handleAmountChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    let value = e.target.value.replace(/[^0-9]/g, '');
    if (value) {
      value = parseInt(value).toLocaleString('vi-VN');
    }
    setValue('amount', value);
  };

  const handleFileUpload = (files: File[]) => {
    const file = files[0];
    if (!file) return;

    setEvidenceFile(file);
    setFileUploadError(null);
  };

  const handleRemoveFile = () => {
    setEvidenceFile(null);
    setFileUploadError(null);
  };

  const onFormSubmit = async (data: FormData) => {
    // Store form data for confirmation step
    setFormData(data);
    setStep('confirm');
  };

  const handleConfirm = async () => {
    if (!formData) return;

    try {
      let proofAssetId: number | undefined;

      // Upload evidence file if needed
      if (evidenceType === 'file' && evidenceFile) {
        try {
          setFileUploadError(null);
          const uploadResult = await uploadAsset.mutateAsync({
            data: {
              file: evidenceFile,
              upload_type: 'ledger_evidence',
            }
          });
          proofAssetId = uploadResult.data.id;
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error';
          setFileUploadError(`Không thể tải lên file: ${errorMessage}`);
          setStep('form');
          return;
        }
      }

      // Parse amount from formatted string
      const parsedAmount = parseFloat(formData.amount.replace(/\./g, ''));

      // Call settlement API with proper data
      await settleTransaction.mutateAsync({
        id: transaction.id,
        data: {
          amount: parsedAmount,
          settlement_date: formData.settlement_date,
          proof_url: evidenceType === 'url' ? formData.evidenceUrl : undefined,
          proof_asset_id: evidenceType === 'file' ? proofAssetId : undefined,
          payment_method: formData.payment_method || undefined,
          notes: formData.notes || undefined,
        },
      });

      onClose();
    } catch (error) {
      console.error('Settlement error:', error);
      setStep('form');
    }
  };

  const handleBack = () => {
    setStep('form');
  };

  const formatCurrency = (amount: number) => {
    return transactionService.formatCurrency(amount);
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return format(date, 'dd/MM/yyyy');
  };

  const isLoading = settleTransaction.isPending || uploadAsset.isPending;

  return (
    <AlertDialog open={isOpen} onOpenChange={onClose}>
      <AlertDialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <AlertDialogHeader>
          <div className="flex items-center gap-3">
            <div className={`h-10 w-10 rounded-xl flex items-center justify-center flex-shrink-0 ${
              transaction.transaction_type === 'revenue' ? 'bg-emerald-100' : 'bg-red-100'
            }`}>
              <CheckCircle className={`h-5 w-5 ${
                transaction.transaction_type === 'revenue' ? 'text-emerald-600' : 'text-red-600'
              }`} />
            </div>
            <div>
              <AlertDialogTitle className="text-lg font-semibold">
                {step === 'form' ? 'Thanh toán giao dịch' : 'Xác nhận thanh toán'}
              </AlertDialogTitle>
              <AlertDialogDescription className="text-sm text-muted-foreground">
                {step === 'form' ? 'Nhập thông tin thanh toán' : 'Kiểm tra lại thông tin trước khi xác nhận'}
              </AlertDialogDescription>
            </div>
          </div>
        </AlertDialogHeader>

        {step === 'form' && (
          <form onSubmit={handleSubmit(onFormSubmit)} className="space-y-4 mt-4">
            {/* Two Column Layout: Transaction Info & Payment Details (Left) | Evidence & Notes (Right) */}
            <div className="grid grid-cols-2 gap-4">
              {/* Left Pane: Transaction Info & Payment Details */}
              <div className="space-y-4">
                {/* Original Transaction Info */}
                <div className="bg-muted/50 rounded-xl p-3 space-y-2">
                  <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Thông tin giao dịch</div>
                  <div className="space-y-1 typography-body-small">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Số tiền gốc:</span>
                      <span className="font-medium">{formatCurrency(transaction.amount)}</span>
                    </div>
                    {transaction.settled_amount !== undefined && transaction.settled_amount > 0 && (
                      <>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Đã thanh toán:</span>
                          <span className="font-medium text-green-600">{formatCurrency(transaction.settled_amount)}</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">Còn lại:</span>
                          <span className="font-medium text-amber-600">{formatCurrency(getTransactionRemainingAmount(transaction))}</span>
                        </div>
                      </>
                    )}
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Đối tượng:</span>
                      <span className="font-medium">{transaction.party}</span>
                    </div>
                  </div>
                </div>

                {/* Amount */}
                <div className="space-y-2">
                  <Label htmlFor="amount" className="text-sm font-medium flex items-center gap-2">
                    <DollarSign className="h-4 w-4" />
                    Số tiền thanh toán *
                  </Label>
                  <div className="relative">
                    <Input
                      id="amount"
                      value={watchedAmount}
                      onChange={handleAmountChange}
                      placeholder="0"
                      className={`pr-12 text-right ${errors.amount ? 'border-red-500' : ''}`}
                      required
                    />
                    <span className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground text-sm">
                      đ
                    </span>
                  </div>
                  {errors.amount && (
                    <p className="text-xs text-red-600">{errors.amount.message}</p>
                  )}
                </div>

                {/* Settlement Date */}
                <div className="space-y-2">
                  <Label htmlFor="settlement_date" className="text-sm font-medium flex items-center gap-2">
                    <Calendar className="h-4 w-4" />
                    Ngày thanh toán *
                  </Label>
                  <Input
                    id="settlement_date"
                    type="date"
                    {...register('settlement_date', { required: 'Ngày thanh toán là bắt buộc' })}
                    className={errors.settlement_date ? 'border-red-500' : ''}
                  />
                  {errors.settlement_date && (
                    <p className="text-xs text-red-600">{errors.settlement_date.message}</p>
                  )}
                </div>


              </div>

              {/* Right Pane: Evidence Upload & Notes */}
              <div className="space-y-4">
                {/* Evidence Upload */}
                <div className="space-y-2">
                  <Label className="text-sm font-medium">Chứng từ thanh toán</Label>

                  <ButtonGroup
                    options={[
                      { value: 'url', label: 'Link URL' },
                      { value: 'file', label: 'Tải lên file' }
                    ]}
                    value={evidenceType}
                    onChange={(value) => {
                      setEvidenceType(value as 'url' | 'file');
                      if (value === 'file') {
                        setValue('evidenceUrl', '');
                      } else {
                        setEvidenceFile(null);
                      }
                      setFileUploadError(null);
                    }}
                    fullWidth
                  />

                  {evidenceType === 'url' && (
                    <UrlInput
                      value={watch('evidenceUrl') || ''}
                      onChange={(url) => setValue('evidenceUrl', url)}
                      placeholder="https://drive.google.com/..."
                      label=""
                    />
                  )}

                  {evidenceType === 'file' && (
                    <div className="space-y-2">
                      {!evidenceFile ? (
                        <FileUpload
                          onFilesSelected={handleFileUpload}
                          multiple={false}
                          maxFiles={1}
                        />
                      ) : (
                        <div className="flex items-center justify-between gap-2 p-3 bg-green-50 rounded border">
                          <div className="flex items-center gap-2">
                            <Receipt className="h-4 w-4 text-green-600" />
                            <span className="text-sm text-green-700">
                              {evidenceFile.name}
                            </span>
                          </div>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onClick={handleRemoveFile}
                            className="h-auto p-1 text-muted-foreground hover:text-red-600"
                          >
                            <X className="h-4 w-4" />
                          </Button>
                        </div>
                      )}
                      {fileUploadError && (
                        <p className="text-xs text-red-600">{fileUploadError}</p>
                      )}
                    </div>
                  )}
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
                    placeholder="Ghi chú về khoản thanh toán này..."
                    className="text-sm resize-none"
                    rows={4}
                  />
                </div>
              </div>
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
                className="inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
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
                Thông tin thanh toán
              </div>

              <div className="space-y-2 typography-body-small">
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Số tiền thanh toán:</span>
                  <span className="font-semibold">
                    {formatCurrency(parseFloat(formData.amount.replace(/\./g, '')))}
                  </span>
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground">Ngày thanh toán:</span>
                  <span className="font-medium">{formatDate(formData.settlement_date)}</span>
                </div>

                {formData.payment_method && (
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Phương thức:</span>
                    <span className="font-medium">{formData.payment_method}</span>
                  </div>
                )}

                {formData.notes && (
                  <div className="pt-2 border-t border-border">
                    <span className="text-muted-foreground">Ghi chú:</span>
                    <div className="mt-1">{formData.notes}</div>
                  </div>
                )}

                {evidenceType === 'url' && formData.evidenceUrl && (
                  <div className="pt-2 border-t border-border">
                    <span className="text-muted-foreground">Chứng từ (URL):</span>
                    <div className="mt-1 break-all text-xs">{formData.evidenceUrl}</div>
                  </div>
                )}

                {evidenceType === 'file' && evidenceFile && (
                  <div className="pt-2 border-t border-border">
                    <span className="text-muted-foreground">Chứng từ (File):</span>
                    <div className="mt-1 flex items-center gap-2">
                      <Receipt className="h-4 w-4" />
                      <span>{evidenceFile.name}</span>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="bg-amber-50 border border-amber-200 rounded-xl p-3">
              <p className="typography-body-small text-amber-800">
                ⚠️ Hành động này sẽ thanh toán giao dịch và không thể hoàn thành.
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
                className="inline-flex items-center justify-center gap-1.5 h-8 px-3 rounded bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:pointer-events-none"
              >
                {isLoading ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    Đang xử lý...
                  </>
                ) : (
                  <>
                    <CheckCircle className="w-4 h-4" />
                    Xác nhận thanh toán
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
export function SettleTransactionModal({
  isOpen: propIsOpen,
  onClose: propOnClose,
  transaction,
}: SettleTransactionModalProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();

  const modalId = searchParams.get('modal');
  const isOpenViaUrl = modalId === 'settle_transaction';

  const isOpen = isOpenViaUrl || propIsOpen;

  const handleClose = () => {
    if (isOpenViaUrl) {
      closeModal();
    } else {
      propOnClose();
    }
  };

  return (
    <SettleTransactionModalComponent
      isOpen={isOpen}
      onClose={handleClose}
      transaction={transaction}
    />
  );
}

export default SettleTransactionModal;
