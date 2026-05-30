// React Query hooks for the admin "Cấu hình phí ứng lương" feature.
// Centralizes loading state, cache invalidation, and toast handling so
// page/dialog components stay UI-only.

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { advancePaymentFeeScheduleService } from "@/services/api/advance-payment-fee-schedule.service";
import { QueryKeys } from "@/lib/queryKeys";
import {
  showErrorNotification,
  showSuccessNotification,
} from "@/utils/error-handler";
import type {
  CreateFeeScheduleRequest,
  FeeScheduleEntry,
  UpdateFeeScheduleRequest,
} from "@/types/api/advance-payment-fee-schedule.types";

export function useFeeSchedules() {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.feeSchedules,
    queryFn: async (): Promise<FeeScheduleEntry[]> => {
      const res = await advancePaymentFeeScheduleService.list();
      return res.data?.entries ?? [];
    },
  });
}

export function useCreateFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateFeeScheduleRequest) =>
      advancePaymentFeeScheduleService.create(body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.feeSchedules,
      });
      showSuccessNotification(response.message ?? "Tạo cấu hình phí thành công");
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

export function useUpdateFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      body,
    }: {
      id: string;
      body: UpdateFeeScheduleRequest;
    }) => advancePaymentFeeScheduleService.update(id, body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.feeSchedules,
      });
      showSuccessNotification(
        response.message ?? "Cập nhật cấu hình phí thành công",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

export function useDeleteFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => advancePaymentFeeScheduleService.delete(id),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.feeSchedules,
      });
      showSuccessNotification(response.message ?? "Đã xóa cấu hình phí");
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}
