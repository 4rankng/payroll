import { useCallback, useMemo, useState } from "react";
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
} from "@/components/shared/AdminPageFrame";
import { PageHeader } from "@/components/shared/PageHeader";
import { SearchBar } from "@/components/shared/SearchBar";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import {
  getFlexPayColumns,
  flexPayMobileFields,
  flexPayEmptyState,
} from "@/components/advance-payment/table-config";
import { EmployeeDetailSheet } from "@/components/advance-payment/EmployeeDetailSheet";
import { useAdvancePaymentsPage } from "@/hooks/advance-payment/useAdvancePaymentsPage";
import { formatMonthDisplay } from "@/utils/advancePaymentHelpers";
import type { FlexPayEmployeeListItem } from "@/types/api/advance-payment.types";

/** Desktop variant of the FlexPay employee-limit list (PAYR-23): dense table
 *  for >=768px, paired with the mobile card page via ResponsivePage. */
const EmployeeListPage = () => {
  const [selectedEmployee, setSelectedEmployee] = useState<FlexPayEmployeeListItem | null>(null);
  const page = useAdvancePaymentsPage({ employeesTabActive: true });

  const flexPayColumns = useMemo(
    () =>
      getFlexPayColumns({
        sortBy: page.flexPayFilters.sortBy,
        sortOrder: page.flexPayFilters.sortOrder,
        onSort: page.handleFlexPaySort,
      }),
    [page.flexPayFilters.sortBy, page.flexPayFilters.sortOrder, page.handleFlexPaySort],
  );

  const handleViewEmployeeDetail = useCallback((emp: FlexPayEmployeeListItem) => {
    setSelectedEmployee(emp);
  }, []);

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader
          title="Danh sách nhân viên"
          description="Hạn mức ứng lương theo tháng"
          actions={[
            {
              label: "Xuất danh sách",
              onClick: page.handleExportFlexPayEmployees,
              variant: "outline" as const,
              disabled: page.exportFlexPayMutation.isPending,
            },
          ]}
        />
      </AdminPageHeaderCard>

      <AdminSectionCard aria-label="Danh sách nhân viên ứng lương">
        <ResponsiveTable
          data={page.flexPayEmployees}
          columns={flexPayColumns}
          mobileFields={flexPayMobileFields}
          embedded
          columnVisibilityKey="adv-payments:employees"
          getRowId={(row) => row.employeeId.toString()}
          onRowClick={handleViewEmployeeDetail}
          pagination={page.flexPayPagination}
          onPageChange={page.handleFlexPayPageChange}
          emptyState={flexPayEmptyState}
          toolbar={
            <>
              <SearchBar
                searchTerm={page.flexPaySearchInput}
                onSearchChange={page.handleFlexPaySearch}
                placeholder="Tên hoặc CCCD..."
                className="h-9 w-full min-w-[220px] sm:w-64"
              />
              <Select
                value={page.selectedViewMonth ?? ""}
                onValueChange={(v) => page.setSelectedViewMonth(v || undefined)}
              >
                <SelectTrigger className="h-9 w-auto min-w-36 shrink-0 text-sm">
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
            </>
          }
        />
      </AdminSectionCard>

      <EmployeeDetailSheet
        employee={selectedEmployee}
        onClose={() => setSelectedEmployee(null)}
      />
    </AdminPageCanvas>
  );
};

export default EmployeeListPage;
