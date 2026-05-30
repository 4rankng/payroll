import { useState, useMemo } from 'react';
import { toast } from '@/components/ui/sonner';
import { Employee } from '@/types/api/employee.types';

interface UseEmployeeSelectionProps {
  bulkUpdateEmployeeStatus: (ids: number[], status: Employee['status']) => Promise<void>;
  displayedEmployees: Employee[];
}

export const useEmployeeSelection = ({
  bulkUpdateEmployeeStatus,
  displayedEmployees,
}: UseEmployeeSelectionProps) => {
  const [selectedEmployeeIds, setSelectedEmployeeIds] = useState<number[]>([]);

  // BUG-007 fix: Memoize computed selectedEmployees
  const selectedEmployees = useMemo(
    () => displayedEmployees.filter(emp => selectedEmployeeIds.includes(emp.id)),
    [displayedEmployees, selectedEmployeeIds]
  );

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedEmployeeIds(displayedEmployees.map(emp => emp.id));
    } else {
      setSelectedEmployeeIds([]);
    }
  };

  const handleSelectEmployee = (employeeId: number, checked: boolean) => {
    if (checked) {
      setSelectedEmployeeIds([...selectedEmployeeIds, employeeId]);
    } else {
      setSelectedEmployeeIds(selectedEmployeeIds.filter(id => id !== employeeId));
    }
  };

  // BUG-001/006 fix: Await the bulk update and show toast only on success
  const handleBulkStatusUpdate = async (status: Employee['status']) => {
    if (selectedEmployeeIds.length === 0) {
      toast({
        title: "Lỗi",
        description: "Vui lòng chọn ít nhất một nhân viên.",
        variant: "destructive"
      });
      return;
    }

    try {
      await bulkUpdateEmployeeStatus(selectedEmployeeIds, status);
      setSelectedEmployeeIds([]);
      toast({
        title: "Thành công",
        description: `Đã cập nhật trạng thái cho ${selectedEmployeeIds.length} nhân viên.`,
      });
    } catch {
      toast({
        title: "Lỗi",
        description: "Cập nhật trạng thái thất bại. Vui lòng thử lại.",
        variant: "destructive"
      });
    }
  };

  const clearSelection = () => {
    setSelectedEmployeeIds([]);
  };

  return {
    selectedEmployees, // Returns Employee objects
    selectedEmployeeIds, // Returns just the IDs  
    setSelectedEmployeeIds,
    handleSelectAll,
    handleSelectEmployee,
    handleBulkStatusUpdate,
    clearSelection,
  };
};