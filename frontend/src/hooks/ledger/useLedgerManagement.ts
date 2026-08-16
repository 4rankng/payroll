import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ledgerService } from '@/services/api/ledger.service';
import { toast } from '@/components/ui/sonner';
import type { CreateLedgerEntry, CreateTransactionRequest } from '@/types/api/financial.types';

export const useLedgerManagement = () => {
  const queryClient = useQueryClient();

  const createEntryMutation = useMutation({
    mutationFn: (data: CreateLedgerEntry) => ledgerService.createEntry(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-overall-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-account-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-project-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-cash-flow'] });
      toast({
        title: 'Thành công',
        description: 'Đã tạo bút toán mới.',
      });
    },
    onError: (error: Error) => {
      toast({
        title: 'Lỗi',
        description: error.message || 'Không thể tạo bút toán. Vui lòng thử lại.',
        variant: 'destructive',
      });
    },
  });


  const reverseEntryMutation = useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      ledgerService.reverseEntry(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-entry'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-overall-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-account-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-project-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-cash-flow'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
      toast({
        title: 'Thành công',
        description: 'Đã tạo bút toán đảo ngược.',
      });
    },
    onError: (error: Error) => {
      toast({
        title: 'Lỗi',
        description: error.message || 'Không thể tạo bút toán đảo ngược. Vui lòng thử lại.',
        variant: 'destructive',
      });
    },
  });

  const recalculateBalancesMutation = useMutation({
    mutationFn: () => ledgerService.recalculateBalances(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-overall-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-account-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-project-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-cash-flow'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
      toast({
        title: 'Thành công',
        description: 'Đã tính lại số dư sổ cái.',
      });
    },
    onError: (error: Error) => {
      toast({
        title: 'Lỗi',
        description: error.message || 'Không thể tính lại số dư. Vui lòng thử lại.',
        variant: 'destructive',
      });
    },
  });

  const importOnePayFeeReportMutation = useMutation({
    mutationFn: (file: File) => ledgerService.importOnePayFeeReport(file),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-overall-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-account-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-project-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-cash-flow'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
      queryClient.invalidateQueries({ queryKey: ['transactions'] });
      toast({
        title: 'Thành công',
        description: `Đã tạo chi phí OnePay ${data.summary.total_fee.toLocaleString('vi-VN')} ₫.`,
      });
    },
    onError: (error: Error) => {
      toast({
        title: 'Không thể nhập phí OnePay',
        description: error.message || 'File chưa khớp dữ liệu đối soát.',
        variant: 'destructive',
      });
    },
  });

  const runWalletSettlementMutation = useMutation({
    mutationFn: () => ledgerService.runWalletSettlement(),
    onSuccess: () => {
      toast({
        title: 'Đã kích chạy chốt lương',
        description: 'Các bản ghi "Wallet disbursement" sẽ xuất hiện trong vài giây.',
      });
      // The consolidation runs async on the asynq worker (~seconds), so the new
      // ledger rows land shortly after the enqueue returns. Re-invalidate the
      // ledger/transaction queries on a short stagger to catch the rows across a
      // range of worker latencies, rather than a single brittle guess. The
      // queryClient is app-scoped, so timers firing after unmount are harmless.
      [2000, 5000].forEach((ms) => {
        setTimeout(() => {
          queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
          queryClient.invalidateQueries({ queryKey: ['ledger-overall-balance'] });
          queryClient.invalidateQueries({ queryKey: ['ledger-account-balance'] });
          queryClient.invalidateQueries({ queryKey: ['ledger-cash-flow'] });
          queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
          queryClient.invalidateQueries({ queryKey: ['transactions'] });
        }, ms);
      });
    },
    onError: (error: Error) => {
      toast({
        title: 'Lỗi',
        description: error.message || 'Không thể kích chạy chốt lương. Vui lòng thử lại.',
        variant: 'destructive',
      });
    },
  });

  const createTransactionMutation = useMutation({
    mutationFn: (data: CreateTransactionRequest) => ledgerService.createTransaction(data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['ledger-entries'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-overall-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-account-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-project-balance'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-cash-flow'] });
      queryClient.invalidateQueries({ queryKey: ['ledger-summary'] });
      toast({
        title: 'Thành công',
        description: `Đã tạo giao dịch với ${data.total_entries} bút toán.`,
      });
    },
    onError: (error: Error) => {
      toast({
        title: 'Lỗi',
        description: error.message || 'Không thể tạo giao dịch. Vui lòng thử lại.',
        variant: 'destructive',
      });
    },
  });

  return {
    createEntry: createEntryMutation.mutateAsync,
    reverseEntry: reverseEntryMutation.mutateAsync,
    recalculateBalances: recalculateBalancesMutation.mutateAsync,
    importOnePayFeeReport: importOnePayFeeReportMutation.mutateAsync,
    createTransaction: createTransactionMutation.mutateAsync,
    runWalletSettlement: runWalletSettlementMutation.mutateAsync,
    isCreating: createEntryMutation.isPending,
    isReversing: reverseEntryMutation.isPending,
    isRecalculating: recalculateBalancesMutation.isPending,
    isImportingOnePayFeeReport: importOnePayFeeReportMutation.isPending,
    isCreatingTransaction: createTransactionMutation.isPending,
    isRunningWalletSettlement: runWalletSettlementMutation.isPending,
  };
};
