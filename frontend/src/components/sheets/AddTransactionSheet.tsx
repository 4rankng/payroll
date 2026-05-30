import { useState, useEffect, useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ButtonGroup } from '@/components/ui/button-group';
import { FileUpload } from '@/components/ui/FileUpload';
import { UrlInput } from '@/components/ui/UrlInput';
import { Skeleton } from '@/components/ui/skeleton';
import { X, Receipt } from 'lucide-react';
import { transactionService, type TransactionType, type TransactionStatus, type TransactionMetadata, type TransactionMetadataItem } from '@/services/api/transaction.service';
import { useUploadAsset } from '@/hooks/api/useAssets';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useAuth, useMetadata } from '@/contexts';
import type { ModalConfig } from '@/types/modal-config.types';
import { useCreateTransaction } from '@/hooks/transactions/useCreateTransaction';
import { UserSelector } from '@/components/ui/user-selector';

export const modalConfig: ModalConfig = {
  id: 'transaction-form',
  name: 'Transaction Form',
  description: 'Create a new transaction',
  category: 'ledger',
  permissions: {
    action: 'create',
    subject: 'Transaction',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['amount', 'party', 'description', 'transaction_type', 'status', 'evidenceUrl'],
    example: '?modal=add_transaction&amount=100000&party=Supplier&transaction_type=expense&status=pending',
  },
  requiresAuth: true,
  encryptData: false,
};

interface TransactionFormProps {
  isOpen: boolean;
  onClose: () => void;
}

interface FormData {
  description: string;
  transaction_type: TransactionType;
  amount: string;
  party: string;
  status: TransactionStatus;
  evidenceUrl?: string;
  user_id?: string; // string for Select value; converted to number on submit
}

