import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsService } from '@/services/api/settings.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification } from '@/utils/error-handler';
import type { CreateSettingData, SettingsFilters, UpdateSettingData } from '@/types/api/settings.types';
import { REFERENCE_DATA_STALE_TIME_MS } from '@/lib/cache/queryCacheTimes';

// Get paginated settings list
export const useSettings = (filters?: SettingsFilters) => {
  return useQuery({
    queryKey: QueryKeys.settings.list(filters),
    queryFn: () => settingsService.getSettings(filters),
    staleTime: REFERENCE_DATA_STALE_TIME_MS, // Admin saves invalidate these keys on success
  });
};

// Get single setting by ID
export const useSetting = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.settings.detail(id),
    queryFn: () => settingsService.getSettingById(id),
    enabled,
    staleTime: REFERENCE_DATA_STALE_TIME_MS,
  });
};

// Get single setting by key
export const useSettingByKey = (key: string, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.settings.byKey(key),
    queryFn: () => settingsService.getSettingByKey(key),
    enabled: enabled && key.length > 0,
    staleTime: REFERENCE_DATA_STALE_TIME_MS,
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

// Create setting (upsert path for keys that may not exist yet)
export const useCreateSetting = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateSettingData) => settingsService.createSetting(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.settings.all });
    },
  });
};

// Get multiple settings by keys. Missing keys (never-configured rows) resolve
// to undefined instead of failing the whole query.
export const useMultipleSettings = (keys: string[]) => {
  return useQuery({
    queryKey: [...QueryKeys.settings.all, 'multiple', keys],
    queryFn: async () => {
      const results = await Promise.allSettled(
        keys.map((key) => settingsService.getSettingByKey(key)),
      );
      return results.map((result) => (result.status === 'fulfilled' ? result.value : undefined));
    },
    enabled: keys.length > 0,
    staleTime: REFERENCE_DATA_STALE_TIME_MS,
  });
};
