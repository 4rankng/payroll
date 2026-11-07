import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { BulkTransferExportDialog } from '@/components/timesheet/BulkTransferExportDialog';
import { BulkTransferResultUploadDialog } from '@/components/timesheet/BulkTransferResultUploadDialog';
import { PayrollReportExportDialog } from '@/components/timesheet/PayrollReportExportDialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useAuth } from '@/contexts/AuthContext';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { getConsoleKicker } from '@/lib/console';
import { useBulkApproveTimesheets, useTimesheets } from '@/hooks/api/useTimesheets';
import { useExportBulkTransfer, useExportPayrollReport } from '@/hooks/api/usePayrolls';
import type { Timesheet } from '@/types/api/timesheet.types';
import { formatCurrency, formatDateForAPI } from '@/utils/formatters';
import { getAvailableWeekPeriods } from '@/utils/weekPeriodHelpers';
import { CalendarCheck01, FileDownload01, FileDownload02, LogOut01, Upload01 } from '@untitledui/icons';

/**
 * Kế toán workspace — một trang duy nhất, không sidebar. Bốn việc:
 * duyệt công, xuất file chuyển lô, nhập KQ chuyển lô, xuất sao kê.
 * Quyền hạn thật nằm ở Casbin (role `accountant`); trang này chỉ là UI.
 */
export default function AccountantPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  // Same pattern as AdminLayout: role tokens on <html> so portalled dialogs
  // (rendered outside this wrapper) inherit the accountant scope too.
  useEffect(() => {
    document.documentElement.classList.add('accountant-route-active');
    return () => document.documentElement.classList.remove('accountant-route-active');
  }, []);

  const handleLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  return (
    <div data-accountant-ui="" className="console-shell min-h-screen bg-card">
      <header className="sticky top-0 z-10 border-b border-utility-gray-200 bg-card">
        <div className="console-page-header mx-auto flex max-w-6xl items-center justify-between gap-3 pr-4">
          <div className="min-w-0">
            <p className="console-kicker mb-1">{getConsoleKicker('/accountant')}</p>
            <h1 className="font-display text-xl font-bold tracking-tight text-fg-primary sm:text-2xl">
              Bàn làm việc
            </h1>
            <p className="mt-1 truncate font-financial text-[11px] text-fg-tertiary">
              {user?.name || user?.username || ''} · Kỳ {format(new Date(), 'MM/yyyy', { locale: vi })}
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={handleLogout}>
            <LogOut01 className="size-4" aria-hidden="true" />
            Đăng xuất
          </Button>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl px-4 py-6">
        <Tabs defaultValue="duyet-cong" className="w-full">
          <TabsList className="mb-4 grid h-auto w-full grid-cols-2 gap-1 sm:grid-cols-4">
            <TabsTrigger value="duyet-cong" className="w-full min-w-0 gap-1.5 px-2.5 sm:h-9 sm:min-h-0 sm:px-3">
              <CalendarCheck01 className="size-4 shrink-0" aria-hidden="true" />
              <span className="truncate">Duyệt công</span>
            </TabsTrigger>
            <TabsTrigger value="xuat-chuyen-lo" className="w-full min-w-0 gap-1.5 px-2.5 sm:h-9 sm:min-h-0 sm:px-3">
              <FileDownload02 className="size-4 shrink-0" aria-hidden="true" />
              <span className="truncate">Xuất file chuyển lô</span>
            </TabsTrigger>
            <TabsTrigger value="nhap-kq" className="w-full min-w-0 gap-1.5 px-2.5 sm:h-9 sm:min-h-0 sm:px-3">
              <Upload01 className="size-4 shrink-0" aria-hidden="true" />
              <span className="truncate">Nhập KQ chuyển lô</span>
            </TabsTrigger>
            <TabsTrigger value="xuat-sao-ke" className="w-full min-w-0 gap-1.5 px-2.5 sm:h-9 sm:min-h-0 sm:px-3">
              <FileDownload01 className="size-4 shrink-0" aria-hidden="true" />
              <span className="truncate">Xuất sao kê</span>
            </TabsTrigger>
          </TabsList>

          <TabsContent value="duyet-cong">
            <ApproveTimesheetsTab />
          </TabsContent>
          <TabsContent value="xuat-chuyen-lo">
            <ExportBulkTransferTab />
          </TabsContent>
          <TabsContent value="nhap-kq">
            <ImportResultTab />
          </TabsContent>
          <TabsContent value="xuat-sao-ke">
            <ExportSaoKeTab />
          </TabsContent>
        </Tabs>
      </main>
    </div>
  );
}