function TransactionFormComponent({
  isOpen,
  onClose
}: TransactionFormProps) {
  const createTransaction = useCreateTransaction();
  const { user } = useAuth();
  // Safe access to metadata; fall back if provider not mounted for non-admin roles
  let transactionMetadata: TransactionMetadata | undefined;
  let isLoadingTransactionMetadata = false;
  try {
    const md = useMetadata();
    transactionMetadata = md.transactionMetadata;
    isLoadingTransactionMetadata = md.isLoadingTransactionMetadata;
  } catch {
    transactionMetadata = undefined;
    isLoadingTransactionMetadata = false;
  }
  const [transactionType, setTransactionType] = useState<TransactionType>('expense');
  const [status, setStatus] = useState<TransactionStatus>('pending');
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null);
  const [evidenceType, setEvidenceType] = useState<'url' | 'file'>('url');
  const [fileUploadError, setFileUploadError] = useState<string | null>(null);
  const uploadAsset = useUploadAsset();

  const [searchParams] = useSearchParams();

  // Fallback options when metadata is unavailable (e.g., partner role)
  const fallbackTypeOptions = useMemo(() => ([
    { value: 'expense', label: 'Chi phí' },
    { value: 'revenue', label: 'Doanh thu' },
    { value: 'capital', label: 'Góp vốn' },
  ]), []);

  const fallbackStatusOptions = useMemo(() => ([
    { value: 'settled', label: 'Đã TT' },
    { value: 'pending', label: 'Chờ TT' },
  ]), []);

  const transactionTypeOptions = useMemo(() => {
    return (transactionMetadata?.transaction_types?.map((t: TransactionMetadataItem) => ({ value: t.type, label: t.label }))
      || fallbackTypeOptions);
  }, [transactionMetadata?.transaction_types, fallbackTypeOptions]);

  const statusOptions = useMemo(() => {
    return (
      (transactionMetadata?.statuses?.filter((s: TransactionMetadataItem) => s.type !== 'partially_settled')
        .map((s: TransactionMetadataItem) => ({
          value: s.type,
          label: transactionService.getStatusDisplay(s.type, transactionMetadata),
        })))
    );
  }, [transactionMetadata]);

  // Parse URL parameters
  const urlParams = useMemo(() => {
    const validateAmount = (amount: string | null): string => {
      if (!amount) return '';
      const numericAmount = amount.replace(/[^0-9]/g, '');
      if (numericAmount && !isNaN(Number(numericAmount))) {
        return Number(numericAmount).toLocaleString('vi-VN');
      }
      return '';
    };

    return {
      amount: validateAmount(searchParams.get('amount')),
      party: searchParams.get('party') || '',
      description: searchParams.get('description') || '',
      transaction_type: (searchParams.get('transaction_type') as TransactionType) || 'expense',
      status: (searchParams.get('status') as TransactionStatus) || 'pending',
      evidenceUrl: searchParams.get('evidenceUrl') || '',
    };
  }, [searchParams]);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    setError,
    clearErrors,
    formState: { errors },
  } = useForm<FormData>({
    defaultValues: {
      description: '',
      transaction_type: 'expense',
      amount: '',
      party: '',
      status: 'pending',
      evidenceUrl: '',
      user_id: ''
    },
  });

  const watchedAmount = watch('amount');
  const selectedUserId = watch('user_id');

  // Initialize form when opening
  useEffect(() => {
    if (isOpen) {
      reset({
        description: urlParams.description,
        transaction_type: urlParams.transaction_type,
        amount: urlParams.amount,
        party: urlParams.party,
        status: urlParams.status,
        evidenceUrl: urlParams.evidenceUrl,
      });
      setTransactionType(urlParams.transaction_type);
      setStatus(urlParams.status);
      setEvidenceFile(null);
      setFileUploadError(null);
      setEvidenceType(urlParams.evidenceUrl ? 'url' : 'file');
    }
  }, [isOpen, reset, urlParams]);

  const onSubmit = async (data: FormData) => {
    try {
      // Convert Vietnamese formatted amount to number
      let cleanAmount = data.amount.trim();

      // Parse Vietnamese number format
      const vietnameseFormatRegex = /^\d{1,3}(\.\d{3})*,\d+$/;
      const vietnameseWholeNumberRegex = /^\d{1,3}(\.\d{3})+$/;

      if (vietnameseFormatRegex.test(cleanAmount)) {
        cleanAmount = cleanAmount.replace(/\./g, '').replace(',', '.');
      } else if (vietnameseWholeNumberRegex.test(cleanAmount)) {
        cleanAmount = cleanAmount.replace(/\./g, '');
      } else if (cleanAmount.includes(',') && !cleanAmount.includes('.')) {
        cleanAmount = cleanAmount.replace(',', '.');
      } else if (cleanAmount.includes('.') && !cleanAmount.includes(',')) {
        const parts = cleanAmount.split('.');
        if (parts.length === 2 && parts[1].length <= 2 && /^\d+$/.test(parts[1])) {
          // Keep as is
        } else {
          cleanAmount = cleanAmount.replace(/\./g, '');
        }
      }

      const amount = parseFloat(cleanAmount);

      if (isNaN(amount) || amount <= 0) {
        setError('amount', {
          type: 'manual',
          message: 'Số tiền phải lớn hơn 0'
        });
        return;
      }

      clearErrors('amount');

      // Capital requires selecting a contributing user
      if (transactionType === 'capital') {
        if (!data.user_id) {
          setError('user_id', {
            type: 'manual',
            message: 'Vui lòng chọn người góp vốn'
          });
          return;
        }
        clearErrors('user_id');
      }

      // Handle evidence
      let assetId: number | undefined;
      let evidenceUrl: string | undefined;

      if (evidenceType === 'file' && evidenceFile) {
        try {
          setFileUploadError(null);
          const uploadedAsset = await uploadAsset.mutateAsync({
            data: {
              file: evidenceFile,
              upload_type: 'ledger_evidence',
            }
          });

          if (uploadedAsset?.data?.id) {
            assetId = uploadedAsset.data.id;
          } else {
            throw new Error('Upload failed: No asset ID returned');
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error';
          setFileUploadError(`Không thể tải lên file: ${errorMessage}`);
          return;
        }
      } else if (evidenceType === 'url' && data.evidenceUrl?.trim()) {
        evidenceUrl = data.evidenceUrl.trim();
      }

      // Create transaction
      await createTransaction.mutateAsync({
        description: data.description,
        transaction_type: transactionType,
        amount: amount,
        party: data.party,
        status: status,
        ...(transactionType === 'capital' && data.user_id
          ? { user_id: parseInt(data.user_id, 10) }
          : {}),
        ...(evidenceUrl && { url: evidenceUrl }),
        ...(assetId && { asset_id: assetId }),
      });

      onClose();
    } catch (error: unknown) {
      console.error('Form submission error:', error);
      if (error instanceof Error && error.message.includes('asset')) {
        setFileUploadError(`Lỗi tải file: ${error.message}`);
      }
    }
  };

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
    clearErrors('evidenceUrl');
  };

  const handleRemoveFile = () => {
    setEvidenceFile(null);
    setFileUploadError(null);
  };

  const isLoading = createTransaction.isPending || uploadAsset.isPending;

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      avatar={{
        custom: (
          <div className="flex items-center gap-4 flex-1 min-w-0">
            <div className="h-12 w-12 rounded-xl bg-blue-100 flex items-center justify-center flex-shrink-0">
              <Receipt className="h-6 w-6 text-blue-600" />
            </div>
            <div className="space-y-1 flex-1 min-w-0">
              <h1 className="typography-headline-medium text-base sm:text-lg font-medium">
                Thêm Giao Dịch Mới
              </h1>
              <p className="typography-body-small text-muted-foreground">
                Ghi lại chi phí hoặc doanh thu
              </p>
            </div>
          </div>
        )
      }}
      footer={
        <div className="grid grid-cols-2 gap-3 w-full">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={isLoading}
            className="w-full"
          >
            <X className="w-4 h-4 mr-2" />
            Đóng
          </Button>
          <Button
            type="submit"
            form="transaction-form"
            disabled={isLoading}
            variant="default"
            className="w-full"
          >
            {isLoading ? 'Đang xử lý...' : 'Lưu'}
          </Button>
        </div>
      }
    >
      <form id="transaction-form" onSubmit={handleSubmit(onSubmit)} className="space-y-3.5 mt-0">
        <div className="space-y-3.5">
          {/* Transaction Type */}
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-muted-foreground">Loại Giao Dịch *</Label>
              {isLoadingTransactionMetadata ? (
                <Skeleton className="w-full h-9" />
              ) : (
                <ButtonGroup
                options={transactionTypeOptions}
                value={transactionType}
                onChange={(value) => {
                  setTransactionType(value as TransactionType);
                  setValue('transaction_type', value as TransactionType);
                  // Reset counterpart fields when switching types
                  if (value === 'capital') {
                    setValue('party', '');
                  } else {
                    setValue('user_id', '');
                  }
                }}
                fullWidth
              />
              )}
            </div>

          {/* Amount and Settlement Status */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="amount" className="text-xs font-medium text-muted-foreground">Số Tiền *</Label>
              <div className="relative">
                <Input
                  id="amount"
                  value={watchedAmount}
                  onChange={handleAmountChange}
                  placeholder="0"
                  className={`pr-12 text-right h-9 ${errors.amount ? 'border-red-500' : ''}`}
                />
                <span className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground text-xs">
                  đ
                </span>
              </div>
              {errors.amount && (
                <p className="text-xs text-financial-negative mt-1">{errors.amount.message}</p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground">Trạng Thái *</Label>
              {isLoadingTransactionMetadata ? (
                <Skeleton className="w-full h-9" />
              ) : (
                <ButtonGroup
                  options={statusOptions}
                  value={status}
                  onChange={(value) => {
                  setStatus(value as TransactionStatus);
                  setValue('status', value as TransactionStatus);
                }}
                fullWidth
              />
              )}
            </div>
          </div>

          {/* Counterparty input: party text or user dropdown for capital */}
          {transactionType === 'capital' ? (
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-muted-foreground">Người góp vốn *</Label>
              <UserSelector
                value={selectedUserId || ''}
                onValueChange={(val) => {
                  setValue('user_id', val, { shouldValidate: true });
                }}
                onUserChange={(user) => {
                  // Keep party synced to selected user's name for API consistency
                  setValue('party', user?.fullname || '');
                  if (user) {
                    clearErrors('user_id');
                  }
                }}
                placeholder="Chọn người góp vốn..."
                className="h-9"
                filterRole="admin"
              />
              {errors.user_id && (
                <p className="text-xs text-financial-negative mt-1">{String(errors.user_id.message)}</p>
              )}
            </div>
          ) : (
            <div className="space-y-1.5">
              <Label htmlFor="party" className="text-xs font-medium text-muted-foreground">
                {transactionType === 'expense' ? 'Nhà cung cấp / Người nhận' : 'Khách hàng / Người trả'} *
              </Label>
              <Input
                id="party"
                {...register('party', {
                  required: 'Đối tượng là bắt buộc'
                })}
                placeholder={transactionType === 'expense' ? 'Tên nhà cung cấp' : 'Tên khách hàng'}
                className={`h-9 ${errors.party ? 'border-red-500' : ''}`}
              />
              {errors.party && (
                <p className="text-xs text-financial-negative mt-1">{errors.party.message}</p>
              )}
            </div>
          )}

          {/* Description */}
          <div className="space-y-1.5">
            <Label htmlFor="description" className="text-xs font-medium text-muted-foreground">Diễn Giải *</Label>
            <Textarea
              id="description"
              {...register('description', {
                required: 'Diễn giải là bắt buộc'
              })}
              placeholder="Mô tả chi tiết về giao dịch"
              rows={3}
              className={`resize-none text-sm ${errors.description ? 'border-red-500' : ''}`}
            />
            {errors.description && (
              <p className="text-xs text-financial-negative mt-1">{errors.description.message}</p>
            )}
          </div>
        </div>

        {/* Evidence */}
        <div className="space-y-2.5">
          <Label className="text-xs font-medium text-muted-foreground">Chứng từ</Label>

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
              clearErrors('evidenceUrl');
              setFileUploadError(null);
            }}
            fullWidth
          />

          {evidenceType === 'url' && (
            <div className="space-y-2">
              <UrlInput
                value={watch('evidenceUrl') || ''}
                onChange={(url) => {
                  setValue('evidenceUrl', url);
                }}
                placeholder="https://drive.google.com/file/d/abc123/view"
                label="Link chứng từ"
              />
            </div>
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
                    <span className="typography-caption text-green-700">
                      Đã chọn: {evidenceFile.name}
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
                <p className="typography-body-small text-financial-negative mt-1">
                  {fileUploadError}
                </p>
              )}
            </div>
          )}
        </div>
      </form>
    </SlideSheetTemplate>
  );
}

/**
 * Container component that provides URL synchronization
 */
export function AddTransactionSheet({
  isOpen: propIsOpen,
  onClose: propOnClose
}: TransactionFormProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();

  const modalId = searchParams.get('modal');
  const isOpenViaUrl = modalId === 'add_transaction';

  const isOpen = isOpenViaUrl || propIsOpen;

  const handleClose = () => {
    if (isOpenViaUrl) {
      closeModal();
    } else {
      propOnClose();
    }
  };

  return (
    <TransactionFormComponent
      isOpen={isOpen}
      onClose={handleClose}
    />
  );
}

export default AddTransactionSheet;
