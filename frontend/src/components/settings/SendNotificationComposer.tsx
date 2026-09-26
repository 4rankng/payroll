import { memo, useCallback, useMemo, useState } from 'react';
import { AlertCircle, Bell, Loader2, Send, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useSendNotification } from '@/hooks/api/useNotifications';
import { cn } from '@/lib/utils';
import { NotificationDialogHeader } from './send-notification-dialog/NotificationDialogHeader';

export const SendNotificationComposer = memo(function SendNotificationComposer() {
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

  const { mutate: sendNotification, isPending } = useSendNotification();

  const resetForm = useCallback(() => {
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
  }, []);

  const canSend = useMemo(
    () =>
      (recipientIds.length > 0 || toAllPartners || toAllAdmins || toAllEmployees) &&
      title.trim().length > 0 &&
      message.trim().length > 0,
    [recipientIds, toAllPartners, toAllAdmins, toAllEmployees, title, message],
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
    if (!title.trim()) {
      setTitleError('Tiêu đề không được để trống');
      hasError = true;
    }
    if (!message.trim()) {
      setMessageError('Nội dung không được để trống');
      hasError = true;
    }
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
      { onSuccess: resetForm },
    );
  }, [
    recipientIds,
    title,
    message,
    toAllPartners,
    toAllAdmins,
    toAllEmployees,
    sendNotification,
    resetForm,
  ]);

  const handleRecipientsChange = useCallback((values: string[]) => {
    setRecipientIds(values.map((id) => Number.parseInt(id, 10)).filter((id) => !Number.isNaN(id)));
  }, []);

  const handleTitleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setTitle(event.target.value);
    setTitleError('');
  }, []);

  const handleMessageChange = useCallback((event: React.ChangeEvent<HTMLTextAreaElement>) => {
    setMessage(event.target.value);
    setMessageError('');
  }, []);

  const handleQuickSelectChange = useCallback((values: string[]) => {
    setQuickSelect(values);
    setToAllAdmins(values.includes('admins'));
    setToAllPartners(values.includes('partners'));
    setToAllEmployees(values.includes('employees'));
    setRecipientError('');
  }, []);

  const recipientCount =
    recipientIds.length +
    (toAllAdmins ? 1 : 0) +
    (toAllPartners ? 1 : 0) +
    (toAllEmployees ? 1 : 0);

  return (
    <form
      className="overflow-hidden rounded-xl border bg-card"
      onSubmit={(event) => {
        event.preventDefault();
        handleSend();
      }}
    >
      <div className="flex flex-col gap-4 border-b px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-5">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-primary/10">
            <Bell className="h-4 w-4 text-primary" />
          </div>
          <div className="min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-sm font-semibold">Gửi thông báo</h2>
              {recipientCount > 0 && (
                <span className="inline-flex items-center gap-1 rounded-full border border-primary/20 bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                  <Users className="h-3 w-3" />
                  {recipientCount} nhóm/người nhận
                </span>
              )}
            </div>
            <p className="text-sm leading-relaxed text-muted-foreground">
              Soạn và gửi thông báo đến người dùng trong hệ thống.
            </p>
          </div>
        </div>

        <Button type="submit" className="min-h-11 shrink-0 sm:min-w-32" disabled={!canSend || isPending}>
          {isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Đang gửi...
            </>
          ) : (
            <>
              <Send className="mr-2 h-4 w-4" />
              Gửi thông báo
            </>
          )}
        </Button>
      </div>

      <div className="grid min-h-[380px] grid-cols-1 lg:grid-cols-[360px_minmax(0,1fr)]">
        <div className="border-b p-4 sm:p-5 lg:border-b-0 lg:border-r">
          <NotificationDialogHeader
            recipientIds={recipientIds}
            quickSelect={quickSelect}
            recipientError={recipientError}
            onRecipientsChange={handleRecipientsChange}
            onQuickSelectChange={handleQuickSelectChange}
            onClearRecipientError={() => setRecipientError('')}
            title={title}
            titleError={titleError}
            onTitleChange={handleTitleChange}
          />
        </div>

        <section className="flex min-h-[280px] flex-col" aria-labelledby="notification-message-label">
          <div className="border-b px-4 py-3 sm:px-5">
            <h3
              id="notification-message-label"
              className="font-display text-xs font-semibold uppercase tracking-widest text-muted-foreground"
            >
              Nội dung thông báo
            </h3>
          </div>
          <div className="flex flex-1 flex-col p-4 sm:p-5">
            <textarea
              value={message}
              onChange={handleMessageChange}
              placeholder="Nhập nội dung thông báo..."
              aria-labelledby="notification-message-label"
              className={cn(
                'min-h-56 flex-1 resize-none rounded-lg border bg-card p-3 text-sm leading-relaxed text-foreground',
                'placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                messageError && 'border-destructive focus-visible:ring-destructive',
              )}
            />
            {messageError && (
              <div className="mt-2 flex items-start gap-1.5 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-xs text-destructive">
                <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                <span>{messageError}</span>
              </div>
            )}
          </div>
        </section>
      </div>
    </form>
  );
});
