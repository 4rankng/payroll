import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  apiKeysService,
  type CreateAPIKeyPayload,
} from "@/services/api/apiKeys.service";
import { getErrorMessage } from "@/utils/error-handler";

const LIST_KEY = ["apiKeys", "list"] as const;

/** useApiKeys — the admin list of machine API keys (revoked included). */
export const useApiKeys = () =>
  useQuery({
    queryKey: LIST_KEY,
    queryFn: () => apiKeysService.list(),
  });

export const useCreateAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateAPIKeyPayload) => apiKeysService.create(payload),
    meta: { skipGlobalError: true },
    onSuccess: () => {
      toast.success("Đã tạo khoá API");
      qc.invalidateQueries({ queryKey: ["apiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};

export const useRevokeAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => apiKeysService.revoke(id),
    meta: { skipGlobalError: true },
    onSuccess: () => {
      toast.success("Đã thu hồi khoá API");
      qc.invalidateQueries({ queryKey: ["apiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};
