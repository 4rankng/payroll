import { useEffect, useRef, useState } from 'react';
import { Loader2, SendHorizontal, Sparkles } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { cn } from '@/lib/utils';
import {
  useChatConversation,
  useCreateChatConversation,
  useSendChatMessage,
} from '@/hooks/api/useChat';
import { useIsMobile } from '@/hooks/useBreakpoint';

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
  const [conversationId, setConversationId] = useState<number>(0);
  const [input, setInput] = useState('');
  const listRef = useRef<HTMLDivElement>(null);

  const createConversation = useCreateChatConversation();
  const sendMessage = useSendChatMessage();
  const detail = useChatConversation(conversationId, open && conversationId > 0);

  useEffect(() => {
    if (open && conversationId === 0 && !createConversation.isPending) {
      createConversation.mutate(undefined, {
        onSuccess: (conv) => setConversationId(conv.id),
      });
    }
  }, [open, conversationId, createConversation]);

  const messages = detail.data?.messages ?? [];
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight });
  }, [messages.length, sendMessage.isPending]);

  const handleSend = (text?: string) => {
    const content = (text ?? input).trim();
    if (!content || conversationId === 0 || sendMessage.isPending) return;
    setInput('');
    sendMessage.mutate({ id: conversationId, content });
  };

  const toolCalls = detail.data?.tool_calls ?? [];

  return (
    <Sheet open={open} onOpenChange={(next) => !next && onClose()}>
      <SheetContent
        side={isMobile ? 'bottom' : 'right'}
        className={cn(
          'flex flex-col gap-0 p-0',
          isMobile ? 'h-[85dvh] rounded-t-2xl' : 'w-[26rem] sm:max-w-[26rem]'
        )}
      >
        <SheetHeader className="border-b px-4 py-3">
          <SheetTitle className="flex items-center gap-2 text-sm font-semibold">
            <Sparkles aria-hidden="true" className="size-4 text-indigo-600" />
            Trợ lý AI
          </SheetTitle>
        </SheetHeader>

        <div ref={listRef} className="flex-1 space-y-3 overflow-y-auto px-4 py-4" role="log" aria-label="Hội thoại với trợ lý AI">
          {messages.length === 0 && !sendMessage.isPending ? (
            <div className="space-y-3 pt-6 text-center">
              <p className="text-sm text-muted-foreground">
                Xin chào! Tôi có thể tra cứu thông tin, giải thích lỗi import bảng công và gửi yêu cầu phê duyệt giúp bạn.
              </p>
              <div className="space-y-2 pt-2">
                {QUICK_ACTIONS.map((action) => (
                  <button
                    key={action}
                    type="button"
                    onClick={() => handleSend(action)}
                    className="block w-full rounded-full border bg-background px-4 py-2.5 text-left text-sm text-foreground transition-colors hover:bg-muted/50"
                  >
                    {action}
                  </button>
                ))}
              </div>
            </div>
          ) : null}

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
              {message.content}
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

        {toolCalls.length > 0 ? (
          <p className="border-t px-4 py-1.5 text-xs text-muted-foreground">
            Trợ lý đã dùng {toolCalls.length} thao tác dữ liệu để trả lời.
          </p>
        ) : null}

        <form
          className="flex items-end gap-2 border-t px-3 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]"
          onSubmit={(event) => {
            event.preventDefault();
            handleSend();
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
            disabled={sendMessage.isPending || conversationId === 0}
            className="max-h-32 min-h-11 flex-1 resize-none rounded-xl border border-input bg-background px-3 py-2.5 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <Button
            type="submit"
            size="icon"
            className="size-11 shrink-0"
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
