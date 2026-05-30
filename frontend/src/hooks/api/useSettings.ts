import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsService } from '@/services/api/settings.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification } from '@/utils/error-handler';
import type { SettingsFilters, UpdateSettingData } from '@/types/api/settings.types';

// Get paginated settings list
export const useSettings = (filters?: SettingsFilters) => {
  return useQuery({
    queryKey: QueryKeys.settings.list(filters),
    queryFn: () => settingsService.getSettings(filters),
  });
};

// Get single setting by ID
export const useSetting = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.settings.detail(id),
    queryFn: () => settingsService.getSettingById(id),
    enabled,
  });
};

// Get single setting by key
export const useSettingByKey = (key: string, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.settings.byKey(key),
    queryFn: () => settingsService.getSettingByKey(key),
    enabled: enabled && key.length > 0,
  });
};

// Update setting
export const useUpdateSetting = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateSettingData }) =>
      settingsService.updateSetting(id, data),
    onSuccess: (response) => {
      // Invalidate all settings queries to refresh
      queryClient.invalidateQueries({ queryKey: QueryKeys.settings.all });

      if (response?.message) {
        showSuccessNotification(response.message);
      }
    },
    // Error handling is now done globally in React Query - will display response.message from backend
  });
};

// Get multiple settings by keys
export const useMultipleSettings = (keys: string[]) => {
  return useQuery({
    queryKey: [...QueryKeys.settings.all, 'multiple', keys],
    queryFn: async () => {
      const allSettings = await settingsService.getSettings();
      return allSettings.filter(setting => keys.includes(setting.key));
    },
    enabled: keys.length > 0,
  });
};
