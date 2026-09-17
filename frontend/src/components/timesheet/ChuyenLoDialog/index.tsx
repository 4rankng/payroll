import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Dialog, DialogContent, DialogClose, DialogFooter, DialogNavyHeader } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Label } from '@/components/ui/label';
import { ButtonGroup } from '@/components/ui/button-group';
import { Progress } from '@/components/ui/progress';
import {
  CreditCard,
  Download,
  FileDown,
  FileSpreadsheet,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Banknote,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatCurrencyFromString } from '@/utils/formatters';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useProjects } from '@/hooks/api/useProjects';
import type { Project } from '@/types/api/project.types';
import { walletService } from '@/services/api/wallet.service';
import { useDisbursementSettings } from '@/hooks/useDisbursementSettings';
import {
  useAutoBulkTransferConfig,
  useAutoBulkTransferStatus,
  useBulkTransferUploadHistoryById,
  useExportBulkTransfer,
  useInitiateAutoBulkTransfer,
} from '@/hooks/api/usePayrolls';
import { useBulkTransferExportForm } from '@/hooks/timesheet/useBulkTransferExportForm';
import { recordPendingExport } from '@/utils/timesheet/pendingExports';
import { BulkTransferDateRangeSection } from '../bulk-transfer-export/BulkTransferDateRangeSection';
import { BulkTransferFiltersSection } from '../bulk-transfer-export/BulkTransferFiltersSection';
import { MarkExternallyPaidDialog } from '../MarkExternallyPaidDialog';
import { generateBulkTransferHistoryPdf } from '@/utils/pdf/bulk-transfer-history';
import { sanitizeFilename } from '@/utils/file-naming';
import { dateToString } from '@/utils/dateHelpers';
import { showErrorNotification } from '@/utils/error-handler';

export type TransferMethod = 'manual' | 'provider';

interface ChuyenLoDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  preselectedEmployeeIds?: number[];
  estimatedTotal?: number | null;
}

