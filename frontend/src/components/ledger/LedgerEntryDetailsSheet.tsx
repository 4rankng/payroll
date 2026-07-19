import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { Textarea } from '@/components/ui/textarea';
import { FileText, User as UserIcon, Calendar, Building, X, Hash, RotateCcw } from 'lucide-react';
import { formatDate, formatDateTime } from '@/utils/formatters';
import { ledgerService } from '@/services/api/ledger.service';
import { projectService } from '@/services/api/project.service';
import { useLedgerEntry, useReverseLedgerEntry } from '@/hooks/ledger/useLedgerEntries';
import { useLedgerMetadata } from '@/hooks/ledger/useLedgerMetadata';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useUsersByIds } from '@/hooks/api/useUsers';
import { getUserFullName } from '@/utils/userHelpers';
import { authManager } from '@/lib/auth';
import { toast } from '@/components/ui/sonner';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import type { LedgerEntry } from '@/types/api/financial.types';
import type { Project } from '@/types/api/project.types';
import type { User } from '@/types/user';
import type { ModalConfig } from '@/types/modal-config.types';

export const modalConfig: ModalConfig = {
  id: 'ledger_entry_details',
  name: 'Chi tiết bút toán',
  description: 'Xem chi tiết thông tin bút toán',
  category: 'report',
  permissions: {
    action: 'read',
    subject: 'LedgerEntry',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id'],
    example: '?modal=ledger_entry_details&id=123',
    validateParams: (params) => {
      if (!params || Object.keys(params).length === 0) return true;
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  requiresAuth: true,
  encryptData: false,
};

interface LedgerEntryDetailsSheetProps {
  entry?: LedgerEntry | null;
  id?: string;
  isOpen: boolean;
  onClose: () => void;
}

interface LedgerEntryDetailsSheetComponentProps extends LedgerEntryDetailsSheetProps {
  entry: LedgerEntry | null;
}

function LedgerEntryDetailsSheetComponent({
  isOpen,
  onClose,
  entry,
  isLoading = false,
  error = null,
}: LedgerEntryDetailsSheetComponentProps & { isLoading?: boolean; error?: Error | null }) {
  const [projectName, setProjectName] = useState<string>('');
  const [showReverseDialog, setShowReverseDialog] = useState(false);
  const [reverseReason, setReverseReason] = useState('');
  const reverseEntry = useReverseLedgerEntry();
  const { data: accountMetadata } = useLedgerMetadata();

  // Batch user lookup for created_by
  const userIds = useMemo(() => {
    return entry?.created_by ? [entry.created_by] : [];
  }, [entry?.created_by]);

  const { data: userMap } = useUsersByIds(userIds);

  // Get creator name using batch lookup
  const creatorName = useMemo(() => {
    return getUserFullName(entry?.created_by, userMap);
  }, [entry?.created_by, userMap]);

  // Get project name from API
  useEffect(() => {
    const getProjectName = async () => {
      if (!entry?.project_id) {
        setProjectName('');
        return;
      }

      // Check permission before fetching projects
      const userRole = authManager.getUserRole();
      if (userRole !== 'admin' && userRole !== 'partner') {
        setProjectName(`ID #${entry.project_id}`);
        return;
      }

      try {
        const projects = await projectService.getProjects({ page: 1, pageSize: 1000 });
        const project = projects.data.find(p => p.id === entry.project_id);
        setProjectName(project?.name || `ID #${entry.project_id}`);
      } catch (error) {
        setProjectName(`ID #${entry.project_id}`);
      }
    };

    getProjectName();
  }, [entry?.project_id]);

  const getAccountBadge = useCallback((account: string) => {
    const variants = {
      cash: 'bg-emerald-100 text-emerald-800',
      receivable: 'bg-blue-100 text-blue-800',
      payable: 'bg-red-100 text-red-800',
      revenue: 'bg-teal-100 text-teal-800',
      expense: 'bg-orange-100 text-orange-800',
    };

    // Use backend label only - no fallback
    const meta = accountMetadata?.find(m => m.value === account);
    if (!meta) {
      console.error(`Missing metadata for account: ${account}`);
      return (
        <Badge className="bg-red-100 text-red-800">
          ERROR: {account}
        </Badge>
      );
    }

    return (
      <Badge className={variants[account as keyof typeof variants] || 'bg-muted text-gray-800'}>
        {meta.label}
      </Badge>
    );
  }, [accountMetadata]);

  const handleReverse = useCallback(async () => {
    if (!entry || !reverseReason.trim()) return;

    try {
      await reverseEntry.mutateAsync({ id: entry.id, reason: reverseReason.trim() });
      toast({
        title: 'Thành công',
        description: 'Bút toán đã được đảo ngược',
      });
      setShowReverseDialog(false);
      setReverseReason('');
      onClose();
    } catch (error: unknown) {
      toast({
        title: 'Lỗi',
        description: error instanceof Error ? error.message : 'Không thể đảo ngược bút toán',
        variant: 'destructive',
      });
    }
  }, [entry, reverseReason, reverseEntry, onClose]);

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={onClose}
        avatar={{
          custom: (
            <div className="flex flex-col gap-2 w-full">
              <div className="flex items-center justify-between gap-2">
                <div className={`text-lg font-bold ${
                  entry && entry.credit > entry.debit ? 'text-green-600' : 'text-red-600'
                }`}>
                  {entry ? `${entry.credit > entry.debit ? '+' : ''}${ledgerService.formatCurrencyShort(Math.max(entry.debit, entry.credit))}` : '--'}
                </div>
                {entry && getAccountBadge(entry.account)}
              </div>
              <div className="space-y-1">
                <h1 className="text-sm font-medium line-clamp-2 leading-tight">
                  {entry?.party || 'Chi tiết bút toán'}
                </h1>
                {entry && (
                  <div className="flex items-center gap-1 text-xs text-muted-foreground">
                    <Calendar className="h-3 w-3 flex-shrink-0" />
                    <span>{formatDate(entry.date)}</span>
                  </div>
                )}
              </div>
            </div>
          )
        }}
        footer={
          entry && (
            <div className="grid grid-cols-2 gap-3 w-full">
              <Button variant="outline" onClick={onClose} className="w-full">
                <X className="w-4 h-4 mr-2" />
                Đóng
              </Button>
              <Button
                variant="destructive"
                onClick={() => setShowReverseDialog(true)}
                disabled={reverseEntry.isPending}
                className="w-full"
              >
                <RotateCcw className="w-4 h-4 mr-2" />
                Đảo ngược
              </Button>
            </div>
          )
        }
      >
        {error ? (
          <div className="text-center py-8">
            <div className="text-destructive mb-4">
              <FileText className="h-12 w-12 mx-auto mb-4 opacity-40" />
            </div>
            <h3 className="typography-title-large text-destructive mb-2">
              Không thể tải dữ liệu
            </h3>
            <p className="text-muted-foreground mb-4">
              {error.message || 'Đã xảy ra lỗi khi tải thông tin bút toán.'}
            </p>
            <Button
              variant="outline"
              onClick={() => window.location.reload()}
              className="flex items-center gap-2"
            >
              <RotateCcw className="h-4 w-4" />
              Thử lại
            </Button>
          </div>
        ) : isLoading || !entry ? (
          <div className="space-y-4">
            <Skeleton className="h-6 w-40" />
            <div className="grid grid-cols-2 gap-3">
              <Skeleton className="h-12" />
              <Skeleton className="h-12" />
            </div>
            <Skeleton className="h-20" />
            <div className="grid grid-cols-3 gap-3">
              <Skeleton className="h-16" />
              <Skeleton className="h-16" />
              <Skeleton className="h-16" />
            </div>
          </div>
        ) : (
          <div className="space-y-3 overflow-hidden">
            {/* Basic Info */}
            <div className="space-y-2">
              <div className="flex items-center gap-2 p-2 bg-muted/20 rounded">
                <Hash className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="text-xs text-muted-foreground">ID</div>
                  <div className="font-medium text-sm">#{entry.id}</div>
                </div>
              </div>
              <div className="flex items-start gap-2 p-2 bg-muted/20 rounded">
                <Building className="h-4 w-4 text-muted-foreground flex-shrink-0 mt-0.5" />
                <div className="min-w-0 flex-1">
                  <div className="text-xs text-muted-foreground">Bên liên quan</div>
                  <div className="font-medium text-sm line-clamp-2 leading-tight">{entry.party}</div>
                </div>
              </div>
            </div>

            {/* Description */}
            <div className="flex items-start gap-2 p-2 bg-muted/20 rounded">
              <FileText className="h-4 w-4 text-muted-foreground mt-0.5 flex-shrink-0" />
              <div className="min-w-0 flex-1">
                <div className="text-xs text-muted-foreground">Diễn giải</div>
                <div className="text-sm font-medium line-clamp-3 leading-tight">{entry.description}</div>
              </div>
            </div>

            {/* Project Info */}
            {entry.project_id && projectName && (
              <div className="flex items-center gap-2 p-2 bg-muted/20 rounded">
                <UserIcon className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="text-xs text-muted-foreground">Dự án</div>
                  <div className="text-sm font-medium line-clamp-2 leading-tight">{projectName}</div>
                </div>
              </div>
            )}

            {/* Financial Info - Compact */}
            <div className="bg-muted/30 p-3 rounded-xl">
              <div className="text-xs text-muted-foreground mb-2">Thông tin tài chính</div>
              <div className="grid grid-cols-1 gap-2 text-left sm:grid-cols-3 sm:text-center">
                <div className="rounded bg-red-50 p-2 text-xs">
                  <div className="text-muted-foreground">Nợ</div>
                  <div className="font-semibold text-red-600 leading-tight">
                    {ledgerService.formatCurrencyShort(entry.debit)}
                  </div>
                </div>
                <div className="rounded bg-green-50 p-2 text-xs">
                  <div className="text-muted-foreground">Có</div>
                  <div className="font-semibold text-green-600 leading-tight">
                    {ledgerService.formatCurrencyShort(entry.credit)}
                  </div>
                </div>
                <div className="rounded bg-blue-50 p-2 text-xs">
                  <div className="text-muted-foreground">Số dư</div>
                  <div className={`font-semibold leading-tight ${
                    entry.balance >= 0 ? 'text-blue-600' : 'text-red-600'
                  }`}>
                    {ledgerService.formatCurrencyShort(entry.balance)}
                  </div>
                </div>
              </div>
            </div>

            {/* System Info - Compact */}
            <div className="bg-muted/30 p-3 rounded-xl">
              <div className="text-xs text-muted-foreground mb-2">Thông tin hệ thống</div>
              <div className="space-y-1.5 text-xs">
                <div className="flex justify-between items-start gap-2">
                  <span className="text-muted-foreground flex-shrink-0">Người tạo:</span>
                  <span className="font-medium text-right min-w-0 break-words">{creatorName || `ID #${entry.created_by}`}</span>
                </div>
                <div className="flex justify-between items-start gap-2">
                  <span className="text-muted-foreground flex-shrink-0">Ngày tạo:</span>
                  <span className="font-medium text-right min-w-0">{formatDateTime(entry.created_at)}</span>
                </div>
                <div className="flex justify-between items-start gap-2">
                  <span className="text-muted-foreground flex-shrink-0">Cập nhật:</span>
                  <span className="font-medium text-right min-w-0">{formatDateTime(entry.updated_at)}</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </SlideSheetTemplate>

      {/* Reverse Confirmation Dialog */}
      <ConfirmDialog
        open={showReverseDialog}
        onOpenChange={(open) => {
          setShowReverseDialog(open);
          if (!open) {
            setReverseReason('');
          }
        }}
        onConfirm={handleReverse}
        title="Đảo ngược bút toán"
        description={
          <div className="space-y-3">
            <p>Việc đảo ngược sẽ tạo một bút toán mới với các giá trị nợ/có ngược lại để hủy bỏ bút toán này.</p>
            <div className="space-y-2">
              <label className="text-sm font-medium">Lý do đảo ngược *</label>
              <Textarea
                value={reverseReason}
                onChange={(e) => setReverseReason(e.target.value)}
                placeholder="Nhập lý do đảo ngược bút toán..."
                className="min-h-[80px]"
              />
            </div>
          </div>
        }
        confirmText="Đảo ngược"
        cancelText="Hủy"
        confirmVariant="destructive"
        loading={reverseEntry.isPending}
        disabled={!reverseReason.trim()}
      />
    </>
  );
}

/**
 * Container component that provides URL synchronization for LedgerEntryDetailsSheet
 * Uses URL as single source of truth for state management
 */
export function LedgerEntryDetailsSheet({
  entry: propEntry,
  id: propId,
  isOpen,
  onClose
}: LedgerEntryDetailsSheetProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();

  // Get entry ID from prop or URL
  const entryId = propId || searchParams.get('id');

  // Fetch entry data if ID is provided but no entry prop
  const { data: fetchedEntry, isLoading, error } = useLedgerEntry(
    entryId ? parseInt(entryId) : 0,
    isOpen && !!entryId && !propEntry
  );


  // Use prop entry or fetched entry
  const entry = propEntry || fetchedEntry;

  // Close modal using centralized navigation or provided onClose
  const handleClose = useCallback(() => {
    if (searchParams.has('modal')) {
      closeModal();
    } else {
      onClose();
    }
  }, [searchParams, closeModal, onClose]);

  return (
    <LedgerEntryDetailsSheetComponent
      isOpen={isOpen}
      onClose={handleClose}
      entry={entry}
      isLoading={isLoading}
      error={error}
    />
  );
}

export default LedgerEntryDetailsSheet;
