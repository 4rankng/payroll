// React Query hooks for the admin "Chuyển tiền" (Manual Disbursement)
// page. Polling cadence is the spec's: every 5s for the first 5 minutes,
// then every 30s, capped at 1 hour total — derived from the row's age,
// so unmount/remount preserves the timeline.

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { manualDisbursementService } from "@/services/api/manual-disbursement.service";
import { QueryKeys } from "@/lib/queryKeys";
import {
  showErrorNotification,
  showSuccessNotification,
} from "@/utils/error-handler";
import type {
  BankInfoResponse,
  InitiateManualDisbursementRequest,
  ManualDisbursementResponse,
  VerifyAccountRequest,
  VerifyAccountResponse,
} from "@/types/api/manual-disbursement.types";
import { isTerminalStatus } from "@/types/api/manual-disbursement.types";

const FAST_POLL_MS = 5_000;
const SLOW_POLL_MS = 30_000;
const FAST_PHASE_MS = 5 * 60 * 1000; // 5 minutes
const POLL_CUTOFF_MS = 60 * 60 * 1000; // 1 hour

function pollIntervalForRow(row: ManualDisbursementResponse | undefined): number | false {
  if (!row) return FAST_POLL_MS;
  if (isTerminalStatus(row.status)) return false;
  // Provider confirmed success — stop polling
  if (row.error_code && ["00", "0", "000"].includes(row.error_code.trim())) return false;
  const ageMs = Date.now() - new Date(row.created_at).getTime();
  if (ageMs > POLL_CUTOFF_MS) return false;
  return ageMs > FAST_PHASE_MS ? SLOW_POLL_MS : FAST_POLL_MS;
}

export function useManualDisbursementList(limit = 20) {
  return useQuery({
    queryKey: QueryKeys.manualDisbursement.list(limit),
    queryFn: async (): Promise<ManualDisbursementResponse[]> => {
      const res = await manualDisbursementService.list(limit);
      return res.data ?? [];
    },
    staleTime: 10_000,
    // Refetch the list periodically while the page is open so the user
    // sees status updates on rows initiated from another tab/session.
    refetchInterval: 30_000,
  });
}

export function useManualDisbursementStatus(txnId: string | null | undefined) {
  return useQuery({
    enabled: Boolean(txnId),
    queryKey: txnId
      ? QueryKeys.manualDisbursement.detail(txnId)
      : ["manual-disbursement", "detail", "noop"],
    queryFn: async (): Promise<ManualDisbursementResponse | null> => {
      if (!txnId) return null;
      const res = await manualDisbursementService.status(txnId);
      return res.data ?? null;
    },
    refetchInterval: (query) => pollIntervalForRow(query.state.data ?? undefined),
    // While a row is in-flight we want fresh data on every interval —
    // staleTime 0 keeps refetchInterval from being skipped.
    staleTime: 0,
  });
}

export function useInitiateManualDisbursement() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: InitiateManualDisbursementRequest) =>
      manualDisbursementService.initiate(body),
    onSuccess: (response) => {
      queryClient.invalidateQueries({
        queryKey: ["admin", "manual-disbursement", "list"],
      });
      showSuccessNotification(
        response.message ?? "Giao dịch đã được khởi tạo",
      );
    },
    onError: (error) => {
      showErrorNotification(error);
    },
  });
}

// useManualDisbursementBanks fetches the live partner-bank list from
// the backend (which proxies and caches 9pay's published list). The
// list rarely changes; aggressive client-side caching keeps page loads
// snappy. The page falls back to a small static list when the query
// errors so an admin is never blocked by a 9pay outage.
export function useManualDisbursementBanks() {
  return useQuery({
    queryKey: ["admin", "manual-disbursement", "banks"],
    queryFn: async (): Promise<BankInfoResponse[]> => {
      const res = await manualDisbursementService.banks();
      return res.data ?? [];
    },
    // Treat cached data as stale immediately so an empty list from a prior
    // session never blocks a real API response from showing up.
    staleTime: 0,
    gcTime: 5 * 60 * 1000,
    refetchOnMount: true,
    retry: 1,
  });
}

export function useVerifyManualDisbursementAccount() {
  return useMutation<
    VerifyAccountResponse | null,
    unknown,
    VerifyAccountRequest
  >({
    mutationFn: async (body) => {
      const res = await manualDisbursementService.verifyAccount(body);
      return res.data ?? null;
    },
    // Verify-account does not mutate server state; on error we surface
    // the message inline next to the form, NOT via a toast — the toast
    // would interrupt the user's flow as they retype the account number.
  });
}
