import { useState } from 'react';
import { userService, type UserFilters } from '@/services/api/user.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

export const useUserExport = () => {
  const [isExporting, setIsExporting] = useState(false);

  const exportUsers = async (filters?: UserFilters) => {
    setIsExporting(true);

    try {
      const response = await userService.exportUsers(filters);
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
    exportUsers,
    isExporting,
  };
};
