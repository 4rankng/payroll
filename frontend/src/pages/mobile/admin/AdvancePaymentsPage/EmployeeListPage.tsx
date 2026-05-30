import { useNavigate } from "react-router-dom";
import { ArrowLeft, Calendar, Download, Users } from "lucide-react";
import { Button } from "@/components/ui/button";
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
    <div className="flex flex-col min-h-full pb-20">
      {/* Sticky header with back button */}
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b border-border/40 shrink-0">
        <div className="flex items-center gap-3 px-4 pt-4 pb-3">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0 -ml-1"
            onClick={() => navigate(-1)}
            aria-label="Quay lại"
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div className="flex items-center gap-2 min-w-0">
            <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/10 shrink-0">
              <Users className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <h1 className="text-base font-semibold text-foreground leading-tight">
                Danh sách nhân viên
              </h1>
              <p className="text-xs text-muted-foreground leading-tight">
                Hạn mức ứng lương theo tháng
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 px-4 py-3 space-y-3">
        {/* Search + export */}
        <div className="flex gap-2">
          <MobileSearchInput
            value={page.flexPaySearchInput}
            onSearch={page.handleFlexPaySearch}
            placeholder="Tìm tên hoặc CCCD..."
            className="h-10"
          />
          <Button
            variant="outline"
            size="sm"
            className="h-10 px-3 shrink-0"
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
          <SelectTrigger className="h-10">
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
