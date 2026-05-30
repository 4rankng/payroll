import { useState } from 'react';
import { employeeService } from '@/services/api/employee.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

export const useEmployeeExport = () => {
  const [isExporting, setIsExporting] = useState(false);

  const exportEmployees = async (projectIds: number[]) => {
    setIsExporting(true);

    try {
      const filters = projectIds.length > 0
        ? { projectIds: projectIds.join(',') }
        : undefined;
      const response = await employeeService.exportEmployees(filters);
      if (response?.message) {
        showSuccessNotification(response.message);
      }
    } catch (error) {
      showErrorNotification(error);
    } finally {
      setIsExporting(false);
    }
  };

  return {
    exportEmployees,
    isExporting
  };
};
