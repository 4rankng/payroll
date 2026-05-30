import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { EmployeeDetailsSheet as EmployeeDetailsSheetComponent } from "@/components/employees/details/EmployeeDetailsSheet";
import { useEmployee } from "@/hooks/api/useEmployees";
import { useSearchParams } from "react-router-dom";
import type { Employee } from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";
import type { ModalConfig } from "@/types/modal-config.types";

export const modalConfig: ModalConfig = {
  id: 'employee_details_sheet',
  name: 'Chi tiết nhân viên',
  description: 'Xem và chỉnh sửa thông tin chi tiết của nhân viên, bao gồm phân quyền truy cập.',
  category: 'employee',
  permissions: {
    action: 'read',
    subject: 'Employee',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['id'],
    example: '?modal=employee_details_sheet&id=123',
    validateParams: (params) => {
      if (!params || Object.keys(params).length === 0) return true;
      return !!(params.id && !isNaN(Number(params.id)));
    }
  },
  requiresAuth: true,
  encryptData: false,
};

interface EmployeeDetailsSheetProps {
  employee?: Employee | null;
  isOpen: boolean;
  onClose: () => void;
  onUpdate?: (employeeId: number, employeeData: Partial<Employee>, bankObject?: Bank | null) => void;
  onDelete?: (employee: Employee) => void;
}

/**
 * Simple container component for EmployeeDetailsSheet with local state management
 * Uses local state for tab switching - no URL synchronization
 */
export function EmployeeDetailsSheet({
  employee: propEmployee,
  isOpen,
  onClose,
  onUpdate,
  onDelete
}: EmployeeDetailsSheetProps) {
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();

  const employeeId = searchParams.get('id');

  const { data: fetchedEmployee } = useEmployee(
    employeeId ? parseInt(employeeId) : 0,
    isOpen && !!employeeId && !propEmployee
  );

  const employee = propEmployee || fetchedEmployee;

  const handleClose = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['employees'] });
    onClose();
  }, [queryClient, onClose]);

  const handleRefetch = useCallback(() => {
    if (employeeId && !propEmployee) {
      queryClient.invalidateQueries({ queryKey: ['employee', parseInt(employeeId)] });
    }
  }, [employeeId, propEmployee, queryClient]);

  return (
    <EmployeeDetailsSheetComponent
      employee={employee}
      isOpen={isOpen}
      onClose={handleClose}
      onUpdate={onUpdate}
      onDelete={onDelete}
      onRefetch={handleRefetch}
    />
  );
}

// Default export for backward compatibility
export default EmployeeDetailsSheet;
