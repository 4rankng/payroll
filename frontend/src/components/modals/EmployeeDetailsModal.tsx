import { useSearchParams } from "react-router-dom";
import { EmployeeDetailsSheet } from "@/components/sheets/EmployeeDetailsSheet";
import { useEmployee, useUpdateEmployee, useDeleteEmployee } from "@/hooks/api/useEmployees";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import type { Employee, UpdateEmployeeData } from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";

export const modalConfig = {
  id: 'employee-details',
};

/**
 * Route-based Employee Details Modal
 * Fetches the specific employee by ID — no need to load the full list.
 */
export function EmployeeDetailsModal() {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const updateEmployeeMutation = useUpdateEmployee();
  const deleteEmployeeMutation = useDeleteEmployee();

  const modalId = searchParams.get('modal');
  const employeeId = searchParams.get('id');
  const isOpen = modalId === 'employee_details' && Boolean(employeeId);
  const numericId = employeeId ? parseInt(employeeId, 10) : 0;

  // Fetch only the specific employee — no full-list fetch
  const { data: fetchedEmployee } = useEmployee(numericId, isOpen && !!numericId);
  const selectedEmployee = fetchedEmployee ?? null;

  const handleClose = () => {
    closeModal();
  };

  const handleUpdate = async (employeeId: number, employeeData: Partial<Employee>, bankObject?: Bank | null) => {
    const updateData = employeeData as UpdateEmployeeData;
    return updateEmployeeMutation.mutateAsync({ id: employeeId, data: updateData, bank: bankObject });
  };

  const handleDelete = (employee: Employee) => {
    deleteEmployeeMutation.mutate(employee.id, {
      onSuccess: () => {
        handleClose();
      }
    });
  };

  if (!isOpen) return null;

  return (
    <EmployeeDetailsSheet
      employee={selectedEmployee}
      isOpen={isOpen}
      onClose={handleClose}
      onUpdate={handleUpdate}
      onDelete={handleDelete}
    />
  );
}
