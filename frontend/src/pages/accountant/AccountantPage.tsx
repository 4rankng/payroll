import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { BulkTransferExportDialog } from '@/components/timesheet/BulkTransferExportDialog';
import { BulkTransferResultUploadDialog } from '@/components/timesheet/BulkTransferResultUploadDialog';
import { PayrollReportExportDialog } from '@/components/timesheet/PayrollReportExportDialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
import { useBulkApproveTimesheets, useTimesheets } from '@/hooks/api/useTimesheets';
import { useExportBulkTransfer, useExportPayrollReport } from '@/hooks/api/usePayrolls';
import type { Timesheet } from '@/types/api/timesheet.types';
import { formatCurrency, formatDateForAPI } from '@/utils/formatters';
import { getAvailableWeekPeriods } from '@/utils/weekPeriodHelpers';
import { CalendarCheck, FileDown, FileSpreadsheet, LogOut, Upload } from 'lucide-react';

/**
 * Kế toán workspace — một trang duy nhất, không sidebar. Bốn việc:
 * duyệt công, xuất file chuyển lô, nhập KQ chuyển lô, xuất sao kê.
 * Quyền hạn thật nằm ở Casbin (role `accountant`); trang này chỉ là UI.
 */
export default function AccountantPage() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login', { replace: true });
  };

  return (
    <div className="min-h-screen bg-card">
      <header className="sticky top-0 z-10 border-b bg-card">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-3">
          <div className="min-w-0">
            <p className="text-sm font-semibold leading-5 text-foreground">Kế toán</p>
            <p className="truncate text-xs text-muted-foreground">
              {user?.name || user?.username || ''}
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={handleLogout}>
            <LogOut className="mr-1.5 h-4 w-4" aria-hidden="true" />
            Đăng xuất
          </Button>
        </div>
      </header>

      <main className="mx-auto w-full max-w-6xl px-4 py-6">
        <Tabs defaultValue="duyet-cong" className="w-full">
          <TabsList className="mb-4 grid h-auto w-full grid-cols-2 gap-1 sm:grid-cols-4">
            <TabsTrigger value="duyet-cong" className="gap-1.5 whitespace-normal text-center leading-snug">
              <CalendarCheck className="h-4 w-4" aria-hidden="true" />
              Duyệt công
            </TabsTrigger>
            <TabsTrigger value="xuat-chuyen-lo" className="gap-1.5 whitespace-normal text-center leading-snug">
              <FileSpreadsheet className="h-4 w-4" aria-hidden="true" />
              Xuất file chuyển lô
            </TabsTrigger>
            <TabsTrigger value="nhap-kq" className="gap-1.5 whitespace-normal text-center leading-snug">
              <Upload className="h-4 w-4" aria-hidden="true" />
              Nhập KQ chuyển lô
            </TabsTrigger>
            <TabsTrigger value="xuat-sao-ke" className="gap-1.5 whitespace-normal text-center leading-snug">
              <FileDown className="h-4 w-4" aria-hidden="true" />
              Xuất sao kê
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
    <Card>
      <CardHeader className="gap-3 border-b py-4">
        <CardTitle className="text-base">Chấm công chờ duyệt</CardTitle>
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
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto" role="region" aria-label="Chấm công chờ duyệt" tabIndex={0}>
        <Table>
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
                <TableCell colSpan={6} className="py-8 text-center text-sm text-muted-foreground">
                  Đang tải…
                </TableCell>
              </TableRow>
            ) : isError ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-sm text-destructive">
                  Không tải được danh sách chấm công. Vui lòng thử lại.
                </TableCell>
              </TableRow>
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="py-8 text-center text-sm text-muted-foreground">
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
                    <div className="text-sm font-medium">{r.employeeName}</div>
                    <div className="text-xs text-muted-foreground">{r.employeeCCCD || r.employeeCode}</div>
                  </TableCell>
                  <TableCell className="text-sm">{r.projectName}</TableCell>
                  <TableCell className="text-sm">{r.date}</TableCell>
                  <TableCell className="text-right text-sm tabular-nums">{r.hours_worked}</TableCell>
                  <TableCell className="text-right text-sm tabular-nums">{formatCurrency(r.amount)}</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
        </div>
      </CardContent>
      <div className="flex flex-wrap items-center justify-between gap-3 border-t px-4 py-3 sm:px-6">
        <div className="min-w-0">
          <p className="text-sm text-muted-foreground">
            Đã chọn {selected.size}/{rows.length}
            {selected.size > 0 && ` · ${formatCurrency(totalAmount)}`}
          </p>
          {hasHiddenRows && (
            <p className="text-sm text-warning">
              Còn {totalRecords - rows.length} dòng chưa hiển thị — thu hẹp khoảng ngày để xem tiếp.
            </p>
          )}
        </div>
        <Button onClick={handleApprove} disabled={selected.size === 0 || bulkApprove.isPending}>
          {bulkApprove.isPending ? 'Đang duyệt…' : `Duyệt (${selected.size})`}
        </Button>
      </div>
    </Card>
  );
}

