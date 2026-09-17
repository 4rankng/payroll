import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useTransaction } from '@/hooks/transactions/useTransactions';
import { useTransactionMetadata } from '@/hooks/transactions/useTransactionMetadata';
import { useUsersByIds } from '@/hooks/api/useUsers';
import { getUserFullName } from '@/utils/userHelpers';
import { transactionService } from '@/services/api/transaction.service';
import { formatCurrency } from '@/utils/formatters';
import { formatSettlementBusinessDate, getLatestSettlementDate } from '@/utils/transactionHelpers';
import { assetService } from '@/services/api/asset.service';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Receipt, X, ExternalLink, Calendar, User, FileText, DollarSign, Link2, Download, Pencil, Trash2, Loader2 } from 'lucide-react';
import { Label } from '@/components/ui/label';
import { UrlInput } from '@/components/ui/UrlInput';
import { FileUpload } from '@/components/ui/FileUpload';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { useUploadAsset, useDownloadAsset } from '@/hooks/api/useAssets';
import { useUpdateTransactionEvidence } from '@/hooks/transactions/useUpdateTransactionEvidence';
import { useDeleteTransaction } from '@/hooks/transactions/useDeleteTransaction';
import { SettlementHistorySection } from '@/components/transaction/SettlementHistorySection';
import type { ModalConfig } from '@/types/modal-config.types';

export const modalConfig: ModalConfig = {
  id: 'transaction-details',
  name: 'Transaction Details',
  description: 'View and manage transaction details',
  category: 'ledger',
  permissions: {
    action: 'read',
    subject: 'Transaction',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id'],
    example: '?modal=transaction_details&id=123',
  },
  requiresAuth: true,
  encryptData: false,
};

interface TransactionDetailsSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

