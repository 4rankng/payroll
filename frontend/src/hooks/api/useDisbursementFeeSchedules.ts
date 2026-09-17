// React Query hooks for the admin "Cấu hình phí giao dịch chi hộ" feature.
// Mirrors useAdvancePaymentFeeSchedules so dialog/page components stay UI-only.

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { disbursementFeeScheduleService } from "@/services/api/disbursement-fee-schedule.service";
import { QueryKeys } from "@/lib/queryKeys";
import {
  showErrorNotification,
  showSuccessNotification,
} from "@/utils/error-handler";
import type {
  CreateDisbursementFeeScheduleRequest,
  DisbursementFeeScheduleEntry,
  UpdateDisbursementFeeScheduleRequest,
} from "@/types/api/disbursement-fee-schedule.types";

export function useDisbursementFeeSchedules() {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.disbursementFeeSchedules,
    queryFn: async (): Promise<DisbursementFeeScheduleEntry[]> => {
      const res = await disbursementFeeScheduleService.list();
      return res.data?.entries ?? [];
    },
  });
}

// Enabled disbursement providers for this environment, straight from the
// schedule list response. Empty until the list query resolves; fall back to
// the static catalog when the backend omits the field.
export function useDisbursementFeeProviders(): string[] {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.disbursementFeeSchedules,
    queryFn: () => disbursementFeeScheduleService.list(),
    select: (res) => res.data?.providers ?? [],
  }).data ?? [];
}

export function useCreateDisbursementFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateDisbursementFeeScheduleRequest) =>
      disbursementFeeScheduleService.create(body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.disbursementFeeSchedules,
      });
      showSuccessNotification(
        response.message ?? "Tạo cấu hình phí giao dịch chi hộ thành công",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

export function useUpdateDisbursementFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      body,
    }: {
      id: string;
      body: UpdateDisbursementFeeScheduleRequest;
    }) => disbursementFeeScheduleService.update(id, body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.disbursementFeeSchedules,
      });
      showSuccessNotification(
        response.message ?? "Cập nhật cấu hình phí giao dịch chi hộ thành công",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

export function useDeleteDisbursementFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => disbursementFeeScheduleService.delete(id),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.disbursementFeeSchedules,
      });
      showSuccessNotification(
        response.message ?? "Đã xóa cấu hình phí giao dịch chi hộ",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}