/** Tab 2 — xuất file chuyển lô (Excel nộp ngân hàng). */
function ExportBulkTransferTab() {
  const [open, setOpen] = useState(false);
  const exportMutation = useExportBulkTransfer();

  return (
    <Card>
      <CardHeader className="border-b py-4">
        <CardTitle className="text-base">Xuất file chuyển lô</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col items-start gap-3 py-6">
        <p className="text-sm text-muted-foreground">
          Chọn tuần (hoặc tháng) và dự án, hệ thống tạo file Excel chuyển lô để nộp ngân hàng.
        </p>
        <Button onClick={() => setOpen(true)}>
          <FileSpreadsheet className="mr-1.5 h-4 w-4" aria-hidden="true" />
          Mở hộp thoại xuất file
        </Button>
      </CardContent>
      <BulkTransferExportDialog
        open={open}
        onOpenChange={setOpen}
        onExport={(params) => exportMutation.mutate(params)}
        isLoading={exportMutation.isPending}
      />
    </Card>
  );
}

/** Tab 3 — nhập kết quả chuyển lô từ file ngân hàng trả về. */
function ImportResultTab() {
  const [open, setOpen] = useState(false);

  return (
    <Card>
      <CardHeader className="border-b py-4">
        <CardTitle className="text-base">Nhập KQ chuyển lô</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col items-start gap-3 py-6">
        <p className="text-sm text-muted-foreground">
          Tải lên file kết quả ngân hàng trả về sau chuyển lô để cập nhật trạng thái thanh toán.
        </p>
        <Button onClick={() => setOpen(true)}>
          <Upload className="mr-1.5 h-4 w-4" aria-hidden="true" />
          Tải lên file kết quả
        </Button>
      </CardContent>
      <BulkTransferResultUploadDialog open={open} onOpenChange={setOpen} />
    </Card>
  );
}

/** Tab 4 — xuất sao kê thanh toán theo ngày. */
function ExportSaoKeTab() {
  const [open, setOpen] = useState(false);
  const exportMutation = useExportPayrollReport();

  return (
    <Card>
      <CardHeader className="border-b py-4">
        <CardTitle className="text-base">Xuất sao kê thanh toán</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col items-start gap-3 py-6">
        <p className="text-sm text-muted-foreground">
          Xuất sao kê thanh toán lương (file Excel) theo ngày đối chiếu.
        </p>
        <Button onClick={() => setOpen(true)}>
          <FileDown className="mr-1.5 h-4 w-4" aria-hidden="true" />
          Mở hộp thoại xuất sao kê
        </Button>
      </CardContent>
      <PayrollReportExportDialog
        open={open}
        onOpenChange={setOpen}
        onExport={(params) => exportMutation.mutate({ atDate: params.atDate })}
        isLoading={exportMutation.isPending}
      />
    </Card>
  );
}
