import type { Employee } from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";

export interface EmployeeDetailsProps {
  employee: Employee;
}

export interface EmployeeFormProps {
  employee: Employee;
  formData: Partial<Employee>;
  selectedBank: Bank | null;
  onInputChange: (field: keyof Employee, value: string) => void;
  onBankChange: (bank: Bank | null) => void;
  canCreateBank?: boolean;
}

export interface EmployeeActionsProps {
  onEdit?: () => void;
  onSave?: () => void;
  onCancel?: () => void;
  onDelete: () => void;
  onResetPassword?: () => void;
  onClose?: () => void;
  isDirty?: boolean;
}