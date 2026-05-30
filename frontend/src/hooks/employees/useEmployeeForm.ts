import { useState, useEffect, useMemo } from "react";
import { useChangePaymentSchedule, useUpdateEmployeeProjectAssignment } from "@/hooks/api/useProjectEmployees";
import { validateFullname } from "@/lib/validation";
import { toast } from "@/components/ui/sonner";
import type { Employee } from "@/types/api/employee.types";
import type { Bank } from "@/types/api/bank.types";

interface UseEmployeeFormProps {
  employee: Employee | null;
  onUpdate: (employeeId: number, employeeData: Partial<Employee>, bankObject?: Bank | null) => Promise<void> | void;
}

type AssignmentChange = {
  position?: string;
  start_date?: string;
  payment_schedule?: 'weekly' | 'monthly';
  assignmentId?: number;
};

export const useEmployeeForm = ({ employee, onUpdate }: UseEmployeeFormProps) => {
  const [isEditing, setIsEditing] = useState(false);
  const [selectedBank, setSelectedBank] = useState<Bank | null>(null);
  const [projectChanges, setProjectChanges] = useState<Record<number, AssignmentChange>>({});
  const updateEmployeeProjectMutation = useUpdateEmployeeProjectAssignment();
  const changeScheduleMutation = useChangePaymentSchedule();

  const [formData, setFormData] = useState<Partial<Employee>>({
    fullname: "",
    email: "",
    cccd: "",
    address: "",
    mobile: "",
    bank: null,
    bank_account_number: "",
    bank_account_name: "",
    date_of_birth: "",
  });

  useEffect(() => {
    if (employee && !isEditing) {
      setFormData({
        fullname: employee.fullname || "",
        email: employee.email || "",
        cccd: employee.cccd || "",
        address: employee.address || "",
        mobile: employee.mobile || "",
        bank: employee.bank || null,
        bank_account_number: employee.bank_account_number || "",
        bank_account_name: employee.bank_account_name || "",
        date_of_birth: employee.date_of_birth || "",
      });
      setSelectedBank(employee.bank || null);
      setProjectChanges({});
    }
  }, [employee, isEditing]);

  const handleInputChange = (field: keyof Employee, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
  };

  const handleBankChange = (bank: Bank | null) => {
    setSelectedBank(bank);
    setFormData(prev => ({ ...prev, bank }));
  };

  const handleProjectChange = (projectId: number, changes: AssignmentChange | null) => {
    if (changes === null) {
      setProjectChanges(prev => {
        const { [projectId]: _removed, ...rest } = prev;
        return rest;
      });
      return;
    }

    setProjectChanges(prev => {
      const existing = prev[projectId] ?? {};
      const next: AssignmentChange = {
        ...existing,
        ...changes,
      };

      if (changes.assignmentId !== undefined) {
        next.assignmentId = changes.assignmentId;
      }

      (Object.keys(next) as (keyof AssignmentChange)[]).forEach((key) => {
        if (next[key] === undefined) {
          delete next[key];
        }
      });

      // Remove assignmentId if it's the only remaining field
      if (Object.keys(next).length === 1 && next.assignmentId) {
        delete next.assignmentId;
      }

      if (Object.keys(next).length === 0) {
        const { [projectId]: _removed, ...rest } = prev;
        return rest;
      }

      return {
        ...prev,
        [projectId]: next,
      };
    });
  };

  const applyProjectChanges = async (projectId: number, changes: AssignmentChange) => {
    if (!employee) {
      return;
    }

    const { position, start_date, payment_schedule, assignmentId } = changes;
    const hasAssignmentUpdates = position !== undefined || start_date !== undefined;

    if (!hasAssignmentUpdates && payment_schedule === undefined) {
      return;
    }

    const resolveAssignmentId = () => {
      if (assignmentId) {
        return assignmentId;
      }
      return employee.current_projects?.find(
        (project) => project.project_id === projectId
      )?.project_employee_id;
    };

    if (hasAssignmentUpdates) {
      const projectUpdateData: {
        project_id: number;
        position?: string;
        start_date?: string;
        end_date?: string | null;
      } = {
        project_id: projectId
      };

      if (position !== undefined) {
        projectUpdateData.position = position;
      }

      if (start_date !== undefined) {
        projectUpdateData.start_date = start_date;
      }

      await updateEmployeeProjectMutation.mutateAsync({
        employeeId: employee.id,
        data: projectUpdateData
      });
    }

    if (payment_schedule !== undefined) {
      const resolvedAssignmentId = resolveAssignmentId();
      if (resolvedAssignmentId) {
        await changeScheduleMutation.mutateAsync({
          assignmentId: resolvedAssignmentId,
          data: {
            new_schedule: payment_schedule
          }
        });
      }
    }

    setProjectChanges((prev) => {
      const { [projectId]: _removed, ...rest } = prev;
      return rest;
    });
  };

  const handleSave = async () => {
    if (employee) {
      try {
        // Validate fullname before saving
        if (formData.fullname) {
          const fullnameValidation = validateFullname(formData.fullname);
          if (!fullnameValidation.valid) {
            toast({
              title: "Lỗi xác thực",
              description: fullnameValidation.error || "Họ tên không hợp lệ",
              variant: "destructive"
            });
            return;
          }
        }

        // Save employee data
        const { bank, ...restData } = formData;

        // Filter out empty or undefined fields to avoid backend validation errors
        const updateData: Record<string, unknown> = {};

        Object.entries(restData).forEach(([key, value]) => {
          if (value !== "" && value !== null && value !== undefined) {
            updateData[key] = value;
          }
        });

        // Only include bank_id if a bank is selected
        if (bank?.id) {
          updateData.bank_id = bank.id;
        }

        await onUpdate(employee.id, updateData, bank);

        // Save project changes
        for (const [projectIdStr, changes] of Object.entries(projectChanges)) {
          const projectId = parseInt(projectIdStr);
          const { position, start_date, payment_schedule, assignmentId } = changes;
          const hasAssignmentUpdates = position !== undefined || start_date !== undefined;

          // Only call API if there are actual changes
          if (hasAssignmentUpdates) {
            const projectUpdateData: {
              project_id: number;
              position?: string;
              start_date?: string;
              end_date?: string | null;
            } = {
              project_id: projectId
            };

            if (position !== undefined) {
              projectUpdateData.position = position;
            }

            if (start_date !== undefined) {
              projectUpdateData.start_date = start_date;
            }

            await updateEmployeeProjectMutation.mutateAsync({
              employeeId: employee.id,
              data: projectUpdateData
            });
          }

          if (payment_schedule !== undefined && assignmentId) {
            await changeScheduleMutation.mutateAsync({
              assignmentId,
              data: {
                new_schedule: payment_schedule
              }
            });
          }
        }

        // Clear project changes and exit edit mode
        setProjectChanges({});
        setIsEditing(false);
      } catch (error) {
        // Error handling is already done in the API hooks
        console.error('Failed to save employee or project changes:', error);
      }
    }
  };

  const resetFormData = () => {
    if (employee) {
      setFormData({
        fullname: employee.fullname || "",
        email: employee.email || "",
        cccd: employee.cccd || "",
        address: employee.address || "",
        mobile: employee.mobile || "",
        bank: employee.bank || null,
        bank_account_number: employee.bank_account_number || "",
        bank_account_name: employee.bank_account_name || "",
        date_of_birth: employee.date_of_birth || "",
      });
      setSelectedBank(employee.bank || null);
      setProjectChanges({});
    }
  };

  const handleCancel = () => {
    resetFormData();
    setIsEditing(false);
  };

  const isDirty = useMemo(() => {
    if (!employee) return false;

    const fields: (keyof typeof formData)[] = [
      'fullname', 'email', 'cccd', 'address', 'mobile',
      'bank_account_number', 'bank_account_name', 'date_of_birth',
    ];

    for (const f of fields) {
      if ((formData[f] ?? '') !== (employee[f] ?? '')) return true;
    }

    if ((selectedBank?.id ?? null) !== (employee.bank?.id ?? null)) return true;

    if (Object.keys(projectChanges).length > 0) return true;

    return false;
  }, [employee, formData, selectedBank, projectChanges]);

  return {
    isEditing,
    setIsEditing,
    formData,
    selectedBank,
    handleInputChange,
    handleBankChange,
    handleSave,
    handleCancel,
    resetFormData,
    projectChanges,
    handleProjectChange,
    applyProjectChanges,
    isDirty,
  };
};
