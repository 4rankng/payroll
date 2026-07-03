import { memo, ChangeEvent } from 'react';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { AlertCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { NotificationRecipientSection } from './NotificationRecipientSection';

interface NotificationDialogHeaderProps {
  recipientIds: number[];
  quickSelect: string[];
  recipientError: string;
  onRecipientsChange: (values: string[]) => void;
  onQuickSelectChange: (values: string[]) => void;
  onClearRecipientError: () => void;
  title: string;
  titleError: string;
  onTitleChange: (event: ChangeEvent<HTMLInputElement>) => void;
  className?: string;
}

export const NotificationDialogHeader = memo(function NotificationDialogHeader({
  recipientIds,
  quickSelect,
  recipientError,
  onRecipientsChange,
  onQuickSelectChange,
  onClearRecipientError,
  title,
  titleError,
  onTitleChange,
  className,
}: NotificationDialogHeaderProps) {
  return (
    <div className={cn('space-y-5', className)} role="region" aria-label="Thông tin thông báo">
      {/* Recipients Section */}
      <section aria-labelledby="recipients-label">
        <span id="recipients-label" className="font-display text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-3 block">
          Người nhận
        </span>
        <NotificationRecipientSection
          recipientIds={recipientIds}
          quickSelect={quickSelect}
          recipientError={recipientError}
          onRecipientsChange={onRecipientsChange}
          onQuickSelectChange={onQuickSelectChange}
          onClearRecipientError={onClearRecipientError}
        />
      </section>

      <div className="border-t border-border/40" />

      {/* Title Section */}
      <section aria-labelledby="title-label">
        <span id="title-label" className="font-display text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-3 block">
          Tiêu đề
        </span>
        <div className="space-y-1.5">
          <Label htmlFor="notif-title" className="sr-only">
            Tiêu đề thông báo
          </Label>
          <Input
            id="notif-title"
            type="text"
            placeholder="Nhập tiêu đề thông báo..."
            aria-label="Tiêu đề thông báo"
            value={title}
            onChange={onTitleChange}
            className={cn(
              'h-11',
              titleError && 'border-red-500 focus-visible:ring-red-500'
            )}
          />
          {titleError && (
            <div className="flex items-start gap-1.5 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-600">
              <AlertCircle className="h-3.5 w-3.5 flex-shrink-0 mt-0.5" />
              <span className="break-words">{titleError}</span>
            </div>
          )}
        </div>
      </section>
    </div>
  );
});
