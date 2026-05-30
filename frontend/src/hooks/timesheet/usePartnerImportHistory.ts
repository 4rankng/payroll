import { useQuery } from '@tanstack/react-query';
import { timesheetService } from '@/services/api/timesheet.service';
import type { PartnerImportListParams } from '@/types/api/timesheet.types';

interface UsePartnerImportHistoryOptions {
  params?: PartnerImportListParams;
  enabled?: boolean;
}

export function usePartnerImportHistory({ params, enabled = true }: UsePartnerImportHistoryOptions = {}) {
  return useQuery({
    queryKey: ['partner-imports', params],
    queryFn: () => timesheetService.listPartnerImports(params),
    staleTime: 30_000,
    enabled: enabled && !!params,
  });
}
