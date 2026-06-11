import { useState, useCallback, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Bell, X, Loader2, AlertCircle, ChevronDown, ChevronUp, Send, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useSendNotification } from '@/hooks/api/useNotifications';
import { NotificationDialogHeader } from '@/components/settings/send-notification-dialog/NotificationDialogHeader';
import { cn } from '@/lib/utils';
import { useIsMobile } from '@/hooks/useBreakpoint';

const SendNotificationPage = () => {
  const navigate = useNavigate();
  const isMobile = useIsMobile();

  const [recipientIds, setRecipientIds] = useState<number[]>([]);
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  const [toAllPartners, setToAllPartners] = useState(false);
  const [toAllAdmins, setToAllAdmins] = useState(false);
  const [toAllEmployees, setToAllEmployees] = useState(false);
  const [quickSelect, setQuickSelect] = useState<string[]>([]);
  const [recipientError, setRecipientError] = useState('');
  const [titleError, setTitleError] = useState('');
  const [messageError, setMessageError] = useState('');
  const [showFields, setShowFields] = useState(true);

  const { mutate: sendNotification, isPending } = useSendNotification();

  const handleBack = useCallback(() => navigate(-1), [navigate]);

  const canSend = useMemo(
    () => (recipientIds.length > 0 || toAllPartners || toAllAdmins || toAllEmployees) && title.trim() && message.trim(),
    [recipientIds, toAllPartners, toAllAdmins, toAllEmployees, title, message]
  );

  const handleSend = useCallback(() => {
    setRecipientError('');
    setTitleError('');
    setMessageError('');
    let hasError = false;
    if (recipientIds.length === 0 && !toAllPartners && !toAllAdmins && !toAllEmployees) {
      setRecipientError('Vui lòng chọn ít nhất một người nhận hoặc chọn gửi đến tất cả');
      hasError = true;
    }
    if (!title.trim()) { setTitleError('Tiêu đề không được để trống'); hasError = true; }
    if (!message.trim()) { setMessageError('Nội dung không được để trống'); hasError = true; }
    if (hasError) return;

    sendNotification(
      {
        recipient_ids: recipientIds,
        title: title.trim(),
        message: message.trim(),
        content_type: 'plain_text',
        to_all_partners: toAllPartners,
        to_all_admins: toAllAdmins,
        to_all_employees: toAllEmployees,
      },
      { onSuccess: handleBack }
    );
  }, [recipientIds, title, message, toAllPartners, toAllAdmins, toAllEmployees, sendNotification, handleBack]);

  const handleRecipientsChange = useCallback((values: string[]) => {
    setRecipientIds(values.map((id) => parseInt(id, 10)).filter((id) => !isNaN(id)));
  }, []);

  const handleTitleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setTitle(e.target.value);
    setTitleError('');
  }, []);

  const handleMessageChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setMessage(e.target.value);
    setMessageError('');
  }, []);

  const handleQuickSelectChange = useCallback((values: string[]) => {
    setQuickSelect(values);
    setToAllAdmins(values.includes('admins'));
    setToAllPartners(values.includes('partners'));
    setToAllEmployees(values.includes('employees'));
    setRecipientError('');
  }, []);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && canSend && !isPending) {
        e.preventDefault();
        handleSend();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [canSend, isPending, handleSend]);

  const recipientCount = recipientIds.length + (toAllAdmins ? 1 : 0) + (toAllPartners ? 1 : 0) + (toAllEmployees ? 1 : 0);

  const notifHeaderProps = {
    recipientIds, quickSelect, recipientError,
    onRecipientsChange: handleRecipientsChange,
    onQuickSelectChange: handleQuickSelectChange,
    onClearRecipientError: () => setRecipientError(''),
    title, titleError, onTitleChange: handleTitleChange,
  };

  return (
    <div className="send-notification-page flex flex-col h-full overflow-hidden bg-gradient-to-b from-background to-muted/20">
      <style>{`
        @keyframes bell-seal-pulse {
          0%, 100% { box-shadow: 0 0 24px hsl(45 95% 55% / 0.3); }
          50% { box-shadow: 0 0 36px hsl(45 95% 55% / 0.5); }
        }
        .bell-seal-ready {
          animation: bell-seal-pulse 3s ease-in-out infinite;
        }

        .send-notification-page input[aria-label="Tiêu đề thông báo"] {
          border: 1px solid hsl(var(--border) / 0.6) !important;
          border-bottom: 2px solid hsl(var(--border) / 0.4) !important;
          border-radius: 0.5rem !important;
          padding: 0.5rem 0.75rem !important;
          background: transparent !important;
          font-family: var(--font-display), Manrope, sans-serif !important;
          font-size: 1rem !important;
          font-weight: 600 !important;
          letter-spacing: -0.01em !important;
          height: 2.75rem !important;
          transition: border-color 0.3s ease !important;
        }
        .send-notification-page input[aria-label="Tiêu đề thông báo"]:focus {
          border-bottom-color: hsl(var(--primary) / 0.6) !important;
          box-shadow: none !important;
        }
        .send-notification-page input[aria-label="Tiêu đề thông báo"]::placeholder {
          color: hsl(var(--muted-foreground) / 0.4);
        }
      `}</style>

      {/* Top Bar — Navy-etched header */}
      <div className="flex-shrink-0 flex items-center justify-between px-4 py-3 md:px-6 md:py-4 bg-gradient-to-b from-primary/[0.04] to-transparent border-b-2 border-primary/10 animate-fade-in-up">
        <div className="flex items-center gap-3 min-w-0">
          <Bell className="h-5 w-5 text-primary flex-shrink-0" />
          <div className="min-w-0">
            <h1 className="text-xl font-extrabold tracking-tight font-display text-foreground leading-none">
              Gửi Thông Báo
            </h1>
            <p className="text-xs text-muted-foreground mt-0.5 hidden md:block">
              Soạn và gửi thông báo đến người dùng trong hệ thống
            </p>
          </div>
          {recipientCount > 0 && (
            <span className="inline-flex items-center gap-1 text-xs font-semibold px-2.5 py-0.5 rounded-full bg-success/10 text-success border border-success/20 flex-shrink-0">
              <Users className="h-3 w-3" />
              {recipientCount}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1.5 md:gap-2 flex-shrink-0">
          <Button
            type="button" variant="ghost" size="sm" onClick={handleBack} disabled={isPending}
            className="h-9 w-9 md:w-auto md:px-3 text-muted-foreground hover:text-foreground hidden md:inline-flex"
            aria-label="Hủy"
          >
            <X className="h-4 w-4 md:mr-1.5" />
            <span className="hidden md:inline text-xs">Hủy</span>
          </Button>
          <Button
            type="button" size="sm" onClick={handleSend}
            disabled={!canSend || isPending}
            className="h-9 px-4 md:min-w-[100px] font-semibold"
            title="Gửi thông báo (Cmd/Ctrl + Enter)"
          >
            {isPending ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin md:mr-1.5" />
                <span className="hidden md:inline">Đang gửi...</span>
              </>
            ) : (
              <>
                <Send className="w-3.5 h-3.5 md:mr-1.5" />
                <span className="hidden md:inline">Gửi</span>
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Mobile layout */}
      {isMobile ? (
        <div className="flex-1 overflow-y-auto flex flex-col min-h-0">
          <div className="bg-background/80 backdrop-blur-sm border-b border-border/30 flex-shrink-0 animate-fade-in-up delay-100">
            <button
              type="button"
              onClick={() => setShowFields((v) => !v)}
              className="w-full flex items-center justify-between px-4 py-3 text-left min-h-[44px]"
              aria-expanded={showFields}
            >
              <span className="font-display text-xs font-semibold uppercase tracking-widest text-muted-foreground">
                Thông tin thông báo
                {recipientCount > 0 && (
                  <span className="ml-2 text-success normal-case font-semibold">
                    · {recipientCount} người nhận
                    {title ? ` · ${title.slice(0, 18)}${title.length > 18 ? '…' : ''}` : ''}
                  </span>
                )}
              </span>
              {showFields
                ? <ChevronUp className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                : <ChevronDown className="h-4 w-4 text-muted-foreground flex-shrink-0" />
              }
            </button>
            {showFields && (
              <div className="px-4 pb-4">
                <NotificationDialogHeader {...notifHeaderProps} />
              </div>
            )}
          </div>
          <div className="flex-1 flex flex-col min-h-[280px] p-3 animate-fade-in-up delay-200">
            <textarea
              value={message}
              onChange={handleMessageChange}
              placeholder="Nhập nội dung thông báo..."
              className="flex-1 w-full resize-none rounded-lg border border-border/60 bg-background text-sm leading-relaxed text-foreground placeholder:text-muted-foreground/40 focus:outline-none focus:ring-1 focus:ring-primary/30 p-3"
            />
            {messageError && (
              <div className="flex items-center gap-1.5 text-xs text-red-600 mt-2 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
                <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
                <span>{messageError}</span>
              </div>
            )}
          </div>
          <div className="flex-shrink-0 px-4 py-3 bg-background/90 backdrop-blur-md border-t border-border/40 flex items-center gap-3">
            <Button type="button" variant="outline" className="flex-1 h-11" onClick={handleBack} disabled={isPending}>
              <X className="h-4 w-4 mr-2" />Hủy
            </Button>
            <Button type="button" className="flex-1 h-11 font-semibold" onClick={handleSend} disabled={!canSend || isPending}>
              {isPending
                ? <><Loader2 className="w-4 h-4 mr-2 animate-spin" />Đang gửi...</>
                : <><Send className="w-4 h-4 mr-2" />Gửi Thông Báo</>
              }
            </Button>
          </div>
        </div>
      ) : (
        /* Desktop layout — single-column immersive */
        <div className="flex-1 overflow-hidden flex flex-col min-h-0">
          {/* Fields strip */}
          <div className="flex-shrink-0 bg-background/80 backdrop-blur-sm border-b border-border/30 overflow-y-auto animate-fade-in-up delay-100">
            <div className="px-6 py-4 max-w-[960px] mx-auto w-full">
              <NotificationDialogHeader {...notifHeaderProps} className="space-y-3" />
            </div>
          </div>

          {/* Editor canvas */}
          <div className="flex-1 flex flex-col min-h-0 relative animate-fade-in-up delay-200">
            <div className="flex-1 flex flex-col min-h-0 px-8 py-6 md:px-12 md:py-8 max-w-[960px] mx-auto w-full">
              <textarea
                value={message}
                onChange={handleMessageChange}
                placeholder="Nhập nội dung thông báo..."
                className="flex-1 w-full resize-none rounded-lg border border-border/60 bg-background text-sm leading-relaxed text-foreground placeholder:text-muted-foreground/40 focus:outline-none focus:ring-1 focus:ring-primary/30 p-3"
              />
              {messageError && (
                <div className="flex items-center gap-1.5 text-xs text-red-600 mt-2 bg-red-50 border border-red-200 rounded-lg px-3 py-2">
                  <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
                  <span>{messageError}</span>
                </div>
              )}
            </div>

            {/* Bell seal — ready indicator */}
            <div
              className={cn(
                'absolute bottom-6 right-8 w-10 h-10 rounded-full flex items-center justify-center transition-all duration-500',
                canSend
                  ? 'bg-gradient-navy bell-seal-ready'
                  : 'border-2 border-border/40'
              )}
              title={canSend ? 'Sẵn sàng gửi' : 'Nhập đủ thông tin để gửi'}
            >
              <Bell className={cn(
                'h-4 w-4 transition-colors duration-500',
                canSend ? 'text-white' : 'text-muted-foreground/30'
              )} />
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SendNotificationPage;
