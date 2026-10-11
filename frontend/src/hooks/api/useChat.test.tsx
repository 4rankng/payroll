import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';

import { useEnsureChatConversation } from '@/hooks/api/useChat';
import { chatService } from '@/services/api/chat.service';
import { authManager } from '@/lib/auth';

vi.mock('@/services/api/chat.service', () => ({
  chatService: { createConversation: vi.fn() },
}));

// The conversation store is module-level and keyed by user id, so each test
// uses its own id to stay isolated from the previous one's cached thread.
// vi.hoisted keeps the binding available inside the hoisted vi.mock factory.
const { userIdRef } = vi.hoisted(() => ({ userIdRef: { current: 1 } }));
vi.mock('@/lib/auth', () => ({
  authManager: { getUserId: () => userIdRef.current },
}));

describe('useEnsureChatConversation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    userIdRef.current += 1;
  });

  it('reuses one conversation across re-renders instead of creating a new thread per message', async () => {
    let nextId = 100;
    vi.mocked(chatService.createConversation).mockImplementation(async () => ({
      id: nextId++,
      user_id: 7,
      role: 'admin',
      title: '',
      message_count: 0,
      created_at: '',
      updated_at: '',
    }));

    const setConversationId = vi.fn();
    const { result, rerender } = renderHook(() => useEnsureChatConversation(setConversationId));

    // First send creates the conversation.
    let first: number | null = null;
    await act(async () => {
      first = await result.current();
    });
    expect(first).toBe(100);

    // A re-render happens on every keystroke/state change. The cache MUST
    // survive it, otherwise the next message opens a brand-new thread and the
    // panel appears to clear its history.
    rerender();
    let second: number | null = null;
    await act(async () => {
      second = await result.current();
    });

    expect(second).toBe(100);
    expect(chatService.createConversation).toHaveBeenCalledTimes(1);
  });

  it('dedupes concurrent calls so a burst of clicks creates only one conversation', async () => {
    let resolveCreate: (value: unknown) => void = () => {};
    vi.mocked(chatService.createConversation).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        })
    );

    const setConversationId = vi.fn();
    const { result } = renderHook(() => useEnsureChatConversation(setConversationId));

    // Two rapid clicks (the quick-action pill double-fire) reach the hook
    // before the first create resolves.
    const pending: Array<Promise<number | null>> = [];
    await act(async () => {
      pending.push(result.current());
      pending.push(result.current());
      resolveCreate({ id: 555 });
      await Promise.all(pending);
    });

    expect(chatService.createConversation).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(pending[0]).resolves.toBe(555));
    expect(await pending[1]).toBe(555);
  });
});