function TransactionDetailsSheetComponent({
  isOpen,
  onClose,
}: TransactionDetailsSheetProps) {
  const [searchParams] = useSearchParams();
  const { openModal } = useModalNavigation();
  const transactionId = searchParams.get('id');

  const { data: transaction, isLoading: isLoadingTransaction } = useTransaction(Number(transactionId));
  const { data: metadata } = useTransactionMetadata();

  // Batch user lookup for created_by and settled_by
  const userIds = useMemo(() => {
    const ids: number[] = [];
    if (transaction?.created_by) ids.push(transaction.created_by);
    if (transaction?.settled_by) ids.push(transaction.settled_by);
    return ids;
  }, [transaction?.created_by, transaction?.settled_by]);

  const { data: userMap, isLoading: isLoadingUsers } = useUsersByIds(userIds);

  const createdByName = getUserFullName(transaction?.created_by, userMap);
  const settledByName = getUserFullName(transaction?.settled_by, userMap);
  const paymentDate = useMemo(
    () => getLatestSettlementDate(transaction?.settlements),
    [transaction?.settlements],
  );

  const isLoading = isLoadingTransaction || isLoadingUsers;

  const updateEvidence = useUpdateTransactionEvidence();
  const uploadAsset = useUploadAsset();
  const downloadAsset = useDownloadAsset();
  const deleteTransaction = useDeleteTransaction();

  const [isEditingEvidence, setIsEditingEvidence] = useState(false);
  const [evidenceUrl, setEvidenceUrl] = useState('');
  const [selectedEvidenceFile, setSelectedEvidenceFile] = useState<File | null>(null);
  const [evidenceError, setEvidenceError] = useState<string | null>(null);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);

  useEffect(() => {
    if (!transaction) {
      return;
    }

    setEvidenceUrl(transaction.url ?? '');
    setSelectedEvidenceFile(null);
    setEvidenceError(null);
    setIsEditingEvidence(false);
    setIsDeleteDialogOpen(false);
  }, [transaction]);

  const isMutatingEvidence = useMemo(() => {
    return updateEvidence.isPending || uploadAsset.isPending;
  }, [updateEvidence.isPending, uploadAsset.isPending]);

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), "dd MMMM yyyy 'lúc' HH:mm", { locale: vi });
    } catch {
      return dateString;
    }
  };

  const getTransactionTypeBadge = (type: string) => {
    const variants = {
      revenue: 'bg-emerald-100 text-emerald-800',
      expense: 'bg-red-100 text-red-800',
      capital: 'bg-teal-100 text-teal-800',
      write_off: 'bg-amber-100 text-amber-800',
    };

    const label = transactionService.getTransactionTypeDisplay(type, metadata);

    return (
      <Badge className={variants[type as keyof typeof variants] || 'bg-slate-100 text-slate-800'}>
        {label}
      </Badge>
    );
  };

  const getStatusBadge = (status: string) => {
    const baseClassName = 'min-w-[80px] justify-center typography-label-small';

    const variants: Record<string, { className: string; variant?: 'outline' }> = {
      settled: { className: 'bg-emerald-600 text-white border-emerald-600', variant: undefined },
      pending: { className: 'border-yellow-600 text-yellow-900', variant: 'outline' },
      partially_settled: { className: 'border-teal-600 text-teal-800', variant: 'outline' },
    };

    const label = transactionService.getStatusDisplay(status, metadata);
    const badgeVariant = variants[status]?.variant;

    return (
      <Badge
        variant={badgeVariant}
        className={`${baseClassName} ${
          variants[status as keyof typeof variants]?.className || 'border-border text-slate-800'
        }`}
      >
        {label}
      </Badge>
    );
  };

  const hasEvidenceUrl = useMemo(() => {
    return Boolean(transaction?.url && transaction.url.trim().length > 0);
  }, [transaction?.url]);

  const hasEvidenceAsset = useMemo(() => {
    return Boolean(transaction?.asset_id && Number.isFinite(transaction.asset_id));
  }, [transaction?.asset_id]);

  const evidenceFileDisplay = useMemo(() => {
    if (transaction?.asset) {
      const displayName = assetService.getDisplayName(transaction.asset);
      return transaction.asset_id ? `${displayName}` : displayName;
    }

    if (hasEvidenceAsset) {
      return `File #${transaction?.asset_id}`;
    }

    return null;
  }, [hasEvidenceAsset, transaction?.asset, transaction?.asset_id]);

  const handleOpenEvidence = useCallback(() => {
    if (!hasEvidenceUrl || !transaction?.url) {
      return;
    }

    window.open(transaction.url, '_blank', 'noopener');
  }, [hasEvidenceUrl, transaction?.url]);

  const handleDownloadEvidence = useCallback(() => {
    if (!hasEvidenceAsset || !transaction?.asset_id) {
      return;
    }

    const filename = transaction.asset
      ? assetService.getDisplayName(transaction.asset)
      : `evidence_${transaction.id}`;

    downloadAsset.mutate({
      id: transaction.asset_id,
      filename,
    });
  }, [hasEvidenceAsset, transaction, downloadAsset]);

  const handleStartEvidenceEditing = useCallback(() => {
    if (!transaction) {
      return;
    }

    setEvidenceUrl(transaction.url ?? '');
    setSelectedEvidenceFile(null);
    setEvidenceError(null);
    setIsEditingEvidence(true);
  }, [transaction]);

  const handleCancelEvidenceEditing = useCallback(() => {
    setEvidenceUrl(transaction?.url ?? '');
    setSelectedEvidenceFile(null);
    setEvidenceError(null);
    setIsEditingEvidence(false);
  }, [transaction?.url]);

  const handleEvidenceUrlChange = useCallback((value: string) => {
    setEvidenceUrl(value);
  }, []);

  const handleFilesSelected = useCallback((files: File[]) => {
    setSelectedEvidenceFile(files[0] ?? null);
    setEvidenceError(null);
  }, []);

  const handleRemoveSelectedFile = useCallback(() => {
    setSelectedEvidenceFile(null);
    setEvidenceError(null);
  }, []);

  const handleSubmitEvidence = useCallback(async () => {
    if (!transaction) {
      return;
    }

    const trimmedUrl = evidenceUrl.trim();

    if (!trimmedUrl && !selectedEvidenceFile) {
      setEvidenceError('Vui lòng nhập link hoặc tải lên file chứng từ');
      return;
    }

    setEvidenceError(null);

    let uploadedAssetId: number | undefined;

    try {
      if (selectedEvidenceFile) {
        const uploadResponse = await uploadAsset.mutateAsync({
          data: {
            file: selectedEvidenceFile,
            upload_type: 'ledger_evidence',
          },
        });

        uploadedAssetId = uploadResponse.data?.id;

        if (!uploadedAssetId) {
          throw new Error('Không tìm thấy mã file sau khi tải lên');
        }
      }

      await updateEvidence.mutateAsync({
        id: transaction.id,
        ...(trimmedUrl ? { url: trimmedUrl } : {}),
        ...(typeof uploadedAssetId === 'number' ? { asset_id: uploadedAssetId } : {}),
      });

      setIsEditingEvidence(false);
      setSelectedEvidenceFile(null);

      if (trimmedUrl) {
        setEvidenceUrl(trimmedUrl);
      }
    } catch (error) {
      console.error('Cập nhật chứng từ thất bại:', error);

      if (!uploadedAssetId && selectedEvidenceFile) {
        const message = error instanceof Error ? error.message : 'Không thể tải lên file chứng từ';
        setEvidenceError(message);
      }
    }
  }, [evidenceUrl, selectedEvidenceFile, transaction, updateEvidence, uploadAsset]);

  const handleSettleClick = () => {
    if (!transaction) return;

    openModal('settle_transaction', {
      id: transaction.id.toString(),
    });
  };

  const handleReverseClick = () => {
    if (!transaction) return;

    openModal('reverse_transaction', {
      id: transaction.id.toString(),
    });
  };

  const handleConfirmDelete = useCallback(() => {
    if (!transaction) return;

    deleteTransaction.mutate(
      { id: transaction.id },
      {
        onSuccess: () => {
          setIsDeleteDialogOpen(false);
          onClose();
        },
      },
    );
  }, [transaction, deleteTransaction, onClose]);

  if (isLoading || !transaction) {
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
                  Chi tiết giao dịch
                </h1>
              </div>
            </div>
          )
        }}
      >
        <div className="flex items-center justify-center py-12">
          <p className="text-muted-foreground">Đang tải...</p>
        </div>
      </SlideSheetTemplate>
    );
  }

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      avatar={{
        custom: (
          <div className="flex items-center gap-4 flex-1 min-w-0">
            <div className={`h-12 w-12 rounded-xl flex items-center justify-center flex-shrink-0 ${
              transaction.transaction_type === 'revenue' ? 'bg-emerald-100' :
              transaction.transaction_type === 'write_off' ? 'bg-amber-100' : 'bg-red-100'
            }`}>
              <Receipt className={`h-6 w-6 ${
                transaction.transaction_type === 'revenue' ? 'text-emerald-600' :
                transaction.transaction_type === 'write_off' ? 'text-amber-600' : 'text-red-600'
              }`} />
            </div>
            <div className="space-y-1 flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="typography-headline-medium text-base sm:text-lg font-medium">
                  Chi tiết giao dịch
                </h1>
                {getTransactionTypeBadge(transaction.transaction_type)}
                {getStatusBadge(transaction.status)}
                {transaction.reversed_transaction_id && (
                  <Badge className="bg-muted text-gray-800">
                    Đã đảo ngược
                  </Badge>
                )}
              </div>
                <div className="flex flex-col gap-0.5 min-[380px]:flex-row min-[380px]:items-baseline">
                  <span className="typography-body-small">Mã giao dịch:</span>
                  <span className="typography-title-small break-all">{transaction.transaction_code}</span>
                </div>
            </div>
          </div>
        )
      }}
      footer={
        // One adaptive row: every action is an equal-width cell and the row
        // wraps only when the count genuinely needs it, so no action ends up
        // alone on a full-width line below the others.
        <div className="flex w-full flex-wrap items-center gap-2">
          {transaction.status === 'settled' && !transaction.reversed_transaction_id && (
            <Button type="button" variant="destructive" size="sm" onClick={handleReverseClick} className="min-h-11 flex-1 basis-0 px-3">
              Đảo ngược
            </Button>
          )}
          {transaction.status === 'pending' && !transaction.reversed_transaction_id && (
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => setIsDeleteDialogOpen(true)}
              disabled={deleteTransaction.isPending}
              className="min-h-11 flex-1 basis-0 px-3"
            >
              <Trash2 className="h-4 w-4" />
              Hủy giao dịch
            </Button>
          )}
          {transaction.status === 'pending' && (
            <Button type="button" variant="success" size="sm" onClick={handleSettleClick} className="min-h-11 flex-1 basis-0 px-3">
              Thanh toán
            </Button>
          )}
          <Button type="button" variant="outline" size="sm" onClick={onClose} className="min-h-11 flex-1 basis-0 px-3">
            Đóng
          </Button>
        </div>
      }
    >
      <div className="space-y-4 mt-2">
        {/* Amount Section */}
        <div className="bg-gradient-subtle rounded-xl p-5 space-y-3 shadow-soft border border-border">          <div className="flex items-center gap-2">
            <DollarSign className="h-4 w-4 text-muted-foreground" />
            <span className="typography-label-medium text-muted-foreground uppercase">Số tiền</span>
          </div>
          <div className={`typography-display-large typography-currency break-words text-3xl sm:text-4xl ${
            transaction.transaction_type === 'revenue' ? 'text-financial-positive' :
            transaction.transaction_type === 'write_off' ? 'text-amber-600' : 'text-financial-negative'
          }`}>
            {formatCurrency(transaction.amount)}
          </div>
        </div>

        {/* Party and Description - 2 column layout */}
        <div className="grid grid-cols-1 gap-4 min-[420px]:grid-cols-2">
          <div className="space-y-2">

            <div className="flex items-center gap-2">
              <User className="h-4 w-4 text-muted-foreground" />
              <div className="typography-label-medium text-muted-foreground uppercase">
                {transaction.transaction_type === 'expense' ? 'Người nhận' :
                 transaction.transaction_type === 'revenue' ? 'Người trả' : 'Đối tượng'}
              </div>
            </div>
            <div className="typography-body-medium text-high-contrast font-semibold break-words">
              {transaction.party}
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-muted-foreground" />
              <div className="typography-label-medium text-muted-foreground uppercase">Diễn giải</div>
            </div>
            <div className="typography-body-medium text-readable whitespace-pre-wrap line-clamp-3">
              {transaction.description}
            </div>
          </div>
        </div>

        <Separator className="my-3" />

        {/* Metadata - 2 column grid */}
        <div className="space-y-3 bg-surface-secondary rounded-xl p-4 shadow-xs border border-border">          <div className="typography-label-medium text-muted-foreground uppercase font-semibold">Thông tin khác</div>
          <div className="grid grid-cols-1 gap-4 typography-body-small min-[420px]:grid-cols-2">

            <div className="space-y-1.5">
              <div className="typography-label-small text-muted-foreground">Người tạo</div>
              <div className="flex items-start gap-2">
                <User className="h-4 w-4 flex-shrink-0 text-slate-400" />
                <span className="typography-body-medium text-high-contrast font-medium break-words">{createdByName}</span>
              </div>
            </div>
            <div className="space-y-3">
              <div className="space-y-1.5">
                <div className="typography-label-small text-muted-foreground">Ngày tạo</div>
                <div className="flex items-start gap-2">
                  <Calendar className="h-4 w-4 flex-shrink-0 text-slate-400" />
                  <span className="typography-body-small text-medium-contrast break-words">{formatDate(transaction.created_at)}</span>
                </div>
              </div>
              {transaction.status === 'settled' && paymentDate && (
                <div className="space-y-1.5">
                  <div className="typography-label-small text-muted-foreground">Ngày thanh toán</div>
                  <div className="flex items-start gap-2">
                    <Calendar className="h-4 w-4 flex-shrink-0 text-slate-400" />
                    <span className="typography-body-small text-medium-contrast break-words">
                      {formatSettlementBusinessDate(paymentDate)}
                    </span>
                  </div>
                </div>
              )}

            </div>
            {transaction.status === 'settled' && transaction.settled_by && (
              <div className="space-y-1.5">
                <div className="typography-label-small text-muted-foreground">Người thanh toán</div>
                <div className="flex items-start gap-2">
                  <User className="h-4 w-4 flex-shrink-0 text-slate-400" />
                  <span className="typography-body-medium text-high-contrast font-medium break-words">{settledByName}</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Evidence section */}
        <div className="space-y-4 bg-card rounded-xl p-4 shadow-xs border border-border">          <div className="flex flex-col gap-3 min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 text-muted-foreground" />
              <span className="typography-label-medium text-muted-foreground uppercase">Chứng từ giao dịch</span>
            </div>
            <div className="flex items-center gap-2">
              {hasEvidenceAsset ? (
                <Button
                  type="button"
                  variant="info"
                  onClick={handleDownloadEvidence}
                  disabled={downloadAsset.isPending}
                  size="icon"
                  className="h-11 w-11"
                  title="Tải xuống chứng từ"
                >
                  <Download className="w-4 h-4" />
                </Button>
              ) : hasEvidenceUrl ? (
                <Button
                  type="button"
                  variant="info"
                  onClick={handleOpenEvidence}
                  size="icon"
                  className="h-11 w-11"
                  title="Xem chứng từ"
                >
                  <ExternalLink className="w-4 h-4" />
                </Button>
              ) : null}
              <Button
                type="button"
                variant={isEditingEvidence ? 'ghost' : 'outline'}
                size="icon"
                onClick={isEditingEvidence ? handleCancelEvidenceEditing : handleStartEvidenceEditing}
                disabled={isMutatingEvidence}
                className="h-11 w-11"
                title={isEditingEvidence ? 'Hủy cập nhật' : 'Cập nhật chứng từ'}
              >
                {isEditingEvidence ? (
                  <X className="h-4 w-4" />
                ) : (
                  <Pencil className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm text-medium-contrast break-all">
              <Link2 className="h-4 w-4 text-slate-400 flex-shrink-0" />
              {hasEvidenceUrl ? (
                <a
                  href={transaction?.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:underline"
                >
                  {transaction?.url}
                </a>
              ) : hasEvidenceAsset ? (
                <span className="text-medium-contrast">Đã tải lên chứng từ</span>
              ) : (
                <span className="text-medium-contrast">Chưa có link chứng từ</span>
              )}
            </div>
            <div className="flex items-start gap-2 text-sm text-medium-contrast break-all">
              <FileText className="h-4 w-4 text-slate-400 flex-shrink-0" />
              {hasEvidenceAsset ? (
                <span>{evidenceFileDisplay}</span>
              ) : !hasEvidenceUrl ? (
                <span>Chưa có file chứng từ</span>
              ) : null}
            </div>
          </div>

          {isEditingEvidence && (
            <div className="space-y-4 border-t border-border pt-4">
              <div className="space-y-2">
                <Label className="text-sm font-medium text-muted-foreground">Chứng từ giao dịch</Label>
                <UrlInput
                  value={evidenceUrl}
                  onChange={handleEvidenceUrlChange}
                  placeholder="https://drive.google.com/file/d/abc123"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-sm font-medium text-muted-foreground">Tải lên file mới</Label>
                {!selectedEvidenceFile ? (
                  <FileUpload
                    onFilesSelected={handleFilesSelected}
                    multiple={false}
                    maxFiles={1}
                  />
                ) : (
                  <div className="flex items-center justify-between gap-3 rounded-xl border border-border bg-muted/50 px-3 py-2">
                    <div className="flex min-w-0 items-center gap-2">
                      <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      <span className="text-sm font-medium text-foreground break-all">
                        {selectedEvidenceFile.name}
                      </span>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={handleRemoveSelectedFile}
                      className="h-11 w-11 shrink-0 px-0 text-muted-foreground hover:text-red-600"
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                )}
                <p className="text-xs text-muted-foreground">
                  Bạn có thể cập nhật link, file hoặc cả hai. Cần cung cấp ít nhất một trong hai.
                </p>
              </div>

              {evidenceError && (
                <p className="text-sm text-red-600">{evidenceError}</p>
              )}

              <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleCancelEvidenceEditing}
                  disabled={isMutatingEvidence}
                  className="min-h-11"
                >
                  Hủy
                </Button>
                <Button
                  type="button"
                  onClick={handleSubmitEvidence}
                  disabled={isMutatingEvidence}
                  className="min-h-11"
                >
                  Lưu chứng từ
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Settlement History Section */}
        <SettlementHistorySection transaction={transaction} />
      </div>

      {/* Cancel pending transaction confirmation */}
      <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Xác nhận hủy giao dịch</AlertDialogTitle>
            <AlertDialogDescription>
              Giao dịch và các bút toán sổ cái liên quan sẽ bị xóa vĩnh viễn.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteTransaction.isPending}>Quay lại</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={deleteTransaction.isPending}
            >
              {deleteTransaction.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Đang hủy...
                </>
              ) : (
                'Hủy giao dịch'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SlideSheetTemplate>
  );
}

/**
 * Container component that provides URL synchronization
 */
export function TransactionDetailsSheet({
  isOpen: propIsOpen,
  onClose: propOnClose,
}: TransactionDetailsSheetProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();

  const modalId = searchParams.get('modal');
  const isOpenViaUrl = modalId === 'transaction_details';

  const isOpen = isOpenViaUrl || propIsOpen;

  const handleClose = () => {
    if (isOpenViaUrl) {
      closeModal();
    } else {
      propOnClose();
    }
  };

  return (
    <TransactionDetailsSheetComponent
      isOpen={isOpen}
      onClose={handleClose}
    />
  );
}

export default TransactionDetailsSheet;
