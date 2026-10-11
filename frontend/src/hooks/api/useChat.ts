import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { chatService } from '@/services/api/chat.service';
import { QueryKeys } from '@/lib/queryKeys';

export const useChatConversations = () =>
  useQuery({
    queryKey: QueryKeys.chat.conversations(),
    queryFn: () => chatService.listConversations(),
    staleTime: 15_000,
  });

/** Poll an open conversation so the assistant reply lands without a manual
 * refresh; polling stops when the panel closes (enabled=false). */
export const useChatConversation = (id: number, active: boolean) =>
  useQuery({
    queryKey: QueryKeys.chat.conversation(id),
    queryFn: () => chatService.getConversation(id),
    enabled: active && id > 0,
    refetchInterval: active ? 5_000 : false,
  });

export const useUploadChatAttachment = () =>
  useMutation({ mutationFn: (file: File) => chatService.uploadAttachment(file) });

export const useCreateChatConversation = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (title?: string) => chatService.createConversation(title),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.chat.all });
    },
  });
};

export const useSendChatMessage = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, content, attachmentIds }: { id: number; content: string; attachmentIds?: number[] }) =>
      chatService.sendMessage(id, content, attachmentIds ?? []),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: QueryKeys.chat.all });
    },
  });
};
