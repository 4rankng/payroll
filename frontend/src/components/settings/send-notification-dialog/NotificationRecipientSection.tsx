import { memo, useCallback } from 'react';
import { UserMultiSelector } from '@/components/ui/user-multi-selector';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { AlertCircle, Shield, UserCog, User } from 'lucide-react';
import { cn } from '@/lib/utils';

interface NotificationRecipientSectionProps {
  recipientIds: number[];
  quickSelect: string[];
  recipientError: string;
  onRecipientsChange: (values: string[]) => void;
  onQuickSelectChange: (values: string[]) => void;
  onClearRecipientError: () => void;
}

export const NotificationRecipientSection = memo(function NotificationRecipientSection({
  recipientIds,
  quickSelect,
  recipientError,
  onRecipientsChange,
  onQuickSelectChange,
  onClearRecipientError,
}: NotificationRecipientSectionProps) {
  const handleRecipientsChange = useCallback(
    (values: string[]) => {
      onRecipientsChange(values);
      onClearRecipientError();
    },
    [onRecipientsChange, onClearRecipientError]
  );

  return (
    <div className="space-y-3">
      {/* Quick Select Toggles */}
      <ToggleGroup
        type="multiple"
        value={quickSelect}
        onValueChange={onQuickSelectChange}
        className="justify-start flex-wrap gap-2"
        aria-label="Chọn nhanh nhóm người nhận"
      >
        <ToggleGroupItem
          value="admins"
          className={cn(
            'h-9 text-xs gap-1.5 px-3 rounded-xl border transition-all duration-150',
            'data-[state=off]:bg-muted/40 data-[state=off]:border-border/50 data-[state=off]:text-muted-foreground',
            'data-[state=off]:hover:bg-purple-50 data-[state=off]:hover:border-purple-200 data-[state=off]:hover:text-purple-700',
            'data-[state=on]:bg-purple-100 data-[state=on]:border-purple-300 data-[state=on]:text-purple-700',
            'dark:data-[state=on]:bg-purple-950/50[state=on]:border-purple-700[state=on]:text-purple-300'
          )}
          aria-label="Gửi đến tất cả Admin"
        >
          <Shield className="h-3.5 w-3.5" />
          <span>Tất cả Admin</span>
        </ToggleGroupItem>

        <ToggleGroupItem
          value="partners"
          className={cn(
            'h-9 text-xs gap-1.5 px-3 rounded-xl border transition-all duration-150',
            'data-[state=off]:bg-muted/40 data-[state=off]:border-border/50 data-[state=off]:text-muted-foreground',
            'data-[state=off]:hover:bg-blue-50 data-[state=off]:hover:border-blue-200 data-[state=off]:hover:text-blue-700',
            'data-[state=on]:bg-blue-100 data-[state=on]:border-blue-300 data-[state=on]:text-blue-700',
            'dark:data-[state=on]:bg-blue-950/50[state=on]:border-blue-700[state=on]:text-blue-300'
          )}
          aria-label="Gửi đến tất cả Quản lý"
        >
          <UserCog className="h-3.5 w-3.5" />
          <span>Tất cả Quản lý</span>
        </ToggleGroupItem>

        <ToggleGroupItem
          value="employees"
          className={cn(
            'h-9 text-xs gap-1.5 px-3 rounded-xl border transition-all duration-150',
            'data-[state=off]:bg-muted/40 data-[state=off]:border-border/50 data-[state=off]:text-muted-foreground',
            'data-[state=off]:hover:bg-green-50 data-[state=off]:hover:border-green-200 data-[state=off]:hover:text-green-700',
            'data-[state=on]:bg-green-100 data-[state=on]:border-green-300 data-[state=on]:text-green-700',
            'dark:data-[state=on]:bg-green-950/50[state=on]:border-green-700[state=on]:text-green-300'
          )}
          aria-label="Gửi đến tất cả Nhân viên"
        >
          <User className="h-3.5 w-3.5" />
          <span>Tất cả Nhân viên</span>
        </ToggleGroupItem>
      </ToggleGroup>

      {/* Specific User Selector */}
      <div className="space-y-1.5">
        <p className="text-xs text-muted-foreground">Hoặc chọn người nhận cụ thể:</p>
        <UserMultiSelector
          value={recipientIds.map((id) => id.toString())}
          onValueChange={handleRecipientsChange}
          placeholder="Chọn người nhận..."
          className={cn(recipientError && 'border-red-500')}
        />
      </div>

      {recipientError && (
        <div className="flex items-center gap-1.5 text-xs text-red-600 bg-red-50 border border-red-200 rounded-xl px-3 py-2">
          <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
          <span>{recipientError}</span>
        </div>
      )}
    </div>
  );
});
