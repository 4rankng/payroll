import { useMutation, useQueryClient } from '@tanstack/react-query';
import { timesheetService } from '@/services/api/timesheet.service';
import { showErrorNotification } from '@/utils/error-handler';
import type { PartnerImportFile } from '@/types/api/timesheet.types';

interface UploadBCCVariables {
  file: File;
  projectId: number;
  forMonth: string;
  idempotencyKey: string;
}

export function useUploadBCCTimesheet() {
  const queryClient = useQueryClient();

  return useMutation<PartnerImportFile, Error, UploadBCCVariables>({
    mutationFn: ({ file, projectId, forMonth, idempotencyKey }) =>
      timesheetService.uploadBCCFile(file, projectId, forMonth, idempotencyKey),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['partner-imports'] });
    },
    onError: (error: unknown) => {
      const e = error as { response?: { data?: { message?: string } }; message?: string };
      showErrorNotification(
        e?.response?.data?.message || e?.message || 'Có lỗi xảy ra khi tải lên file BCC'
      );
    },
  });
}