export const ChuyenLoDialog = memo(function ChuyenLoDialog({
  open,
  onOpenChange,
  preselectedEmployeeIds,
  estimatedTotal,
}: ChuyenLoDialogProps) {
  const [activeTab, setActiveTab] = useState<TransferMethod>('manual');
  const [ninePayBatchId, setNinePayBatchId] = useState<string | null>(null);
  const [ninePayFileId, setNinePayFileId] = useState<number | null>(null);
  const [confirmed, setConfirmed] = useState(false);
  const [markExternallyPaidOpen, setMarkExternallyPaidOpen] = useState(false);
  const isMobile = useIsMobile();
  const queryClient = useQueryClient();

  const { data: disbursementSettings } = useDisbursementSettings();
  const showBulkAuto = !!disbursementSettings?.active_provider?.for_bulk_transfer;

  // Reset state when dialog closes
  useEffect(() => {
    if (!open) {
      setNinePayBatchId(null);
      setNinePayFileId(null);
      setConfirmed(false);
    }
  }, [open]);

  // Data sources
  const { data: ninePayConfig } = useAutoBulkTransferConfig();
  const ninePayEnabled = ninePayConfig?.enabled === true;

  const { data: balanceData, isLoading: balanceLoading } = useQuery({
    queryKey: ['wallet', 'balance'],
    queryFn: () => walletService.getBalance(),
    enabled: open,
  });
  const walletBalance = balanceData?.available ?? null;

  const { data: projectsResponse } = useProjects();
  const projects = useMemo(() => projectsResponse?.data || [], [projectsResponse?.data]);

  const form = useBulkTransferExportForm({
    isOpen: open,
    onOpenChange,
    projects,
  });

  // Mutations
  const exportMutation = useExportBulkTransfer();
  const initiateNinePayMutation = useInitiateAutoBulkTransfer();
  const { data: ninePayStatus } = useAutoBulkTransferStatus(ninePayBatchId);
  const isCompleted = ninePayStatus?.status === 'completed';
  const { data: historyDetail } = useBulkTransferUploadHistoryById(
    isCompleted ? ninePayFileId : null,
  );

  // While a provider batch is processing, invalidate timesheet caches each time
  const lastResolvedCountRef = useRef(0);
  useEffect(() => {
    if (!ninePayBatchId || !ninePayStatus) return;
    const resolved = (ninePayStatus.completed ?? 0) + (ninePayStatus.failed ?? 0);
    if (resolved !== lastResolvedCountRef.current) {
      lastResolvedCountRef.current = resolved;
      void queryClient.invalidateQueries({ queryKey: ['timesheets'] });
      void queryClient.invalidateQueries({ queryKey: ['timesheet-summary'] });
    }
  }, [ninePayBatchId, ninePayStatus, queryClient]);

  useEffect(() => {
    if (!ninePayBatchId) lastResolvedCountRef.current = 0;
  }, [ninePayBatchId]);

  // Provider disabled reasons
  const insufficientBalance =
    walletBalance !== null && estimatedTotal != null && walletBalance < estimatedTotal;
  const providerDisabled = !showBulkAuto || !ninePayEnabled || insufficientBalance;
  let providerWarning: string | undefined;
  if (!showBulkAuto) {
    providerWarning = 'Nhà cung cấp hiện không hỗ trợ chuyển lô tự động.';
  } else if (!ninePayEnabled) {
    providerWarning = 'Chuyển lô tự động chưa được kích hoạt.';
  } else if (insufficientBalance && walletBalance !== null && estimatedTotal != null) {
    providerWarning = `Số dư ví không đủ — cần thêm ${(estimatedTotal - walletBalance).toLocaleString('vi-VN')} ₫.`;
  }

  useEffect(() => {
    if (activeTab === 'provider' && providerDisabled) setActiveTab('manual');
  }, [activeTab, providerDisabled]);

  useEffect(() => {
    if (open && preselectedEmployeeIds && preselectedEmployeeIds.length > 0) {
      form.setSelectedEmployees(preselectedEmployeeIds);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const isSubmitting = exportMutation.isPending || initiateNinePayMutation.isPending;

  const handleManualExport = useCallback(async () => {
    const params = form.buildExportParams();
    try {
      await exportMutation.mutateAsync(params);
      recordPendingExport({
        fromDate: params.fromDate,
        toDate: params.toDate,
        forMonth: params.for_month,
        employeeIdsCount: params.employee_ids?.length,
        projectIdsCount: params.project_ids?.length,
      });
      onOpenChange(false);
    } catch {
      // Error handled by mutation
    }
  }, [form, exportMutation, onOpenChange]);

  const handleNinePayConfirmed = useCallback(async () => {
    const params = form.buildExportParams();
    try {
      const resp = await initiateNinePayMutation.mutateAsync(params);
      setNinePayBatchId(resp.batch_id);
      setNinePayFileId(resp.file_id);
    } catch {
      // Error handled by mutation; stay on confirm screen so user can retry
    }
  }, [form, initiateNinePayMutation]);

  const handleDownloadPdf = useCallback(async () => {
    if (!historyDetail) return;
    try {
      const rawName = `ket_qua_chuyen_tien_${dateToString(new Date())}.pdf`;
      await generateBulkTransferHistoryPdf(historyDetail, {
        fileName: sanitizeFilename(rawName),
      });
    } catch (error) {
      showErrorNotification(error, 'Không thể tạo PDF');
    }
  }, [historyDetail]);

  // ── Provider progress / completed view ──
  if (ninePayBatchId) {
    const completed = ninePayStatus?.status === 'completed';
    const progressPercent = ninePayStatus
      ? Math.round(((ninePayStatus.completed + ninePayStatus.failed) / ninePayStatus.total_count) * 100)
      : 0;

    // Completed view — show detailed transaction list
    if (completed && historyDetail) {
      const detailItems = historyDetail.items ?? [];

      return (
        <Dialog open={open} onOpenChange={onOpenChange}>
          <DialogContent className="max-w-4xl w-full max-h-[85vh] flex flex-col gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
            {/* Header */}
            <div className="shrink-0 bg-emerald-950 px-4 pt-4 pb-3 text-white">
              <div className="flex items-center gap-3">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-white leading-tight">Kết quả chuyển lô</p>
                  <p className="text-xs text-emerald-200/75 mt-0.5">
                    {ninePayStatus?.completed ?? 0} thành công · {ninePayStatus?.failed ?? 0} thất bại
                  </p>
                </div>
                <div className="flex items-center gap-1.5 shrink-0">
                  <Button
                    size="sm"
                    onClick={handleDownloadPdf}
                    className="h-7 px-2.5 text-xs gap-1 bg-white/10 hover:bg-white/20 text-white border-0 shadow-none"
                  >
                    <FileDown className="w-3 h-3" />
                    PDF
                  </Button>
                  <DialogClose className="w-7 h-7 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors outline-none">
                    <X className="w-3.5 h-3.5 text-white" />
                  </DialogClose>
                </div>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-muted min-h-0">
              {/* Inline summary strip — use ninePayStatus (real-time from wallet_payments) over cached file counts */}
              <div className="flex items-center gap-4 px-4 py-2.5 bg-muted/50 border-b text-xs">
                <span className="text-muted-foreground">
                  Tổng: <span className="font-semibold text-slate-800">{ninePayStatus?.total_count ?? historyDetail.total_txn}</span>
                </span>
                <span className="text-slate-300">|</span>
                <span className="text-muted-foreground flex items-center gap-1">
                  <CheckCircle2 className="w-3 h-3 text-emerald-500" />
                  Thành công: <span className="font-semibold text-emerald-700 ml-0.5">{ninePayStatus?.completed ?? historyDetail.completed_txn}</span>
                </span>
                <span className="text-slate-300">|</span>
                <span className="text-muted-foreground flex items-center gap-1">
                  <XCircle className="w-3 h-3 text-red-400" />
                  Thất bại: <span className={cn('font-semibold ml-0.5', (ninePayStatus?.failed ?? historyDetail.failed_txn) > 0 ? 'text-red-600' : 'text-muted-foreground')}>
                    {ninePayStatus?.failed ?? historyDetail.failed_txn}
                  </span>
                </span>
              </div>

              {/* Transaction list */}
              {detailItems.length > 0 ? (
                <div className="divide-y-0">
                  <div className="grid grid-cols-[1.5rem_minmax(8rem,1fr)_5rem_8rem_7rem_7rem] gap-x-2 px-3 py-1.5 bg-muted/50 border-b">
                    <div />
                    <p className="typography-label-small text-slate-400 uppercase">Nhân viên</p>
                    <p className="typography-label-small text-slate-400 uppercase">Ngân hàng</p>
                    <p className="typography-label-small text-slate-400 uppercase">STK</p>
                    <p className="typography-label-small text-slate-400 uppercase">CCCD</p>
                    <p className="typography-label-small text-slate-400 uppercase text-right">Số tiền</p>
                  </div>
                  {detailItems.map((detail, index) => {
                    const isPaid = detail.payment_status === 'paid';
                    return (
                      <div
                        key={index}
                        className={cn(
                          'grid grid-cols-[1.5rem_minmax(8rem,1fr)_5rem_8rem_7rem_7rem] items-center gap-x-2 py-2 px-3',
                          index < detailItems.length - 1 && 'border-b border-border',
                        )}
                      >
                        <div className="flex justify-center">
                          {isPaid
                            ? <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                            : <XCircle className="w-3.5 h-3.5 text-red-500 shrink-0" />}
                        </div>
                        <p className="typography-body-small font-medium text-foreground truncate leading-tight">
                          {detail.employee_name}
                        </p>
                        <p className="typography-body-small text-slate-500 truncate">
                          {detail.employee_bank_code || detail.employee_bank}
                        </p>
                        <p className="typography-body-small text-slate-500 truncate">
                          {detail.employee_account_number}
                        </p>
                        <p className="typography-body-small text-slate-400 truncate">
                          {detail.employee_cccd}
                        </p>
                        <p className="typography-body-small font-semibold text-slate-800 tabular-nums text-right whitespace-nowrap">
                          {formatCurrencyFromString(detail.amount)}
                        </p>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="flex items-center justify-center py-10">
                  <p className="text-sm text-slate-400">Không có chi tiết giao dịch</p>
                </div>
              )}

              {/* Failed items warning */}
              {ninePayStatus && ninePayStatus.failed > 0 && (
                <div className="rounded-xl border border-orange-200 bg-orange-50/60 p-3 m-3 space-y-2">
                  <p className="text-xs leading-relaxed text-orange-900">
                    Có <span className="font-semibold">{ninePayStatus.failed}</span> giao dịch thất bại.
                    Các bảng công đó đã quay về trạng thái <span className="font-medium">Chờ thanh toán</span>.
                  </p>
                  <Button
                    size="sm"
                    variant="outline"
                    className="border-orange-300 text-orange-800 hover:bg-orange-100"
                    onClick={() => setMarkExternallyPaidOpen(true)}
                  >
                    <Banknote className="w-4 h-4 mr-1.5" />
                    Đã chuyển
                  </Button>
                </div>
              )}
            </div>

            <DialogFooter className="shrink-0 px-4 py-3 border-t">
              <Button onClick={() => onOpenChange(false)} className="w-full">
                Đóng
              </Button>
            </DialogFooter>

            <MarkExternallyPaidDialog
              open={markExternallyPaidOpen}
              onOpenChange={setMarkExternallyPaidOpen}
              timesheetIds={(ninePayStatus?.failed_items ?? []).map((item) => item.timesheet_id)}
              batchLabel={`Batch #${ninePayBatchId.slice(0, 8)}`}
              onSuccess={() => {
                setMarkExternallyPaidOpen(false);
              }}
            />
          </DialogContent>
        </Dialog>
      );
    }

    // Processing / polling view
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-lg gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
          <DialogNavyHeader
            title={
              <span className="flex items-center gap-2">
                <CreditCard className="w-4 h-4" />
                Chuyển lô qua nhà cung cấp
              </span>
            }
            description="Đang theo dõi tiến trình chuyển tiền qua nhà cung cấp"
          />
          <div className="px-5 py-4 space-y-4">
            <Progress value={progressPercent} className="h-3" />
            <div className="grid grid-cols-1 gap-2 text-center sm:grid-cols-3 sm:gap-3">
              <div className="rounded-lg border bg-muted/30 p-3">
                <p className="text-2xl font-bold text-primary">{ninePayStatus?.total_count ?? '...'}</p>
                <p className="text-xs text-muted-foreground">Tổng</p>
              </div>
              <div className="rounded-lg border bg-muted/30 p-3">
                <p className="text-2xl font-bold text-green-600">{ninePayStatus?.completed ?? 0}</p>
                <p className="text-xs text-muted-foreground">Hoàn tất</p>
              </div>
              <div className="rounded-lg border bg-muted/30 p-3">
                <p className="text-2xl font-bold text-red-500">{ninePayStatus?.failed ?? 0}</p>
                <p className="text-xs text-muted-foreground">Thất bại</p>
              </div>
            </div>
            {ninePayStatus && ninePayStatus.processing > 0 && (
              <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" />
                Đang xử lý {ninePayStatus.processing} giao dịch...
              </div>
            )}
            {completed && !historyDetail && (
              <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" />
                Đang tải chi tiết...
              </div>
            )}
          </div>
          <DialogFooter className="px-5 py-3 border-t">
            <Button onClick={() => onOpenChange(false)} className="w-full">
              Đóng
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // ── Tabbed form ──
  const selectionCount = preselectedEmployeeIds?.length ?? 0;

  return (
    <Dialog open={open} onOpenChange={form.handleDialogOpen}>
      <DialogContent className="sm:max-w-lg gap-0 overflow-hidden" contentPadding="none" hideCloseButton>
        <DialogNavyHeader
          title="Chuyển lô"
          description={selectionCount > 0
            ? `Đã chọn ${selectionCount} nhân viên${estimatedTotal ? ` · Tổng: ${estimatedTotal.toLocaleString('vi-VN')} ₫` : ''}`
            : undefined}
        />

        <div className="px-5 py-4 space-y-3">
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as TransferMethod)} className="w-full">
          <TabsList className="grid w-full grid-cols-2 h-9 p-0.5">
            <TabsTrigger value="manual" className="gap-1.5 h-8 text-xs">
              <FileSpreadsheet className="h-3.5 w-3.5" />
              Thủ công
            </TabsTrigger>
            <TabsTrigger value="provider" disabled={providerDisabled} className="gap-1.5 h-8 text-xs">
              <CreditCard className="h-3.5 w-3.5" />
              Tự động
            </TabsTrigger>
          </TabsList>

          {/* Manual tab */}
          <TabsContent value="manual" className="space-y-3 mt-3">
            <p className="text-xs text-muted-foreground">
              Xuất Excel → CK qua app ngân hàng → upload kết quả tại{' '}
              <span className="font-medium text-foreground">Nhập KQ chuyển lô</span>.
            </p>
            <SharedExportForm form={form} projects={projects} isMobile={isMobile} />
          </TabsContent>

          {/* Provider tab */}
          <TabsContent value="provider" className="space-y-3 mt-3">
            <div className="flex items-center justify-between gap-3 rounded-md border bg-muted/30 px-3 py-1.5 text-xs">
              <span className="text-muted-foreground">Số dư</span>
              <div className="flex items-baseline gap-3 text-right">
                <span className="font-semibold tabular-nums text-foreground">
                  {balanceLoading ? '…' : walletBalance !== null ? `${walletBalance.toLocaleString('vi-VN')} ₫` : '—'}
                </span>
                {estimatedTotal != null && walletBalance !== null && (
                  <span
                    className={cn(
                      'tabular-nums text-xs',
                      insufficientBalance ? 'text-red-600 font-semibold' : 'text-muted-foreground',
                    )}
                    title="Số dư sau giao dịch (ước tính)"
                  >
                    → {(walletBalance - estimatedTotal).toLocaleString('vi-VN')} ₫
                  </span>
                )}
              </div>
            </div>
            {providerWarning && (
              <div className="flex items-start gap-1.5 rounded border border-amber-300/60 bg-amber-50 px-2 py-1.5 text-xs text-amber-800">
                <AlertTriangle className="h-3 w-3 shrink-0 mt-0.5" />
                <span>{providerWarning}</span>
              </div>
            )}
            <SharedExportForm form={form} projects={projects} isMobile={isMobile} />
          </TabsContent>
        </Tabs>

        {activeTab === 'provider' && !ninePayBatchId && (
          <label className="flex items-start gap-2 text-xs text-muted-foreground leading-relaxed cursor-pointer">
            <Checkbox
              checked={confirmed}
              onCheckedChange={(v) => setConfirmed(v === true)}
              className="mt-0.5 shrink-0"
            />
            <span>Hành động này sẽ khởi tạo các lệnh chuyển khoản qua nhà cung cấp và <span className="font-semibold text-foreground">không thể hoàn tác</span> sau khi thực hiện.</span>
          </label>
        )}
        </div>
        <DialogFooter className="px-5 py-3 border-t flex-row justify-end gap-2">
          <Button size="sm" variant="ghost" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            Hủy
          </Button>
          {activeTab === 'manual' ? (
            <Button size="sm" onClick={handleManualExport} disabled={!form.canExport || isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
                  Đang xuất...
                </>
              ) : (
                <>
                  <Download className="w-3.5 h-3.5 mr-1.5" />
                  Xuất Excel
                </>
              )}
            </Button>
          ) : (
            <Button size="sm" onClick={handleNinePayConfirmed} disabled={!form.canExport || isSubmitting || providerDisabled || !confirmed}>
              {isSubmitting ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />
                  Đang khởi tạo...
                </>
              ) : (
                <>
                  <CreditCard className="w-3.5 h-3.5 mr-1.5" />
                  Xác nhận
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});

// ── Shared form (date range + filters) ──
interface SharedExportFormProps {
  form: ReturnType<typeof useBulkTransferExportForm>;
  projects: Project[];
  isMobile: boolean;
}

function SharedExportForm({ form, projects, isMobile }: SharedExportFormProps) {
  return (
    <div className="space-y-2.5">
      {/* Payment Schedule */}
      <div className="space-y-1">
        <Label className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
          Chu kỳ trả lương <span className="text-red-500">*</span>
        </Label>
        <ButtonGroup
          options={[
            { value: 'weekly', label: 'Lương tuần' },
            { value: 'monthly', label: 'Lương tháng' },
          ]}
          value={form.paymentSchedule}
          onChange={form.setPaymentSchedule}
          fullWidth
        />
      </div>

      {/* Date Range */}
      <BulkTransferDateRangeSection
        paymentSchedule={form.paymentSchedule}
        fromDate={form.fromDate}
        toDate={form.toDate}
        weekPeriods={form.weekPeriods}
        monthPeriods={form.monthPeriods}
        customDateRanges={form.customDateRanges}
        selectedCustomRange={form.selectedCustomRange}
        hasDateError={form.hasDateError}
        isMobile={isMobile}
        onFromDateChange={form.setFromDate}
        onToDateChange={form.setToDate}
        onWeekPeriodApply={form.applyWeekPeriod}
        onMonthPeriodApply={form.applyMonthPeriod}
        onCustomRangeApply={form.applyCustomRange}
      />

      {/* Filters */}
      <BulkTransferFiltersSection
        selectedProjects={form.selectedProjects}
        selectedEmployees={form.selectedEmployees}
        projects={projects}
        onProjectChange={form.setSelectedProjects}
        onEmployeeChange={form.setSelectedEmployees}
      />
    </div>
  );
}
