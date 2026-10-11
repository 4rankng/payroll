import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { approvalsService } from '@/services/api/approvals.service';
import { QueryKeys } from '@/lib/queryKeys';
import type { ApprovalListFilters, FileApprovalData } from '@/types/api/approval.types';

/** Approval decisions are low-frequency and admin-facing: 30s stale keeps the
 * Tổng Quan counts fresh without polling the engine on every mount. */
const APPROVALS_STALE_TIME_MS = 30_000;

export const useApprovalsList = (filters?: ApprovalListFilters) =>
  useQuery({
    queryKey: QueryKeys.approvals.list(filters as unknown as Record<string, unknown>),
    queryFn: () => approvalsService.getApprovals(filters),
    staleTime: APPROVALS_STALE_TIME_MS,
  });

export const useApprovalDetail = (id: number, enabled = true) =>
  useQuery({
    queryKey: QueryKeys.approvals.detail(id),
    queryFn: () => approvalsService.getApproval(id),
    enabled: enabled && id > 0,
    staleTime: APPROVALS_STALE_TIME_MS,
  });

const useApprovalDecision = (
  mutate: (id: number, note: string) => Promise<unknown>,
  successMessage: string
) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, note = '' }: { id: number; note?: string }) => mutate(id, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.approvals.all });
    },
    meta: { successMessage },
  });
};

export const useApproveApproval = () =>
  useApprovalDecision((id) => approvalsService.approve(id), 'Đã duyệt yêu cầu');

export const useRejectApproval = () =>
  useApprovalDecision((id, note) => approvalsService.reject(id, note), 'Đã từ chối — sẽ thử lại theo lịch');

export const useCancelApproval = () =>
  useApprovalDecision((id, note) => approvalsService.cancel(id, note), 'Đã huỷ yêu cầu');

export const useFileApproval = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: FileApprovalData) => approvalsService.fileApproval(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.approvals.all });
    },
  });
};
