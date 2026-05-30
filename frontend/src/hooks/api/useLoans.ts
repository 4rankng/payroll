import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { loanService } from '@/services/api/loan.service';
import { QueryKeys } from '@/lib/queryKeys';
import { showSuccessNotification } from '@/utils/error-handler';
import type {
  Lender,
  LenderWithSummary,
  CreateLenderRequest,
  UpdateLenderRequest,
  LenderFilters,
  Loan,
  CreateLoanRequest,
  DisburseLoanRequest,
  RepayLoanRequest,
  RepayScheduleRequest,
  UpdateLoanRequest,
  LoanFilters,
  ScheduleItem,
} from '@/types/api/loan.types';
import type { ApiResponse } from '@/services/api/client';

// ============================================================================
// Lenders Hooks
// ============================================================================

/**
 * Get paginated list of lenders
 */
export const useLenders = (filters?: LenderFilters, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: QueryKeys.lenders.list(filters),
    queryFn: () => loanService.getLenders(filters),
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
};

/**
 * Get single lender by ID with summary
 */
export const useLender = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.lenders.detail(id),
    queryFn: () => loanService.getLenderById(id),
    enabled: enabled && !!id,
  });
};

/**
 * Create new lender
 */
export const useCreateLender = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateLenderRequest) => loanService.createLender(data),
    onSuccess: async (response) => {
      const newLender = response.data!;
      // Note: Avoid setting detail cache with incomplete type (Lender vs LenderWithSummary)
      // The detail query will be fetched when needed.

      // Update lenders list cache (guard array shape)
      queryClient.setQueriesData(
        { queryKey: QueryKeys.lenders.lists() },
        (oldData: ApiResponse<Lender[]> | undefined) => {
          if (
            oldData &&
            Object.prototype.hasOwnProperty.call(oldData, 'data') &&
            Array.isArray(oldData.data)
          ) {
            return {
              ...oldData,
              data: [newLender, ...oldData.data],
              pagination: oldData.pagination
                ? {
                    ...oldData.pagination,
                    totalRecords: oldData.pagination.totalRecords + 1,
                  }
                : undefined,
            };
          }
          return oldData;
        }
      );

      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Update lender
 */
export const useUpdateLender = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateLenderRequest }) =>
      loanService.updateLender(id, data),
    onSuccess: async (response) => {
      const updatedLender = response.data!;

      // Optimistically update the lender detail
      queryClient.setQueryData(
        QueryKeys.lenders.detail(updatedLender.id),
        updatedLender
      );

      // Invalidate lists to refresh
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Delete lender
 */
export const useDeleteLender = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => loanService.deleteLender(id),
    onSuccess: async (response, deletedId) => {
      // Remove the lender detail query
      queryClient.removeQueries({ queryKey: QueryKeys.lenders.detail(deletedId) });

      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

// ============================================================================
// Loans Hooks
// ============================================================================

/**
 * Get paginated list of loans
 */
export const useLoans = (filters?: LoanFilters, options?: { enabled?: boolean }) => {
  return useQuery({
    queryKey: QueryKeys.loans.list(filters),
    queryFn: () => loanService.getLoans(filters),
    enabled: options?.enabled !== undefined ? options.enabled : true,
  });
};

/**
 * Get single loan by ID
 */
export const useLoan = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.loans.detail(id),
    queryFn: () => loanService.getLoanById(id),
    enabled: enabled && !!id,
  });
};

/**
 * Get loan payment schedule
 */
export const useLoanSchedule = (id: number, enabled = true) => {
  return useQuery({
    queryKey: QueryKeys.loans.schedule(id),
    queryFn: () => loanService.getLoanSchedule(id),
    enabled: enabled && !!id,
  });
};

/**
 * Create new loan
 */
export const useCreateLoan = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateLoanRequest) => loanService.createLoan(data),
    onSuccess: async (response) => {
      const newLoan = response.data!;

      // Set the new loan in cache for detail views
      if (newLoan.id) {
        queryClient.setQueryData(QueryKeys.loans.detail(newLoan.id), newLoan);
      }

      // Update loans list cache (guard array shape)
      queryClient.setQueriesData(
        { queryKey: QueryKeys.loans.lists() },
        (oldData: ApiResponse<Loan[]> | undefined) => {
          if (
            oldData &&
            Object.prototype.hasOwnProperty.call(oldData, 'data') &&
            Array.isArray(oldData.data)
          ) {
            return {
              ...oldData,
              data: [newLoan, ...oldData.data],
              pagination: oldData.pagination
                ? {
                    ...oldData.pagination,
                    totalRecords: oldData.pagination.totalRecords + 1,
                  }
                : undefined,
            };
          }
          return oldData;
        }
      );

      // Invalidate to ensure consistency
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.lists() });
      // Also invalidate lender details as the summary might have changed
      if (newLoan.lender.id) {
        queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.detail(newLoan.lender.id) });
        queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });
      }

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Disburse a loan
 */
export const useDisburseLoan = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: DisburseLoanRequest }) =>
      loanService.disburseLoan(id, data),
    onSuccess: async (response, variables) => {
      const { id: loanId } = variables;

      // Invalidate loan detail to refetch with updated disbursement status
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.detail(loanId) });
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.lists() });

      // Invalidate ledger entries
      queryClient.invalidateQueries({ queryKey: ['ledger'] });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Repay principal (full or partial)
 */
export const useRepayLoan = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: RepayLoanRequest }) =>
      loanService.repayPrincipal(id, data),
    onSuccess: async (response, variables) => {
      const { id: loanId } = variables;

      // Invalidate loan detail to refetch with updated principal
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.detail(loanId) });
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.lists() });
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.schedule(loanId) });

      // Invalidate ledger entries
      queryClient.invalidateQueries({ queryKey: ['ledger'] });

      // Invalidate lender details (outstanding principal changed)
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Process scheduled payment for custom schedule loans
 */
export const useRepaySchedule = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: RepayScheduleRequest }) =>
      loanService.repaySchedule(id, data),
    onSuccess: async (response, variables) => {
      const { id: loanId } = variables;

      // Invalidate loan detail to refetch with updated schedule status
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.detail(loanId) });
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.lists() });
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.schedule(loanId) });

      // Invalidate ledger entries
      queryClient.invalidateQueries({ queryKey: ['ledger'] });

      // Invalidate lender details (outstanding principal changed)
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Update loan metadata
 */
export const useUpdateLoan = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateLoanRequest }) =>
      loanService.updateLoan(id, data),
    onSuccess: async (response) => {
      const updatedLoan = response.data!;

      // Optimistically update the loan detail
      queryClient.setQueryData(
        QueryKeys.loans.detail(updatedLoan.id),
        updatedLoan
      );

      // Invalidate lists to refresh
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};

/**
 * Delete loan (only if not disbursed)
 */
export const useDeleteLoan = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => loanService.deleteLoan(id),
    onSuccess: async (response, deletedId) => {
      // Remove the loan detail query
      queryClient.removeQueries({ queryKey: QueryKeys.loans.detail(deletedId) });

      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: QueryKeys.loans.lists() });

      // Invalidate lender lists (summary might have changed)
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.lists() });

      if (response.message) {
        showSuccessNotification(response.message);
      }
    },
  });
};
