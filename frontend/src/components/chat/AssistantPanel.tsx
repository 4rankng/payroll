import { useEffect, useRef, useState } from 'react';
import { Loader2, Paperclip, SendHorizontal, X } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Sheet, SheetContent } from '@/components/ui/sheet';
import { cn } from '@/lib/utils';
import { useChatConversation, useEnsureChatConversation, useSendChatMessage, useUploadChatAttachment } from '@/hooks/api/useChat';
import type { ChatMessage } from '@/services/api/chat.service';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

// Internal paths the assistant may deep-link to. Anything outside this list
// (a model-invented path like /admin/sao-ke) stays plain text.
const LINKABLE = /^\/(admin|partner|employee)\/(timesheet|approvals|payment-history|advance-payments|wallet|settings|dashboard|employees|projects|users)(\/[a-z0-9-]+)*$/;

// The model abbreviates routes the way people speak ("/admin/advance",
// "/admin/timesheets"). Without a mapping those fail LINKABLE and render as
// dead plain text, so the reader has nowhere to click. Only map names that
// resolve unambiguously to a real route.
const PATH_ALIASES: Record<string, string> = {
  'advance': 'advance-payments',
  'timesheets': 'timesheet',
};

const resolveAlias = (path: string): string => {
  const segments = path.split('/');
  const last = segments[segments.length - 1];
  const alias = PATH_ALIASES[last];
  if (!alias) return path;
  return [...segments.slice(0, -1), alias].join('/');
};

const splitPaths = (text: string): Array<{ path?: string; text: string }> => {
  const out: Array<{ path?: string; text: string }> = [];
  const re = /\/(admin|partner|employee)\/[a-z0-9/-]+/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text))) {
    // Alias first, then whitelist: "/admin/advance" resolves to the real
    // "/admin/advance-payments" and only then passes the safety check.
    const resolved = resolveAlias(m[0]);
    if (LINKABLE.test(resolved)) {
      if (m.index > last) out.push({ text: text.slice(last, m.index) });
      // Link text keeps what the model wrote; the href is the real route.
      out.push({ text: m[0], path: resolved });
      last = m.index + m[0].length;
    }
  }
  if (last < text.length) out.push({ text: text.slice(last) });
  return out;
};

const QUICK_ACTIONS = [
  'Sao kê tháng này thế nào?',
  'Import bảng công bị lỗi, giúp tôi xem',
  'Trạng thái yêu cầu phê duyệt của tôi',
];

interface AssistantPanelProps {
  open: boolean;
  onClose: () => void;
}

/** MiMo assistant panel: one conversation, quick-action chips, auto-resize
 * composer. The reply polls in while the panel is open. */
