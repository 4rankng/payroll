import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  opsApiKeysService,
  type CreateOpsAPIKeyPayload,
  type SetOpsAPIKeyScopePayload,
} from "@/services/api/opsApiKeys.service";
import { getErrorMessage } from "@/utils/error-handler";

const LIST_KEY = ["opsApiKeys", "list"] as const;

/** useOpsApiKeys — the admin list of pk_ops_* keys (revoked included). */
export const useOpsApiKeys = () =>
  useQuery({
    queryKey: LIST_KEY,
    queryFn: () => opsApiKeysService.list(),
  });

export const useCreateOpsAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateOpsAPIKeyPayload) => opsApiKeysService.create(payload),
    meta: { skipGlobalError: true },
    onSuccess: () => {
      toast.success("Đã tạo khoá ops");
      qc.invalidateQueries({ queryKey: ["opsApiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};

export const useRevokeOpsAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => opsApiKeysService.revoke(id),
    meta: { skipGlobalError: true },
    onSuccess: () => {
      toast.success("Đã thu hồi khoá ops");
      qc.invalidateQueries({ queryKey: ["opsApiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};

/**
 * useSetOpsAPIKeyScope — the read_only ↔ read_write toggle. The backend
 * demands an OTP-verified session; a 403 here usually means the admin must
 * re-verify (email OTP) before flipping a key's scope.
 */
export const useSetOpsAPIKeyScope = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: SetOpsAPIKeyScopePayload }) =>
      opsApiKeysService.setScope(id, payload),
    meta: { skipGlobalError: true },
    onSuccess: (_data, { payload }) => {
      toast.success(
        payload.scope === "read_write"
          ? "Đã chuyển khoá sang read_write — tự hồi về read_only sau cửa sổ ghi"
          : "Đã chuyển khoá sang read_only",
      );
      qc.invalidateQueries({ queryKey: ["opsApiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};
