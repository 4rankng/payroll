import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  mcpApiKeysService,
  type CreateMCPAPIKeyPayload,
  type MCPAPIKey,
  type SetMCPAPIKeyScopePayload,
} from "@/services/api/mcpApiKeys.service";
import { getErrorMessage } from "@/utils/error-handler";

const LIST_KEY = ["mcpApiKeys", "list"] as const;

/** useMcpApiKeys — the admin list of MCP API keys (revoked included). */
export const useMcpApiKeys = () =>
  useQuery({
    queryKey: LIST_KEY,
    queryFn: () => mcpApiKeysService.list(),
  });

export const useCreateMCPAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateMCPAPIKeyPayload) => mcpApiKeysService.create(payload),
    meta: { skipGlobalError: true },
    onSuccess: () => {
      toast.success("Đã tạo khoá MCP");
      qc.invalidateQueries({ queryKey: ["mcpApiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};

export const useRevokeMCPAPIKey = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => mcpApiKeysService.revoke(id),
    meta: { skipGlobalError: true },
    onSuccess: () => {
      toast.success("Đã thu hồi khoá MCP");
      qc.invalidateQueries({ queryKey: ["mcpApiKeys"] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });
};

/**
 * useSetMCPAPIKeyScope — the read_only <-> full toggle on the same key. The
 * list flips optimistically (rollback on error), then refetches with the
 * server truth (scope_updated_at). The backend demands an OTP-verified
 * session; a 403 here usually means the admin must re-verify (email OTP)
 * before flipping a key's scope.
 */
export const useSetMCPAPIKeyScope = () => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: SetMCPAPIKeyScopePayload }) =>
      mcpApiKeysService.setScope(id, payload),
    meta: { skipGlobalError: true },
    onMutate: async ({ id, payload }) => {
      await qc.cancelQueries({ queryKey: LIST_KEY });
      const previous = qc.getQueryData<{ data: MCPAPIKey[] | undefined }>(LIST_KEY);
      const keys = previous?.data;
      if (keys) {
        qc.setQueryData(LIST_KEY, {
          ...previous,
          data: keys.map((k) =>
            k.id === id ? { ...k, scope: payload.scope, scope_updated_at: new Date().toISOString() } : k,
          ),
        });
      }
      return { previous };
    },
    onError: (error, _vars, context) => {
      if (context?.previous) qc.setQueryData(LIST_KEY, context.previous);
      toast.error(getErrorMessage(error));
    },
    onSuccess: (_data, { payload }) => {
      toast.success(
        payload.scope === "full"
          ? "Đã chuyển khoá MCP sang ALL — khoá được phép ghi dữ liệu"
          : "Đã chuyển khoá MCP về READ ONLY",
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: ["mcpApiKeys"] });
    },
  });
};
