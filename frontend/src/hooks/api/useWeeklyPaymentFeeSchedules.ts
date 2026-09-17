// React Query hooks for the admin "Phí trả lương tuần" feature.
// Centralizes loading state, cache invalidation, and toast handling so
// page/dialog components stay UI-only.

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { weeklyPaymentFeeScheduleService } from "@/services/api/weekly-payment-fee-schedule.service";
import { QueryKeys } from "@/lib/queryKeys";
import {
  showErrorNotification,
  showSuccessNotification,
} from "@/utils/error-handler";
import type {
  CreateWeeklyPaymentFeeScheduleRequest,
  UpdateWeeklyPaymentFeeScheduleRequest,
  WeeklyPaymentFeeScheduleEntry,
} from "@/types/api/weekly-payment-fee-schedule.types";

export function useWeeklyPaymentFeeSchedules() {
  return useQuery({
    queryKey: QueryKeys.advancePayments.admin.weeklyPaymentFeeSchedules,
    queryFn: async (): Promise<WeeklyPaymentFeeScheduleEntry[]> => {
      const res = await weeklyPaymentFeeScheduleService.list();
      return res.data?.entries ?? [];
    },
  });
}

export function useCreateWeeklyPaymentFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateWeeklyPaymentFeeScheduleRequest) =>
      weeklyPaymentFeeScheduleService.create(body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.weeklyPaymentFeeSchedules,
      });
      showSuccessNotification(
        response.message ?? "Tạo cấu hình phí trả lương tuần thành công",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

export function useUpdateWeeklyPaymentFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      body,
    }: {
      id: string;
      body: UpdateWeeklyPaymentFeeScheduleRequest;
    }) => weeklyPaymentFeeScheduleService.update(id, body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.weeklyPaymentFeeSchedules,
      });
      showSuccessNotification(
        response.message ?? "Cập nhật cấu hình phí trả lương tuần thành công",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

export function useDeleteWeeklyPaymentFeeSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => weeklyPaymentFeeScheduleService.delete(id),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: QueryKeys.advancePayments.admin.weeklyPaymentFeeSchedules,
      });
      showSuccessNotification(response.message ?? "Đã xóa cấu hình phí");
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}
