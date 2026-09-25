import { useState, useMemo, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { format } from "date-fns";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Wallet, Users, Calendar, ChevronLeft, ChevronRight, Download, FileText, ScanFace } from "lucide-react";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { SearchBar } from "@/components/shared/SearchBar";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ImportPayrollDialog } from "@/components/advance-payment/ImportPayrollDialog";
import { FlexibleEmployeeListUploadDialog } from "@/components/advance-payment/FlexibleEmployeeListUploadDialog";
import { EmployeeAdvancePaymentDetailSheet } from "@/components/advance-payment/EmployeeAdvancePaymentDetailSheet";
import { AdvPartnerHeroStrip } from "@/components/advance-payment/AdvPartnerHeroStrip";
import { AdvPartnerStatusOverview } from "@/components/advance-payment/AdvPartnerStatusOverview";
import {
  getAdvancePaymentColumns,
  getFlexPayColumns,
  requestMobileFields,
  flexPayMobileFields,
  requestEmptyState,
  flexPayEmptyState,
} from "@/components/advance-payment/table-config";
import { useAdvancePaymentsPage } from "@/hooks/advance-payment/useAdvancePaymentsPage";
import { useExportAdvancePayments } from "@/hooks/api/useAdvancePayments";
import { apiClient } from "@/services/api/client";
import { API_ENDPOINTS } from "@/config/api.config";
import { showErrorNotification } from "@/utils/error-handler";
import { useAuth } from "@/contexts";
import { cn } from "@/lib/utils";
import type {
  AdvancePaymentListItem,
  FlexPayEmployeeListItem,
  AdvancePaymentRequestStatus,
} from "@/types/api/advance-payment.types";
import type { ActiveTab } from "./types";

/* ------------------------------------------------------------------ */
/*  Tab pill bar                                                       */
/* ------------------------------------------------------------------ */

interface ViewTab {
  id: ActiveTab;
  label: string;
  icon: typeof Wallet;
  count?: number;
}

