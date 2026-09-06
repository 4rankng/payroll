import { memo } from "react";
import { Label } from "@/components/ui/label";
import { SearchableSelect } from "@/components/ui/searchable-select";
import type { Employee } from "@/types/api/employee.types";

interface EmployeeSelectorProps {
  projectId: number;
  selectedEmployee: Employee | null;
  availableEmployees: Employee[];
  isLoadingProjectEmployees: boolean;
  onEmployeeChange: (employee: Employee | null) => void;
  onUpdateUrlParams: (updates: { employeeId?: number }) => void;
}

export const EmployeeSelector = memo(
  ({
    projectId,
    selectedEmployee,
    availableEmployees,
    isLoadingProjectEmployees,
    onEmployeeChange,
    onUpdateUrlParams,
  }: EmployeeSelectorProps) => {
    const handleEmployeeSelect = (value: string) => {
      const employee = availableEmployees.find(
        (emp) => emp.id.toString() === value,
      );
      const newEmployeeId = employee?.id || 0;

      onEmployeeChange(employee || null);
      onUpdateUrlParams({ employeeId: newEmployeeId });
    };

    return (
      <div>
        <Label htmlFor="timesheet-entry-employee">Nhân viên *</Label>
        {projectId ? (
          <div className="space-y-2">
            {isLoadingProjectEmployees ? (
              <div className="flex items-center justify-center p-4 border bg-muted/30 rounded-xl">
                <div className="animate-spin rounded-full h-4 w-4 border-2 border-current border-t-transparent mr-2" />
                Đang tải nhân viên...
              </div>
            ) : availableEmployees.length > 0 ? (
              <SearchableSelect
                triggerId="timesheet-entry-employee"
                value={selectedEmployee?.id.toString() || ""}
                onChange={handleEmployeeSelect}
                placeholder="Chọn nhân viên từ dự án"
                searchPlaceholder="Tìm nhân viên..."
                options={availableEmployees.map((employee) => ({
                  value: employee.id.toString(),
                  label: employee.fullname,
                  searchText: employee.position
                    ? `Vị trí: ${employee.position}`
                    : undefined,
                }))}
              />
            ) : (
              <div className="p-4 border bg-muted/30 rounded-xl text-muted-foreground">
                Không có nhân viên nào đang làm việc trong dự án này
              </div>
            )}
          </div>
        ) : (
          <div className="p-4 border bg-muted/30 rounded-xl text-muted-foreground">
            Vui lòng chọn dự án trước
          </div>
        )}
      </div>
    );
  },
);

EmployeeSelector.displayName = "EmployeeSelector";
