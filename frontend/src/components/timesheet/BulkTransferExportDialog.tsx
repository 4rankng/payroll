import { memo, useCallback, useMemo } from 'react';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { ButtonGroup } from '@/components/ui/button-group';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import { Download, Loader2, FileSpreadsheet, CreditCard, CheckCircle2, XCircle } from 'lucide-react';
import { useProjects } from '@/hooks/api/useProjects';
import { useEmployees } from '@/hooks/api/useEmployees';
import { useBulkTransferExportForm } from '@/hooks/timesheet/useBulkTransferExportForm';
import { useAutoBulkTransferStatus } from '@/hooks/api/usePayrolls';
import { type BulkTransferExportParams } from '@/services/api/bulk-transfer.service';
import { BulkTransferDateRangeSection } from './bulk-transfer-export/BulkTransferDateRangeSection';
import { BulkTransferFiltersSection } from './bulk-transfer-export/BulkTransferFiltersSection';

export type BulkTransferMode = 'export' | 'provider';

interface BulkTransferExportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onExport: (params: BulkTransferExportParams) => void;
  isLoading?: boolean;
  mode?: BulkTransferMode;
  ninePayBatchId?: string | null;
}

export const BulkTransferExportDialog = memo(function BulkTransferExportDialog({
  open,
  onOpenChange,
  onExport,
  isLoading = false,
  mode = 'export',
  ninePayBatchId = null,
}: BulkTransferExportDialogProps) {
  const { data: projectsResponse } = useProjects();
  const { data: employeesResponse } = useEmployees();

  const projects = useMemo(() => projectsResponse?.data || [], [projectsResponse?.data]);
  const employees = useMemo(() => employeesResponse?.data || [], [employeesResponse?.data]);

  const form = useBulkTransferExportForm({
    isOpen: open,
    onOpenChange,
    projects,
    employees
  });

  const isMobile = useIsMobile();
  const isProvider = mode === 'provider';

  const handleExport = useCallback(() => {
    const params = form.buildExportParams();
    onExport(params);
  }, [form, onExport]);

  // ── Provider progress state ──
  const { data: status } = useAutoBulkTransferStatus(ninePayBatchId);

  const isCompleted = status?.status === 'completed';
  const progressPercent = status
    ? Math.round(((status.completed + status.failed) / status.total_count) * 100)
    : 0;

  const isSubmitting = isLoading;

  // ── Render: Provider progress view ──
  if (isProvider && ninePayBatchId) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className="sm:max-w-lg shadow-none"
          title="Chuyển lô qua nhà cung cấp"
          description="Theo dõi tiến trình chuyển tiền"
        >
          <DialogHeader className="pb-2">
            <DialogTitle className="flex items-center gap-2">
              <CreditCard className="w-5 h-5 text-white/80" />
              Chuyển lô qua nhà cung cấp
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <Progress value={progressPercent} className="h-3" />

            <div className="grid grid-cols-3 gap-3 text-center">
              <div className="rounded-lg border p-3">
                <p className="text-2xl font-bold text-primary">{status?.total_count ?? '...'}</p>
                <p className="text-xs text-muted-foreground">Tổng</p>
              </div>
              <div className="rounded-lg border p-3">
                <p className="text-2xl font-bold text-green-600">{status?.completed ?? 0}</p>
                <p className="text-xs text-muted-foreground">Hoàn tất</p>
              </div>
              <div className="rounded-lg border p-3">
                <p className="text-2xl font-bold text-red-500">{status?.failed ?? 0}</p>
                <p className="text-xs text-muted-foreground">Thất bại</p>
              </div>
            </div>

            {status && status.processing > 0 && (
              <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" />
                Đang xử lý {status.processing} giao dịch...
              </div>
            )}

            {isCompleted && (
              <div className="flex items-center justify-center gap-2 text-sm font-medium">
                {status && status.failed === 0 ? (
                  <>
                    <CheckCircle2 className="w-5 h-5 text-green-600" />
                    <span className="text-green-600">Tất cả giao dịch đã hoàn tất</span>
                  </>
                ) : (
                  <>
                    <XCircle className="w-5 h-5 text-orange-500" />
                    <span className="text-orange-600">
                      Hoàn tất ({status.completed} thành công, {status.failed} thất bại)
                    </span>
                  </>
                )}
              </div>
            )}
          </div>

          <DialogFooter className="pt-2">
            <Button onClick={() => onOpenChange(false)} className="w-full">
              {isCompleted ? 'Đóng' : 'Đóng'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  }

  // ── Render: Provider submitting spinner ──
  if (isProvider && isSubmitting) {
    return (
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent
          className="sm:max-w-lg shadow-none"
          title="Đang khởi tạo"
          description="Vui lòng đợi..."
        >
          <DialogHeader className="pb-2">
            <DialogTitle className="flex items-center gap-2">
              <CreditCard className="w-5 h-5 text-white/80" />
              Đang khởi tạo chuyển lô
            </DialogTitle>
          </DialogHeader>
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  // ── Render: Export form (shared for both modes when no batch yet) ──
  return (
    <Dialog open={open} onOpenChange={form.handleDialogOpen}>
      <DialogContent
        className="sm:max-w-lg shadow-none"
        title={isProvider ? 'Chuyển lô qua nhà cung cấp' : 'Xuất file chuyển lô'}
        description={isProvider ? 'Chọn khoảng thời gian để chuyển tiền tự động qua nhà cung cấp' : 'Chọn khoảng thời gian và bộ lọc để xuất file Excel'}
      >
        <DialogHeader className="pb-2">
          <DialogTitle className="flex items-center gap-2">
            {isProvider ? (
              <CreditCard className="w-5 h-5 text-white/80" />
            ) : (
              <FileSpreadsheet className="w-5 h-5 text-white/80" />
            )}
            {isProvider ? 'Chuyển lô qua nhà cung cấp' : 'Xuất file chuyển lô'}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Payment Schedule */}
          <div className="space-y-2">
            <Label className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Chu kỳ trả lương <span className="text-red-500">*</span>
            </Label>
            <ButtonGroup
              options={[
                { value: 'weekly', label: 'Lương tuần' },
                { value: 'monthly', label: 'Lương tháng' }
              ]}
              value={form.paymentSchedule}
              onChange={form.setPaymentSchedule}
              fullWidth
            />
          </div>

          {/* Date Range */}
          <div className="space-y-2">
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
          </div>

          {/* Filters (collapsible) */}
          <BulkTransferFiltersSection
            selectedProjects={form.selectedProjects}
            selectedEmployees={form.selectedEmployees}
            projects={projects}
            employees={employees}
            onProjectChange={form.setSelectedProjects}
            onEmployeeChange={form.setSelectedEmployees}
          />
        </div>

        <DialogFooter className="flex-row gap-3 pt-2">
          <Button
            variant="outline"
            onClick={form.handleClose}
            disabled={isLoading}
            className="flex-1"
          >
            Đóng
          </Button>
          <Button
            onClick={handleExport}
            disabled={!form.canExport || isLoading}
            className="flex-1"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                {isProvider ? 'Đang xử lý...' : 'Đang xuất...'}
              </>
            ) : isProvider ? (
              <>
                <CreditCard className="w-4 h-4 mr-2" />
                Chuyển khoản tự động
              </>
            ) : (
              <>
                <Download className="w-4 h-4 mr-2" />
                Xuất Excel
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
});
