import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { Textarea } from '@/components/ui/textarea';
import { FileText, User as UserIcon, Calendar, Building, Hash, RotateCcw } from 'lucide-react';
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
      cash: 'border-success/20 bg-success/10 text-success',
      receivable: 'border-info/20 bg-info/10 text-info',
      payable: 'border-destructive/20 bg-destructive/10 text-destructive',
      revenue: 'border-primary/20 bg-primary/10 text-primary',
      expense: 'border-warning/20 bg-warning/10 text-warning',
    };

    // Use backend label only - no fallback
    const meta = accountMetadata?.find(m => m.value === account);
    if (!meta) {
      console.error(`Missing metadata for account: ${account}`);
      return (
        <Badge className="border-destructive/20 bg-destructive/10 text-destructive">
          ERROR: {account}
        </Badge>
      );
    }

    return (
      <Badge
        variant="outline"
        className={variants[account as keyof typeof variants] || 'border-border bg-muted text-muted-foreground'}
      >
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
            <div className="w-full min-w-0">
              <div className="flex flex-col items-start gap-2 min-[360px]:flex-row min-[360px]:justify-between min-[360px]:gap-3">
                <div className="min-w-0">
                  <p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                    Giá trị bút toán
                  </p>
                  <div className={`mt-1 max-w-full whitespace-nowrap font-financial text-xl font-bold leading-none tabular-nums min-[360px]:text-2xl ${
                    entry && entry.credit > entry.debit ? 'text-success' : 'text-destructive'
                  }`}>
                    {entry
                      ? `${entry.credit > entry.debit ? '+' : ''}${ledgerService.formatCurrencyShort(Math.max(entry.debit, entry.credit))}`
                      : '--'}
                  </div>
                </div>
                <div className="shrink-0">
                  {entry && getAccountBadge(entry.account)}
                </div>
              </div>
              <div className="mt-3 flex min-w-0 items-center gap-2 text-sm">
                <span className="truncate font-semibold text-foreground">
                  {entry?.party || 'Chi tiết bút toán'}
                </span>
                {entry && (
                  <>
                    <span className="h-1 w-1 shrink-0 rounded-full bg-muted-foreground/40" aria-hidden="true" />
                    <span className="flex shrink-0 items-center gap-1 text-muted-foreground">
                      <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
                      {formatDate(entry.date)}
                    </span>
                  </>
                )}
              </div>
            </div>
          )
        }}
        compact
        footer={
          entry && (
            <div className="grid w-full grid-cols-2 gap-3">
              <Button variant="outline" onClick={onClose} className="h-11 w-full">
                Đóng
              </Button>
              <Button
                variant="destructive"
                onClick={() => setShowReverseDialog(true)}
                disabled={reverseEntry.isPending}
                className="h-11 w-full"
              >
                <RotateCcw className="mr-2 h-4 w-4" aria-hidden="true" />
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
          <div className="space-y-3">
            <section
              className="ct-card ct-card-border overflow-hidden border-border/80 bg-card text-card-foreground shadow-none"
              aria-labelledby="ledger-entry-overview"
            >
              <div className="ct-card-body gap-0 p-0">
                <h2 id="ledger-entry-overview" className="sr-only">
                  Thông tin bút toán
                </h2>
                <div className="divide-y divide-border/60">
                  <div className="grid grid-cols-[1.5rem_minmax(0,1fr)_auto] items-center gap-3 px-4 py-3">
                    <Hash className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    <span className="text-sm text-muted-foreground">Mã bút toán</span>
                    <span className="font-semibold tabular-nums text-foreground">#{entry.id}</span>
                  </div>
                  <div className="grid grid-cols-[1.5rem_minmax(0,1fr)] items-start gap-3 px-4 py-3">
                    <Building className="mt-0.5 h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    <div className="min-w-0">
                      <p className="text-xs text-muted-foreground">Bên liên quan</p>
                      <p className="mt-0.5 break-words text-sm font-semibold text-foreground">
                        {entry.party || 'Chưa có thông tin'}
                      </p>
                    </div>
                  </div>
                  {entry.project_id && projectName && (
                    <div className="grid grid-cols-[1.5rem_minmax(0,1fr)] items-start gap-3 px-4 py-3">
                      <UserIcon className="mt-0.5 h-4 w-4 text-muted-foreground" aria-hidden="true" />
                      <div className="min-w-0">
                        <p className="text-xs text-muted-foreground">Dự án</p>
                        <p className="mt-0.5 break-words text-sm font-semibold text-foreground">
                          {projectName}
                        </p>
                      </div>
                    </div>
                  )}
                  <div className="grid grid-cols-[1.5rem_minmax(0,1fr)] items-start gap-3 px-4 py-3">
                    <FileText className="mt-0.5 h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    <div className="min-w-0">
                      <p className="text-xs text-muted-foreground">Diễn giải</p>
                      <p className="mt-0.5 whitespace-pre-wrap break-words text-sm font-medium leading-relaxed text-foreground">
                        {entry.description || 'Không có diễn giải'}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <section
              className="ct-card ct-card-border overflow-hidden border-border/80 bg-card text-card-foreground shadow-none"
              aria-labelledby="ledger-financial-info"
            >
              <div className="ct-card-body gap-3 p-4">
                <h2 id="ledger-financial-info" className="text-sm font-semibold text-foreground">
                  Thông tin tài chính
                </h2>
                <dl className="grid w-full grid-cols-2 overflow-hidden rounded-xl border border-border/70 bg-muted/20 min-[440px]:grid-cols-3">
                  <div className="ct-stat min-w-0 border-r border-border/60 px-3 py-3">
                    <dt className="ct-stat-title flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                      <span className="h-1.5 w-1.5 rounded-full bg-destructive" aria-hidden="true" />
                      Nợ
                    </dt>
                    <dd className="ct-stat-value mt-1 whitespace-nowrap text-xs font-bold tabular-nums text-destructive min-[360px]:text-sm">
                      {ledgerService.formatCurrencyShort(entry.debit)}
                    </dd>
                  </div>
                  <div className="ct-stat min-w-0 px-3 py-3 min-[440px]:border-r min-[440px]:border-border/60">
                    <dt className="ct-stat-title flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                      <span className="h-1.5 w-1.5 rounded-full bg-success" aria-hidden="true" />
                      Có
                    </dt>
                    <dd className="ct-stat-value mt-1 whitespace-nowrap text-xs font-bold tabular-nums text-success min-[360px]:text-sm">
                      {ledgerService.formatCurrencyShort(entry.credit)}
                    </dd>
                  </div>
                  <div className="ct-stat col-span-2 min-w-0 border-t border-border/60 px-3 py-3 min-[440px]:col-span-1 min-[440px]:border-t-0">
                    <dt className="ct-stat-title flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
                      <span className="h-1.5 w-1.5 rounded-full bg-info" aria-hidden="true" />
                      Số dư
                    </dt>
                    <dd className={`ct-stat-value mt-1 whitespace-nowrap text-xs font-bold tabular-nums min-[360px]:text-sm ${
                      entry.balance >= 0 ? 'text-info' : 'text-destructive'
                    }`}>
                      {ledgerService.formatCurrencyShort(entry.balance)}
                    </dd>
                  </div>
                </dl>
              </div>
            </section>

            <section
              className="ct-card ct-card-border border-border/80 bg-card text-card-foreground shadow-none"
              aria-labelledby="ledger-system-info"
            >
              <div className="ct-card-body gap-3 p-4">
                <h2 id="ledger-system-info" className="text-sm font-semibold text-foreground">
                  Thông tin hệ thống
                </h2>
                <dl className="space-y-2.5 text-sm">
                  <div className="flex items-start justify-between gap-4">
                    <dt className="shrink-0 text-muted-foreground">Người tạo</dt>
                    <dd className="min-w-0 break-words text-right font-medium text-foreground">
                      {creatorName || `ID #${entry.created_by}`}
                    </dd>
                  </div>
                  <div className="flex items-start justify-between gap-4">
                    <dt className="shrink-0 text-muted-foreground">Ngày tạo</dt>
                    <dd className="text-right font-medium tabular-nums text-foreground">
                      {formatDateTime(entry.created_at)}
                    </dd>
                  </div>
                  <div className="flex items-start justify-between gap-4">
                    <dt className="shrink-0 text-muted-foreground">Cập nhật</dt>
                    <dd className="text-right font-medium tabular-nums text-foreground">
                      {formatDateTime(entry.updated_at)}
                    </dd>
                  </div>
                </dl>
              </div>
            </section>
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
