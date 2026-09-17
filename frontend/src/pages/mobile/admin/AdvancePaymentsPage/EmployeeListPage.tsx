import { useNavigate } from "react-router-dom";
import { ArrowLeft, Calendar, Download, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { AdvancePaymentMobileEmployeeList } from "@/components/advance-payment/AdvancePaymentMobileEmployeeList";
import { useAdvancePaymentsPage } from "@/hooks/advance-payment/useAdvancePaymentsPage";
import { formatMonthDisplay } from "@/utils/advancePaymentHelpers";
import type { FlexPayEmployeeListItem } from "@/types/api/advance-payment.types";
import { useState, useCallback } from "react";
import { EmployeeDetailSheet } from "./EmployeeDetailSheet";

const EmployeeListPage = () => {
  const navigate = useNavigate();
  const [selectedEmployee, setSelectedEmployee] =
    useState<FlexPayEmployeeListItem | null>(null);

  const page = useAdvancePaymentsPage({ employeesTabActive: true });

  const handleViewEmployeeDetail = useCallback(
    (emp: FlexPayEmployeeListItem) => {
      setSelectedEmployee(emp);
    },
    [],
  );

  return (
    <div className="flex min-h-full flex-col pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <MobilePageHeader
        title="Danh sách nhân viên"
        subtitle="Hạn mức ứng lương theo tháng"
        icon={Users}
        sticky={false}
        bordered={false}
        actions={
          <Button variant="ghost" size="icon" className="h-11 w-11" aria-label="Quay lại" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
        }
      />

      {/* Content */}
      <div className="flex-1 px-4 py-3 space-y-3">
        {/* Search + export */}
        <div className="flex gap-2">
          <MobileSearchInput
            value={page.flexPaySearchInput}
            onSearch={page.handleFlexPaySearch}
            placeholder="Tìm tên hoặc CCCD..."
            className="h-11"
          />
          <Button
            variant="outline"
            size="sm"
            className="h-11 min-w-11 px-3 shrink-0"
            onClick={page.handleExportFlexPayEmployees}
            disabled={page.exportFlexPayMutation.isPending}
            aria-label="Xuất danh sách"
          >
            <Download className="h-4 w-4" />
          </Button>
        </div>

        {/* Month selector */}
        <Select
          value={page.selectedViewMonth ?? ""}
          onValueChange={(v) => page.setSelectedViewMonth(v || undefined)}
        >
          <SelectTrigger aria-label="Tháng ứng lương" className="h-11">
            <Calendar className="h-4 w-4 mr-2 text-muted-foreground shrink-0" />
            <SelectValue placeholder="Chọn tháng" />
          </SelectTrigger>
          <SelectContent>
            {page.availableMonths.length === 0 ? (
              <SelectItem value="none" disabled>
                Chưa có dữ liệu
              </SelectItem>
            ) : (
              page.availableMonths.map((m) => (
                <SelectItem key={m.forMonth} value={m.forMonth}>
                  {formatMonthDisplay(m.forMonth)} ({m.employeeCount} NV)
                </SelectItem>
              ))
            )}
          </SelectContent>
        </Select>

        <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
          <Select
            value={page.flexPayFilters.sortBy ?? "max_advance_amount"}
            onValueChange={page.handleFlexPaySort}
          >
            <SelectTrigger aria-label="Sắp xếp nhân viên" className="h-11 min-w-0">
              <SelectValue placeholder="Sắp xếp theo" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="max_advance_amount">Hạn mức</SelectItem>
              <SelectItem value="utilized_amount">Đã dùng</SelectItem>
              <SelectItem value="pending_amount">Chờ xử lý</SelectItem>
              <SelectItem value="available_amount">Có thể ứng</SelectItem>
            </SelectContent>
          </Select>
          <Button
            type="button"
            variant="outline"
            className="h-11 min-w-20"
            onClick={() =>
              page.handleFlexPaySort(
                page.flexPayFilters.sortBy ?? "max_advance_amount",
              )
            }
          >
            {page.flexPayFilters.sortOrder === "ASC" ? "Tăng" : "Giảm"}
          </Button>
        </div>

        {/* Employee list */}
        <AdvancePaymentMobileEmployeeList
          employees={page.flexPayEmployees}
          onEmployeePress={handleViewEmployeeDetail}
          pagination={page.flexPayPagination}
          onPageChange={page.handleFlexPayPageChange}
        />
      </div>

      {/* Employee detail sheet (stays as sheet since it's a detail overlay) */}
      <EmployeeDetailSheet
        employee={selectedEmployee}
        onClose={() => setSelectedEmployee(null)}
      />
    </div>
  );
};

export default EmployeeListPage;
