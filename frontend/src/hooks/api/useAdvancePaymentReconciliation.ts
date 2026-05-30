import { useMutation, useQueryClient } from '@tanstack/react-query';
import { advancePaymentService } from '@/services/api/advance-payment.service';
import { showSuccessNotification } from '@/utils/error-handler';
import type { ApiResponse } from '@/services/api/client';

// Reconciliation email data type
export interface SendReconciliationEmailData {
  forMonth: string;
  recipients: string[];
  cc?: string[];
}

// Upload and settle parameters
export interface UploadAndSettleReconciliationData {
  file: File;
  forMonth?: string;
}


export const useSendReconciliationEmail = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: SendReconciliationEmailData) =>
      advancePaymentService.sendReconciliationEmail(data),
    onSuccess: (response, variables) => {
      showSuccessNotification(`Đã gửi email thành công cho tháng ${variables.forMonth}`);
      // You might want to invalidate or refresh email history if needed
      queryClient.invalidateQueries({ queryKey: ['email'] });
    },
  });
};

export const useUploadAndSettleReconciliation = () => {
  return useMutation({
    mutationFn: (formData: FormData) =>
      advancePaymentService.uploadAndSettleReconciliation(formData),
    onSuccess: (response) => {
      showSuccessNotification('Tải lên và xử lý file điều chỉnh thành công');
      // You might want to refresh the advance payments list
    },
  });
};