/** Tab 1 — duyệt công: chọn tuần, tick các dòng chờ duyệt, duyệt loạt. */
function ApproveTimesheetsTab() {
  const defaultWeek = useMemo(() => getAvailableWeekPeriods().defaultPeriod, []);
  const [fromDate, setFromDate] = useState<Date>(new Date(defaultWeek.from));
  const [toDate, setToDate] = useState<Date>(new Date(defaultWeek.to));
  const [selected, setSelected] = useState<Set<number>>(new Set());

  // Changing the week invalidates the row set — drop stale selections instead
  // of approving ids the accountant can no longer see on screen.
  useEffect(() => {
    setSelected(new Set());
  }, [fromDate, toDate]);

  const { data, isLoading, isError } = useTimesheets({
    status: 'pending_approval',
    fromDate: formatDateForAPI(fromDate),
    toDate: formatDateForAPI(toDate),
    page: 1,
    pageSize: 500,
    sortBy: 'date',
    sortOrder: 'asc',
  });
  const rows: Timesheet[] = data?.data ?? [];
  // Page 1 of pageSize 500 only — surface the remainder instead of letting a
  // busy week silently hide pending rows the accountant believes are loaded.
  const totalRecords = data?.pagination?.totalRecords ?? rows.length;
  const hasHiddenRows = totalRecords > rows.length;
  const bulkApprove = useBulkApproveTimesheets();

  const toggleAll = (checked: boolean) => {
    setSelected(checked ? new Set(rows.map((r) => r.id)) : new Set());
  };
  const toggleOne = (id: number, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  const handleApprove = () => {
    if (selected.size === 0) return;
    bulkApprove.mutate(
      { timesheet_ids: Array.from(selected) },
      { onSuccess: () => setSelected(new Set()) },
    );
  };

  const totalAmount = rows
    .filter((r) => selected.has(r.id))
    .reduce((sum, r) => sum + (r.amount ?? 0), 0);

  return (
    <div className="console-panel">
      <div className="console-panel-header flex-wrap">
        <p className="text-sm font-semibold text-fg-primary">Chấm công chờ duyệt</p>
        <div className="flex flex-wrap items-end gap-3">
          <div className="space-y-1">
            <Label htmlFor="acc-from" className="text-xs">
              Từ ngày
            </Label>
            <Input
              id="acc-from"
              type="date"
              className="h-11 w-[150px] sm:h-9"
              value={formatDateForAPI(fromDate)}
              onChange={(e) => {
                if (e.target.value) setFromDate(new Date(e.target.value));
              }}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="acc-to" className="text-xs">
              Đến ngày
            </Label>
            <Input
              id="acc-to"
              type="date"
              className="h-11 w-[150px] sm:h-9"
              value={formatDateForAPI(toDate)}
              onChange={(e) => {
                if (e.target.value) setToDate(new Date(e.target.value));
              }}
            />
          </div>
        </div>
      </div>
      <div className="overflow-x-auto" role="region" aria-label="Chấm công chờ duyệt" tabIndex={0}>
        <Table variant="console">
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">
                <label className="inline-flex h-11 w-11 cursor-pointer items-center justify-center sm:h-4 sm:w-4">
                <Checkbox
                  aria-label="Chọn tất cả"
                  checked={rows.length > 0 && selected.size === rows.length}
                  onCheckedChange={(v) => toggleAll(v === true)}
                />
                </label>
              </TableHead>
              <TableHead>Nhân viên</TableHead>
              <TableHead>Dự án</TableHead>
              <TableHead>Ngày</TableHead>
              <TableHead className="text-right">Giờ</TableHead>
              <TableHead className="text-right">Thành tiền</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-sm text-fg-tertiary">
                  Đang tải…
                </TableCell>
              </TableRow>
            ) : isError ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-sm text-fg-error-primary">
                  Không tải được danh sách chấm công. Vui lòng thử lại.
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-sm text-fg-tertiary">
                  Không có chấm công chờ duyệt trong khoảng ngày đã chọn.
                </TableCell>
              </TableRow>
            ) : (
              rows.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>
                    <label className="inline-flex h-11 w-11 cursor-pointer items-center justify-center sm:h-4 sm:w-4">
                    <Checkbox
                      aria-label={`Chọn ${r.employeeName}`}
                      checked={selected.has(r.id)}
                      onCheckedChange={(v) => toggleOne(r.id, v === true)}
                    />
                    </label>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm font-medium text-fg-primary">{r.employeeName}</div>
                    <div className="text-xs text-fg-tertiary">{r.employeeCCCD || r.employeeCode}</div>
                  </TableCell>
                  <TableCell className="text-sm">{r.projectName}</TableCell>
                  <TableCell className="num">{r.date}</TableCell>
                  <TableCell className="num tabular-nums">{r.hours_worked}</TableCell>
                  <TableCell className="num tabular-nums">{formatCurrency(r.amount)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
      <div className="console-divider flex flex-wrap items-center justify-between gap-3 border-t px-4 py-3 sm:px-6">
        <div className="min-w-0">
          <p className="text-sm text-fg-tertiary">
            Đã chọn {selected.size}/{rows.length}
            {selected.size > 0 && ` · ${formatCurrency(totalAmount)}`}
          </p>
          {hasHiddenRows && (
            <p className="text-sm text-fg-warning-primary">
              Còn {totalRecords - rows.length} dòng chưa hiển thị — thu hẹp khoảng ngày để xem tiếp.
            </p>
          )}
        </div>
        <Button onClick={handleApprove} disabled={selected.size === 0 || bulkApprove.isPending}>
          {bulkApprove.isPending ? 'Đang duyệt…' : `Duyệt (${selected.size})`}
        </Button>
      </div>
    </div>
  );
}

/** Tab 2 — xuất file chuyển lô (Excel nộp ngân hàng). */
function ExportBulkTransferTab() {
  const [open, setOpen] = useState(false);
  const exportMutation = useExportBulkTransfer();

  return (
    <div className="console-panel">
      <div className="console-panel-header">
        <p className="text-sm font-semibold text-fg-primary">Xuất file chuyển lô</p>
      </div>
      <div className="console-panel-body flex flex-col items-start gap-3">
        <p className="text-sm text-fg-tertiary">
          Chọn tuần (hoặc tháng) và dự án, hệ thống tạo file Excel chuyển lô để nộp ngân hàng.
        </p>
        <Button onClick={() => setOpen(true)}>
          <FileDownload02 className="mr-1.5 h-4 w-4" aria-hidden="true" />
          Mở hộp thoại xuất file
        </Button>
      </div>
      <BulkTransferExportDialog
        open={open}
        onOpenChange={setOpen}
        onExport={(params) => exportMutation.mutate(params)}
        isLoading={exportMutation.isPending}
      />
    </div>
  );
}

/** Tab 3 — nhập kết quả chuyển lô từ file ngân hàng trả về. */
function ImportResultTab() {
  const [open, setOpen] = useState(false);

  return (
    <div className="console-panel">
      <div className="console-panel-header">
        <p className="text-sm font-semibold text-fg-primary">Nhập KQ chuyển lô</p>
      </div>
      <div className="console-panel-body flex flex-col items-start gap-3">
        <p className="text-sm text-fg-tertiary">
          Tải lên file kết quả ngân hàng trả về sau chuyển lô để cập nhật trạng thái thanh toán.
        </p>
        <Button onClick={() => setOpen(true)}>
          <Upload01 className="mr-1.5 h-4 w-4" aria-hidden="true" />
          Tải lên file kết quả
        </Button>
      </div>
      <BulkTransferResultUploadDialog open={open} onOpenChange={setOpen} />
    </div>
  );
}

/** Tab 4 — xuất sao kê thanh toán theo ngày. */
function ExportSaoKeTab() {
  const [open, setOpen] = useState(false);
  const exportMutation = useExportPayrollReport();

  return (
    <div className="console-panel">
      <div className="console-panel-header">
        <p className="text-sm font-semibold text-fg-primary">Xuất sao kê thanh toán</p>
      </div>
      <div className="console-panel-body flex flex-col items-start gap-3">
        <p className="text-sm text-fg-tertiary">
          Xuất sao kê thanh toán lương (file Excel) theo ngày đối chiếu.
        </p>
        <Button onClick={() => setOpen(true)}>
          <FileDownload01 className="mr-1.5 h-4 w-4" aria-hidden="true" />
          Mở hộp thoại xuất sao kê
        </Button>
      </div>
      <PayrollReportExportDialog
        open={open}
        onOpenChange={setOpen}
        onExport={(params) => exportMutation.mutate({ atDate: params.atDate })}
        isLoading={exportMutation.isPending}
      />
    </div>
  );
}