export const AssistantPanel = ({ open, onClose }: AssistantPanelProps) => {
  const isMobile = useIsMobile();
  const navigate = useNavigate();
  const [conversationId, setConversationId] = useState<number>(0);
  const [input, setInput] = useState('');
  const [pendingImage, setPendingImage] = useState<{ id: number; name: string } | null>(null);
  const [optimistic, setOptimistic] = useState<ChatMessage[]>([]);
  const listRef = useRef<HTMLDivElement>(null);
  // Set synchronously on send so a burst of clicks cannot slip past the
  // mutation's own pending state.
  const sendingRef = useRef(false);
  const optimisticCounter = useRef(0);

  const chatEnsure = useEnsureChatConversation(setConversationId);
  const sendMessage = useSendChatMessage();
  const uploadAttachment = useUploadChatAttachment();
  const detail = useChatConversation(conversationId, open && conversationId > 0);

  // Auto-open ONE conversation per panel session. The ref guard makes this
  // idempotent across re-renders and remounts — an unstable mutation object
  // in the dependency array previously re-fired creation every render and
  // flooded the backend with thousands of empty conversations.

  const messages = detail.data?.messages ?? [];
  // The polled detail may land the user message before the turn settles —
  // hide the optimistic copy then, or the thread shows it twice.
  const fetchedContent = new Set(messages.filter((m) => m.role === 'user').map((m) => m.content));
  const visibleOptimistic = optimistic.filter((m) => !fetchedContent.has(m.content));
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight });
  }, [messages.length, optimistic.length, sendMessage.isPending]);

  const handleSend = async (text?: string) => {
    const content = (text ?? input).trim();
    if (!content) return;
    // Synchronous lock, set BEFORE the first await. `sendMessage.isPending`
    // only flips once mutate() runs, which is after chatEnsure() resolves —
    // guarding on it let a second click slip through the gap and duplicate
    // the message.
    if (sendingRef.current) return;
    sendingRef.current = true;
    try {
      // Conversation is created AT SEND TIME: one code path, immune to remounts
      // (an open-time create lost its state across FAB remounts and silently
      // swallowed sends).
      const convId = await chatEnsure();
      if (!convId) {
        toast.error('Không tạo được hội thoại — thử lại.');
        return;
      }
      setInput('');
      const attachmentIds = pendingImage ? [pendingImage.id] : undefined;
      setPendingImage(null);
      // Optimistic user bubble: the click is ALWAYS visibly acknowledged,
      // even before the round trip lands. Keyed by a monotonic counter —
      // -Date.now() collides when two sends land in the same millisecond,
      // which React then drops as a duplicate key.
      const optimisticId = -++optimisticCounter.current;
      setOptimistic((prev) => [
        ...prev,
        { id: optimisticId, role: 'user', content, created_at: new Date().toISOString() },
      ]);
      sendMessage.mutate(
        { id: convId, content, attachmentIds },
        { onSettled: () => setOptimistic((prev) => prev.filter((m) => m.id !== optimisticId)) }
      );
    } finally {
      sendingRef.current = false;
    }
  };

  const handlePickImage = (file: File | undefined) => {
    if (!file) return;
    // HEIC camera-roll screenshots and other formats are rejected server-side
    // by content sniffing — reject them here with clear copy instead.
    if (file.type !== 'image/png' && file.type !== 'image/jpeg') {
      toast.error('Chỉ nhận ảnh PNG hoặc JPG — ảnh HEIC vui lòng chuyển đổi trước khi gửi.');
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      toast.error('Ảnh tối đa 5MB.');
      return;
    }
    uploadAttachment.mutate(file, {
      onSuccess: (id) => setPendingImage({ id, name: file.name }),
      onError: () => toast.error('Không thể tải ảnh lên — thử lại.'),
    });
  };


  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent
        side={isMobile ? 'bottom' : 'right'}
        aria-label="Trợ lý AI"
        className={cn(
          'flex flex-col gap-0 border-0 p-0',
          isMobile ? 'h-[85dvh] rounded-t-2xl' : 'w-[26rem] sm:max-w-[26rem]'
        )}
      >

        <div ref={listRef} className="flex-1 space-y-3 overflow-y-auto px-4 py-4" role="log" aria-label="Hội thoại với trợ lý AI">
          {messages.length === 0 && optimistic.length === 0 && !sendMessage.isPending ? (
            <div className="space-y-3 pt-6 text-center">
              <p className="text-sm text-muted-foreground">
                Xin chào! Tôi có thể tra cứu thông tin, giải thích lỗi import bảng công và gửi yêu cầu phê duyệt giúp bạn.
              </p>
              <div className="space-y-2 pt-2">
                {QUICK_ACTIONS.map((action) => (
                  <button
                    key={action}
                    type="button"
                    onClick={() => void handleSend(action)}
                    disabled={sendMessage.isPending}
                    className="block w-full rounded-full border bg-background px-4 py-2.5 text-left text-sm text-foreground transition-colors hover:bg-muted/50 disabled:opacity-60"
                  >
                    {action}
                  </button>
                ))}
              </div>
            </div>
          ) : null}

          {visibleOptimistic.map((message) => (
            <div
              key={message.id}
              className="ml-auto max-w-[85%] rounded-2xl bg-primary/80 px-3.5 py-2.5 text-sm leading-5 whitespace-pre-line text-primary-foreground"
            >
              {message.content}
            </div>
          ))}

          {messages.map((message) => (
            <div
              key={message.id}
              className={cn(
                'max-w-[85%] rounded-2xl px-3.5 py-2.5 text-sm leading-5 whitespace-pre-line',
                message.role === 'user'
                  ? 'ml-auto bg-primary text-primary-foreground'
                  : 'mr-auto border bg-background text-foreground'
              )}
            >
              {message.role === 'user'
                ? message.content
                : splitPaths(message.content).map((part, i) =>
                    part.path ? (
                      <button
                        key={i}
                        type="button"
                        className="font-medium text-indigo-600 underline underline-offset-2 hover:text-indigo-800"
                        onClick={() => {
                          onClose();
                          navigate(part.path!);
                        }}
                      >
                        {part.text}
                      </button>
                    ) : (
                      <span key={i}>{part.text}</span>
                    )
                  )}
            </div>
          ))}

          {sendMessage.isPending ? (
            <div className="mr-auto flex items-center gap-2 rounded-2xl border bg-background px-3.5 py-2.5 text-sm text-muted-foreground">
              <Loader2 aria-hidden="true" className="size-4 animate-spin" />
              Đang xử lý…
            </div>
          ) : null}

          {sendMessage.isError ? (
            <p role="alert" className="mr-auto max-w-[85%] rounded-2xl border border-destructive/30 bg-destructive/5 px-3.5 py-2.5 text-sm text-destructive">
              Không gửi được tin nhắn — vui lòng thử lại.
            </p>
          ) : null}
        </div>

        {pendingImage ? (
          <div className="flex items-center gap-2 border-t px-3 py-2 text-xs text-muted-foreground">
            <span className="truncate">Đính kèm: {pendingImage.name}</span>
            <button
              type="button"
              className="ml-auto shrink-0 rounded p-1 hover:bg-muted"
              aria-label="Bỏ đính kèm"
              onClick={() => setPendingImage(null)}
            >
              <X aria-hidden="true" className="size-3.5" />
            </button>
          </div>
        ) : null}

        <form
          className="flex items-end gap-2 border-t px-3 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]"
          onSubmit={(event) => {
            event.preventDefault();
            void handleSend();
          }}
        >
          <textarea
            value={input}
            onChange={(event) => setInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                handleSend();
              }
            }}
            rows={1}
            placeholder="Nhập câu hỏi…"
            aria-label="Nội dung tin nhắn"
            disabled={sendMessage.isPending}
            className="max-h-32 h-9 flex-1 resize-none rounded-xl border border-input bg-background px-3 py-2.5 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <label className="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-input bg-background text-foreground transition-colors hover:bg-muted/50" aria-label="Đính kèm ảnh">
            {uploadAttachment.isPending ? (
              <Loader2 aria-hidden="true" className="size-4 animate-spin" />
            ) : (
              <Paperclip aria-hidden="true" className="size-4" />
            )}
            <input
              type="file"
              accept="image/png,image/jpeg"
              className="hidden"
              aria-label="Đính kèm ảnh chụp màn hình"
              disabled={uploadAttachment.isPending}
              onChange={(event) => {
                handlePickImage(event.target.files?.[0]);
                event.target.value = '';
              }}
            />
          </label>
          <Button
            type="submit"
            size="icon"
            className="h-9 w-9 shrink-0"
            aria-label="Gửi"
            disabled={sendMessage.isPending || input.trim().length === 0}
          >
            {sendMessage.isPending ? (
              <Loader2 aria-hidden="true" className="size-4 animate-spin" />
            ) : (
              <SendHorizontal aria-hidden="true" className="size-4" />
            )}
          </Button>
        </form>
      </SheetContent>
    </Sheet>
  );
};
