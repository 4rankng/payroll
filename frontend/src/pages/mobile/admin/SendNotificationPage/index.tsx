import { useState, useCallback, useMemo } from 'react';
import { Bell, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useSendNotification } from '@/hooks/api/useNotifications';
import { NotificationDialogHeader } from '@/components/settings/send-notification-dialog/NotificationDialogHeader';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { cn } from '@/lib/utils';

export default function SendNotificationPageMobile() {
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

    sendNotification({
      recipient_ids: recipientIds,
      title: title.trim(),
      message: message.trim(),
      content_type: 'plain_text',
      to_all_partners: toAllPartners,
      to_all_admins: toAllAdmins,
      to_all_employees: toAllEmployees,
    });
  }, [recipientIds, title, message, toAllPartners, toAllAdmins, toAllEmployees, sendNotification]);

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

  return (
    <div className="pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <MobilePageHeader
        title="Gửi thông báo"
        icon={Bell}
        subtitle="Tạo và gửi thông báo đến người dùng"
      />

      <div className="p-4 space-y-6">
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

        <section aria-labelledby="message-label">
          <span id="message-label" className="font-display text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-3 block">
            Nội dung
          </span>
          <div className="space-y-1.5">
            <Label htmlFor="notif-message" className="sr-only">Nội dung thông báo</Label>
            <Textarea
              id="notif-message"
              placeholder="Nhập nội dung thông báo..."
              rows={5}
              value={message}
              onChange={handleMessageChange}
              className={cn('min-h-[132px]', messageError && 'border-red-500 focus-visible:ring-red-500')}
            />
            {messageError && (
              <div className="flex items-start gap-1.5 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-600">
                <AlertCircle className="h-3.5 w-3.5 flex-shrink-0 mt-0.5" />
                <span className="break-words">{messageError}</span>
              </div>
            )}
          </div>
        </section>

        <Button
          type="button"
          className="w-full h-12 font-semibold text-base"
          onClick={handleSend}
          disabled={!canSend || isPending}
        >
          {isPending ? (
            <><Loader2 className="mr-2 h-4 w-4 animate-spin" />Đang gửi...</>
          ) : (
            <><Bell className="w-4 h-4 mr-2" />Gửi thông báo</>
          )}
        </Button>
      </div>
    </div>
  );
}