function ViewTabs({
  tabs,
  activeTab,
  onTabChange,
}: {
  tabs: ViewTab[];
  activeTab: ActiveTab;
  onTabChange: (id: ActiveTab) => void;
}) {
  return (
    <div
      className="inline-flex items-center gap-0.5 p-0.5 rounded-lg bg-muted/60"
      role="tablist"
      aria-label="Chuyển tab"
    >
      {tabs.map((tab) => {
        const isActive = activeTab === tab.id;
        const Icon = tab.icon;
        return (
          <button
            key={tab.id}
            role="tab"
            aria-selected={isActive}
            onClick={() => onTabChange(tab.id)}
            className={cn(
              "inline-flex items-center gap-1.5 px-3 h-7 rounded-md text-[13px] font-medium transition-all whitespace-nowrap",
              isActive
                ? "bg-card text-foreground shadow-[0_1px_2px_0_rgb(15_23_42/0.06)]"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <Icon className="h-3.5 w-3.5 shrink-0 opacity-80" />
            {tab.label}
            {tab.count != null && (
              <span
                className={cn(
                  "tabular-nums text-xs px-1.5 py-px rounded font-semibold leading-none",
                  isActive
                    ? "bg-muted text-foreground/70"
                    : "bg-transparent text-muted-foreground",
                )}
              >
                {tab.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Month navigation                                                   */
/* ------------------------------------------------------------------ */

function MonthNavigator({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  const handlePrev = useCallback(() => {
    const [y, m] = value.split("-").map(Number);
    const d = new Date(y, m - 2, 1);
    onChange(format(d, "yyyy-MM"));
  }, [value, onChange]);

  const handleNext = useCallback(() => {
    const [y, m] = value.split("-").map(Number);
    const d = new Date(y, m, 1);
    onChange(format(d, "yyyy-MM"));
  }, [value, onChange]);

  const displayLabel = useMemo(() => {
    const [y, m] = value.split("-").map(Number);
    const d = new Date(y, m - 1, 1);
    return format(d, "MM / yyyy");
  }, [value]);

  const now = new Date();
  const isCurrent = value === format(now, "yyyy-MM");

  return (
    <div className="inline-flex items-center gap-0.5 rounded-lg border border-border/70 bg-card px-1 h-8">
      <button
        onClick={handlePrev}
        className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors"
        aria-label="Tháng trước"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
      </button>
      <span className="px-1.5 text-[13px] font-medium tabular-nums text-foreground min-w-[80px] text-center">
        {displayLabel}
        {isCurrent && (
          <span className="ml-1 inline-flex items-center align-middle">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" aria-hidden />
          </span>
        )}
      </span>
      <button
        onClick={handleNext}
        className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground hover:text-foreground hover:bg-muted/60 transition-colors"
        aria-label="Tháng sau"
      >
        <ChevronRight className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

/* ------------------------------------------------------------------ */
/*  Page component                                                     */
/* ------------------------------------------------------------------ */

const AdvPartnerAdvancePaymentsPage = () => {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<ActiveTab>("requests");
  const [isImportSheetOpen, setIsImportSheetOpen] = useState(false);
  const [selectedEmployee, setSelectedEmployee] =
    useState<FlexPayEmployeeListItem | null>(null);

  const { user } = useAuth();

  const page = useAdvancePaymentsPage({
    employeesTabActive: activeTab === "employees",
  });
  const exportBatchMutation = useExportAdvancePayments();

  /* ── Derived data ── */
  const columns = useMemo(
    () => getAdvancePaymentColumns({ onCancel: page.handleCancelRequest }),
    [page.handleCancelRequest],
  );

  const flexPayColumns = useMemo(
    () =>
      getFlexPayColumns({
        sortBy: page.flexPayFilters.sortBy,
        sortOrder: page.flexPayFilters.sortOrder,
        onSort: page.handleFlexPaySort,
      }),
    [
      page.flexPayFilters.sortBy,
      page.flexPayFilters.sortOrder,
      page.handleFlexPaySort,
    ],
  );

  const summaryData = page.summary?.data;

  const heroProps = useMemo(
    () => ({
      totalAmount: summaryData?.totalAmount ?? 0,
      totalPaid: summaryData?.totalPaid ?? 0,
    }),
    [summaryData],
  );

  const statusProps = useMemo(
    () => ({
      totalPaid: summaryData?.totalPaid ?? 0,
      totalPending: summaryData?.totalPending ?? 0,
      totalFailed: summaryData?.totalFailed ?? 0,
      totalCancelled: summaryData?.totalCancelled ?? 0,
      totalRequests: summaryData?.totalRequests ?? 0,
      successRate: summaryData?.successRate ?? 0,
    }),
    [summaryData],
  );

  const resultCount = activeTab === "requests"
    ? page.pagination?.totalRecords ?? page.requests.length
    : page.flexPayPagination?.totalRecords ?? page.flexPayEmployees.length;

  const handleEmployeeClose = useCallback(() => setSelectedEmployee(null), []);

  /* ── Render ── */
  return (
    <div className="p-4 lg:p-8 max-w-[1280px] mx-auto space-y-6">
      {/* Zone 1: Header */}
      <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        {/* Title row — on mobile also includes the month navigator */}
        <div className="flex items-center justify-between gap-3 sm:block min-w-0">
          <h1 className="text-[18px] sm:text-[22px] font-semibold text-foreground tracking-tight leading-tight whitespace-nowrap">
            Quản lý ứng lương
          </h1>
          {/* Month nav visible on mobile only here */}
          <div className="shrink-0 sm:hidden">
            <MonthNavigator
              value={page.selectedMonth}
              onChange={page.setSelectedMonth}
            />
          </div>
        </div>

        {/* Action row */}
        <div className="flex items-center gap-2 shrink-0 flex-wrap">
          <Button
            size="sm"
            onClick={() => navigate("/adv-partner/advance-payments/check-in-settings")}
            className="h-8 gap-1.5 rounded-lg px-3 text-[13px] font-medium"
          >
            <ScanFace className="h-3.5 w-3.5 shrink-0" aria-hidden />
            Cấu hình điểm danh
          </Button>
          <Button
            size="sm"
            onClick={() => {
              apiClient.download(
                `${API_ENDPOINTS.advancePayments.reconciliation.export}?forMonth=${page.selectedMonth}`,
                `sao_ke_thanh_toan_${page.selectedMonth}.xlsx`,
              ).catch((error) => { showErrorNotification(error); });
            }}
            className="h-8 px-3 font-medium text-[13px] gap-1.5 rounded-lg bg-white/80 hover:bg-white border border-neutral-300 text-neutral-700 shadow-sm hover:shadow-md transition-all duration-200"
          >
            <FileText className="h-3.5 w-3.5 shrink-0" strokeWidth={2} />
            Xuất sao kê
          </Button>
          <Button
            size="sm"
            onClick={page.handleExportFlexPayEmployees}
            disabled={page.exportFlexPayMutation.isPending}
            aria-label="Xuất danh sách"
            className="h-8 px-3 font-medium text-[13px] gap-1.5 rounded-lg bg-white/80 hover:bg-white border border-neutral-300 text-neutral-700 shadow-sm hover:shadow-md transition-all duration-200 disabled:opacity-50"
          >
            <Download className="h-3.5 w-3.5 shrink-0" strokeWidth={2} />
            Xuất DS
          </Button>
          {/* Month nav on desktop only here */}
          <div className="hidden sm:flex items-center gap-2 ml-0">
            <div className="w-px h-4 bg-border/60 mx-1" />
            <MonthNavigator
              value={page.selectedMonth}
              onChange={page.setSelectedMonth}
            />
          </div>
        </div>
      </header>

      {/* Zone 2 & 3: Hero metrics & Status overview */}
      <div className="space-y-5">
        <AdvPartnerHeroStrip {...heroProps} isLoading={page.summaryLoading} />
        <AdvPartnerStatusOverview
          {...statusProps}
          isLoading={page.summaryLoading}
        />
      </div>

      {/* Zone 4: Tabs + Toolbar + Table */}
      <div className="rounded-xl border border-border/70 bg-card shadow-[0_1px_2px_0_rgb(15_23_42/0.04)] overflow-hidden">
        {/* Tab bar + filters — stacked on mobile, inline on sm+ */}
        <div className="px-4 sm:px-5 py-3 flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3">

          {/* Row 1: Tab pills + result count (count shows here on mobile) */}
          <div className="flex items-center justify-between gap-2">
            <ViewTabs
              tabs={[
                {
                  id: "requests",
                  label: "Yêu cầu",
                  icon: Wallet,
                  count: page.statusCounts?.all ?? 0,
                },
                {
                  id: "employees",
                  label: "Nhân viên",
                  icon: Users,
                  count: page.flexPayPagination?.totalRecords ?? 0,
                },
              ]}
              activeTab={activeTab}
              onTabChange={setActiveTab}
            />
            <span className="sm:hidden text-xs font-medium text-muted-foreground tabular-nums whitespace-nowrap">
              <span className="text-foreground/80 font-semibold">{resultCount}</span> kết quả
            </span>
          </div>

          {/* Row 2: Search + filter (count shows here on desktop) */}
          <div className="flex items-center gap-2 sm:ml-auto">
            {activeTab === "requests" && (
              <>
                <SearchBar
                  searchTerm={page.searchInput}
                  onSearchChange={page.handleSearch}
                  placeholder="Tên hoặc CCCD..."
                  className="h-8 flex-1 sm:flex-none sm:w-56 text-sm"
                />
                <Select
                  value={(page.filters.status as AdvancePaymentRequestStatus) || "all"}
                  onValueChange={page.handleStatusChange}
                >
                  <SelectTrigger aria-label="Lọc trạng thái yêu cầu ứng lương" className="h-8 min-h-0 shrink-0 w-auto min-w-[90px] sm:min-w-[120px] text-[13px] font-medium gap-1">
                    <SelectValue placeholder="Trạng thái" />
                  </SelectTrigger>
                  <SelectContent className="min-w-[140px]">
                    <SelectItem value="all" className="text-[13px] py-1">Tất cả</SelectItem>
                    <SelectItem value="pending" className="text-[13px] py-1">
                      <span className="flex items-center gap-2">
                        <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
                        Chờ xử lý
                      </span>
                    </SelectItem>
                    <SelectItem value="completed" className="text-[13px] py-1">
                      <span className="flex items-center gap-2">
                        <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                        Hoàn tất
                      </span>
                    </SelectItem>
                    <SelectItem value="failed" className="text-[13px] py-1">
                      <span className="flex items-center gap-2">
                        <span className="h-1.5 w-1.5 rounded-full bg-red-400" />
                        Thất bại
                      </span>
                    </SelectItem>
                    <SelectItem value="cancelled" className="text-[13px] py-1">
                      <span className="flex items-center gap-2">
                        <span className="h-1.5 w-1.5 rounded-full bg-gray-400" />
                        Đã hủy
                      </span>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </>
            )}

            {activeTab === "employees" && (
              <SearchBar
                searchTerm={page.flexPaySearchInput}
                onSearchChange={page.handleFlexPaySearch}
                placeholder="Tên hoặc CCCD..."
                className="h-8 flex-1 sm:flex-none sm:w-56 text-sm"
              />
            )}

            <span className="hidden sm:inline text-xs font-medium text-muted-foreground tabular-nums tracking-wide whitespace-nowrap">
              <span className="text-foreground/80 font-semibold">{resultCount}</span> kết quả
            </span>
          </div>
        </div>

        {/* Tables */}
        {activeTab === "requests" && (
          <ResponsiveTable
            data={page.requests}
            columns={columns}
            mobileFields={requestMobileFields}
            rowTitle={(row: AdvancePaymentListItem) => (
              <div className="text-[15px] font-semibold text-foreground leading-tight">{row.employeeName}</div>
            )}
            rowSubtitle={(row: AdvancePaymentListItem) => (
              <div className="space-y-0.5 mt-0.5">
                <div className="text-[12px] text-muted-foreground font-medium tabular-nums">{row.employeeCCCD}</div>
                <div className="text-[12px] text-muted-foreground truncate max-w-[150px]">{row.projectName || "—"}</div>
              </div>
            )}
            getRowId={(row: AdvancePaymentListItem) => row.id.toString()}
            pagination={page.pagination}
            onPageChange={page.handlePageChange}
            onPageSizeChange={page.handlePageSizeChange}
            emptyState={requestEmptyState}
            embedded
          />
        )}

        {activeTab === "employees" && (
          <ResponsiveTable
            data={page.flexPayEmployees}
            columns={flexPayColumns}
            mobileFields={flexPayMobileFields}
            rowTitle={(row: FlexPayEmployeeListItem) => (
              <div className="text-[15px] font-semibold text-foreground leading-tight">{row.fullname}</div>
            )}
            rowSubtitle={(row: FlexPayEmployeeListItem) => (
              <div className="space-y-0.5 mt-0.5">
                <div className="text-[12px] text-muted-foreground font-medium tabular-nums">{row.cccd}</div>
                <div className="text-[12px] text-muted-foreground truncate max-w-[150px]">{row.project?.name || "—"}</div>
              </div>
            )}
            getRowId={(row: FlexPayEmployeeListItem) =>
              `${row.employeeId}-${row.project.id}`
            }
            onRowClick={(row: FlexPayEmployeeListItem) =>
              setSelectedEmployee(row)
            }
            pagination={page.flexPayPagination}
            onPageChange={page.handleFlexPayPageChange}
            onPageSizeChange={page.handleFlexPayPageSizeChange}
            emptyState={flexPayEmptyState}
            embedded
          />
        )}
      </div>

      {/* Dialogs & Sheets */}
      <ImportPayrollDialog
        open={isImportSheetOpen}
        onOpenChange={setIsImportSheetOpen}
      />

      <EmployeeAdvancePaymentDetailSheet
        employee={selectedEmployee}
        onClose={handleEmployeeClose}
      />

      {/* Cancel confirm dialog */}
      <AlertDialog
        open={page.cancelConfirmId !== null}
        onOpenChange={(open) => {
          if (!open) page.dismissCancelConfirm();
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Hủy yêu cầu ứng lương?</AlertDialogTitle>
            <AlertDialogDescription>
              Yêu cầu ứng lương sẽ bị hủy vĩnh viễn.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Giữ lại</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={page.confirmCancelRequest}
            >
              {page.cancelMutation.isPending
                ? "Đang hủy..."
                : "Hủy yêu cầu"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default AdvPartnerAdvancePaymentsPage;
