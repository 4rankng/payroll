import { useState, memo, useCallback, useMemo, useEffect } from 'react';
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { Bell, AlertCircle, X, ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useSendNotification } from '@/hooks/api/useNotifications';
import { NotificationDialogHeader } from './send-notification-dialog/NotificationDialogHeader';
import { Loader2, Users } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useIsMobile } from '@/hooks/useBreakpoint';

interface SendNotificationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const SendNotificationDialog = memo(function SendNotificationDialog({
  open,
  onOpenChange,
}: SendNotificationDialogProps) {
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
  const isMobile = useIsMobile();

  useEffect(() => {
    if (open) {
      setRecipientIds([]);
      setTitle('');
      setMessage('');
      setToAllPartners(false);
      setToAllAdmins(false);
      setToAllEmployees(false);
      setQuickSelect([]);
      setRecipientError('');
      setTitleError('');
      setMessageError('');
      setShowFields(true);
    }
  }, [open]);

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
      { onSuccess: () => onOpenChange(false) }
    );
  }, [recipientIds, title, message, toAllPartners, toAllAdmins, toAllEmployees, sendNotification, onOpenChange]);

  const handleDialogClose = useCallback(() => onOpenChange(false), [onOpenChange]);

  const handleRecipientsChange = useCallback((values: string[]) => {
    setRecipientIds(values.map((id) => parseInt(id, 10)).filter((id) => !isNaN(id)));
  }, []);

  const handleTitleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setTitle(event.target.value);
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
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && canSend && !isPending && open) {
        e.preventDefault();
        handleSend();
      }
    };
    if (open) {
      window.addEventListener('keydown', handleKeyDown);
      return () => window.removeEventListener('keydown', handleKeyDown);
    }
  }, [open, canSend, isPending, handleSend]);

  const recipientCount = recipientIds.length + (toAllAdmins ? 1 : 0) + (toAllPartners ? 1 : 0) + (toAllEmployees ? 1 : 0);

  const notifHeaderProps = {
    recipientIds, quickSelect, recipientError,
    onRecipientsChange: handleRecipientsChange,
    onQuickSelectChange: handleQuickSelectChange,
    onClearRecipientError: () => setRecipientError(''),
    title, titleError, onTitleChange: handleTitleChange,
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="w-screen h-screen max-w-none max-h-none shadow-none p-0 flex flex-col overflow-hidden bg-muted/30"
        title="GỬI THÔNG BÁO"
        description="Gửi thông báo đến người dùng"
        hideCloseButton
      >
        <DialogTitle className="sr-only">Gửi thông báo</DialogTitle>
        <DialogDescription className="sr-only">Soạn và gửi thông báo đến người dùng trong hệ thống</DialogDescription>

        {/* Top Bar */}
        <div className="flex-shrink-0 flex items-center justify-between px-3 py-2.5 md:px-5 md:py-3 bg-background border-b border-border/60">
          <div className="flex items-center gap-2 md:gap-3 min-w-0">
            <div className="flex items-center justify-center w-8 h-8 rounded-xl bg-orange-100 flex-shrink-0">
              <Bell className="h-4 w-4 text-orange-600" />
            </div>
            <div className="min-w-0">
              <p className="text-sm font-semibold leading-none">Gửi thông báo</p>
              <p className="text-xs text-muted-foreground mt-0.5 hidden md:block">
                Soạn và gửi thông báo đến người dùng trong hệ thống
              </p>
            </div>
            {recipientCount > 0 && (
              <span className="inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full bg-primary/10 text-primary border border-primary/20 flex-shrink-0">
                <Users className="h-3 w-3" />
                {recipientCount}
              </span>
            )}
          </div>
          <div className="flex items-center gap-1 md:gap-2 flex-shrink-0">
            <Button
              type="button" variant="ghost" size="sm" onClick={handleDialogClose} disabled={isPending}
              className="h-9 w-9 md:w-auto md:px-3 text-muted-foreground hover:text-foreground"
              aria-label="Hủy"
            >
              <X className="h-4 w-4 md:mr-1.5" />
              <span className="hidden md:inline">Hủy</span>
            </Button>
            <Button
              type="button" variant="warning" size="sm" onClick={handleSend}
              disabled={!canSend || isPending}
              className="h-9 px-3 md:px-4 md:min-w-[100px]"
              title="Cmd/Ctrl + Enter"
            >
              {isPending ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin md:mr-1.5" />
                  <span className="hidden md:inline">Đang gửi...</span>
                </>
              ) : (
                <>
                  <Users className="w-3.5 h-3.5 md:mr-1.5" />
                  <span className="hidden md:inline">Gửi</span>
                </>
              )}
            </Button>
          </div>
        </div>

        {/* Mobile: stacked layout */}
        {isMobile ? (
          <div className="flex-1 overflow-y-auto flex flex-col min-h-0">
            {/* Collapsible fields */}
            <div className="bg-background border-b border-border/60 flex-shrink-0">
              <button
                type="button"
                onClick={() => setShowFields((v) => !v)}
                className="w-full flex items-center justify-between px-4 py-3 text-left min-h-[44px]"
                aria-expanded={showFields}
              >
                <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  Thông tin thông báo
                  {recipientCount > 0 && (
                    <span className="ml-2 text-orange-600 normal-case font-medium">
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

            {/* Editor */}
            <div className="flex flex-col flex-1 min-h-0">
              <div className="flex items-center justify-between px-4 py-2.5 border-b border-border/40 bg-background/60 flex-shrink-0">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Nội dung thông báo</p>
              </div>
              <div className="flex-1 flex flex-col min-h-[280px] p-3">
                <textarea
                  value={message}
                  onChange={handleMessageChange}
                  placeholder="Nhập nội dung thông báo..."
                  className="flex-1 w-full resize-none bg-transparent text-sm leading-relaxed text-foreground placeholder:text-muted-foreground/40 focus:outline-none p-2"
                />
                {messageError && (
                  <div className="flex items-center gap-1.5 text-xs text-red-600 mt-2 bg-red-50 border border-red-200 rounded-xl px-3 py-2">
                    <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
                    <span>{messageError}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Mobile bottom action bar */}
            <div className="flex-shrink-0 px-4 py-3 bg-background border-t border-border/60 flex items-center gap-3">
              <Button type="button" variant="outline" className="flex-1 h-11" onClick={handleDialogClose} disabled={isPending}>
                <X className="h-4 w-4 mr-2" />Hủy
              </Button>
              <Button type="button" variant="warning" className="flex-1 h-11" onClick={handleSend} disabled={!canSend || isPending}>
                {isPending
                  ? <><Loader2 className="w-4 h-4 mr-2 animate-spin" />Đang gửi...</>
                  : <><Users className="w-4 h-4 mr-2" />Gửi thông báo</>
                }
              </Button>
            </div>
          </div>
        ) : (
          /* Desktop: side-by-side layout */
          <div className="flex-1 overflow-hidden flex min-h-0">
            <div className="w-[340px] flex-shrink-0 flex flex-col border-r border-border/60 bg-background overflow-y-auto">
              <div className="p-5 space-y-6">
                <NotificationDialogHeader {...notifHeaderProps} />
              </div>
            </div>
            <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
              <div className="flex-shrink-0 flex items-center justify-between px-5 py-3 border-b border-border/40 bg-background/60">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Nội dung thông báo</p>
              </div>
              <div className="flex-1 flex flex-col min-h-0 p-4">
                <textarea
                  value={message}
                  onChange={handleMessageChange}
                  placeholder="Nhập nội dung thông báo..."
                  className="flex-1 w-full resize-none bg-transparent text-sm leading-relaxed text-foreground placeholder:text-muted-foreground/40 focus:outline-none p-2"
                />
                {messageError && (
                  <div className={cn('flex items-center gap-1.5 text-xs text-red-600 mt-2', 'bg-red-50 border border-red-200 rounded-xl px-3 py-2')}>
                    <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
                    <span>{messageError}</span>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
});
