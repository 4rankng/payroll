import { useCallback, useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { chatService } from '@/services/api/chat.service';
import { QueryKeys } from '@/lib/queryKeys';
import { authManager } from '@/lib/auth';

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

// Module-level, keyed by user id: the FAB can unmount/remount the panel
// (role toggle, settings load, logout) and a ref alone would drop the thread,
// leaving the panel pointed at conversation 0 with no history.
const activeConversationByUser = new Map<number, number>();

/** Resolves the current conversation id, creating one on first send. Kept as
 * a hook so the panel owns zero creation effects — remounts can no longer
 * orphan an in-flight create.
 *
 * Three guards make the panel idempotent:
 *  - `cachedRef` survives re-renders. A plain `let cached` was reset to null on
 *    every render, so the second message created a SECOND conversation and the
 *    panel switched to it — history appeared to clear itself on every send.
 *  - `inFlightRef` dedupes concurrent calls. Without it, two quick clicks both
 *    saw an empty cache and both created a conversation, producing duplicate
 *    messages in two different threads.
 *  - `activeConversationByUser` outlives the component, so a remounted panel
 *    rejoins the same thread instead of starting over. */
export const useEnsureChatConversation = (
  setConversationId: (id: number) => void
) => {
  const cachedRef = useRef<number | null>(null);
  const inFlightRef = useRef<Promise<number | null> | null>(null);

  // Rehydrate from the store so a remounted panel polls the existing thread.
  const [restored] = useState<number>(() => {
    const userId = authManager.getUserId();
    const stored = userId ? activeConversationByUser.get(userId) ?? null : null;
    cachedRef.current = stored;
    return stored ?? 0;
  });
  useEffect(() => {
    if (restored > 0) setConversationId(restored);
  }, [restored, setConversationId]);

  return useCallback(async (): Promise<number | null> => {
    if (cachedRef.current && cachedRef.current > 0) return cachedRef.current;
    if (inFlightRef.current) return inFlightRef.current;

    inFlightRef.current = (async () => {
      try {
        const conv = await chatService.createConversation();
        cachedRef.current = conv.id;
        const userId = authManager.getUserId();
        if (userId) activeConversationByUser.set(userId, conv.id);
        setConversationId(conv.id);
        return conv.id;
      } catch {
        return null;
      } finally {
        inFlightRef.current = null;
      }
    })();

    return inFlightRef.current;
  }, [setConversationId]);
};
