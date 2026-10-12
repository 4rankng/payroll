import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  mcpApiKeysService,
  type CreateMCPAPIKeyPayload,
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
