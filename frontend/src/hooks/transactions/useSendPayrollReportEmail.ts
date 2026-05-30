import { useMutation } from '@tanstack/react-query';
import { transactionService } from '@/services/api/transaction.service';
import { showErrorNotification, showSuccessNotification } from '@/utils/error-handler';

export function useSendPayrollReportEmail() {
  return useMutation({
    mutationFn: (params: { reportAtDate: string; recipients: string[]; cc?: string[]; bcc?: string[] }) =>
      transactionService.sendPayrollReportEmail(params),
    onSuccess: () => {
      showSuccessNotification('Email sao kê đã được gửi thành công');
    },
    onError: (error: unknown) => {
      showErrorNotification(error, 'Gửi email sao kê thất bại');
    },
  });
}
