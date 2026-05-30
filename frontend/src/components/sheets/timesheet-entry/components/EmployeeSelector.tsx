import { memo } from 'react';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { Employee } from '@/types/api/employee.types';

interface EmployeeSelectorProps {
  projectId: number;
  selectedEmployee: Employee | null;
  availableEmployees: Employee[];
  isLoadingProjectEmployees: boolean;
  onEmployeeChange: (employee: Employee | null) => void;
  onUpdateUrlParams: (updates: { employeeId?: number }) => void;
}

export const EmployeeSelector = memo(({
  projectId,
  selectedEmployee,
  availableEmployees,
  isLoadingProjectEmployees,
  onEmployeeChange,
  onUpdateUrlParams
}: EmployeeSelectorProps) => {
  const handleEmployeeSelect = (value: string) => {
    const employee = availableEmployees.find(emp => emp.id.toString() === value);
    const newEmployeeId = employee?.id || 0;

    onEmployeeChange(employee || null);
    onUpdateUrlParams({ employeeId: newEmployeeId });
  };

  return (
    <div>
      <Label htmlFor="employee">Nhân viên *</Label>
      {projectId ? (
        <div className="space-y-2">
          {isLoadingProjectEmployees ? (
            <div className="flex items-center justify-center p-4 border rounded-xl">
              <div className="animate-spin rounded-full h-4 w-4 border-2 border-current border-t-transparent mr-2" />
              Đang tải nhân viên...
            </div>
          ) : availableEmployees.length > 0 ? (
            <Select
              value={selectedEmployee?.id.toString() || ''}
              onValueChange={handleEmployeeSelect}
            >
              <SelectTrigger>
                <SelectValue placeholder="Chọn nhân viên từ dự án" />
              </SelectTrigger>
              <SelectContent>
                {availableEmployees.map((employee) => (
                  <SelectItem key={employee.id} value={employee.id.toString()}>
                    <div className="flex flex-col">
                      <span className="font-medium">{employee.fullname}</span>
                      {employee.position && (
                        <span className="text-sm text-muted-foreground">
                          Vị trí: {employee.position}
                        </span>
                      )}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <div className="p-4 border rounded-xl text-muted-foreground">
              Không có nhân viên nào đang làm việc trong dự án này
            </div>
          )}
        </div>
      ) : (
        <div className="p-4 border rounded-xl text-muted-foreground">
          Vui lòng chọn dự án trước
        </div>
      )}
    </div>
  );
});

EmployeeSelector.displayName = 'EmployeeSelector';