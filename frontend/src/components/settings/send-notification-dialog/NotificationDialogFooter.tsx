import { memo } from 'react';
import { Button } from '@/components/ui/button';
import { DialogFooter } from '@/components/ui/dialog';
import { Loader2, Users } from 'lucide-react';

interface NotificationDialogFooterProps {
  canSend: boolean;
  isPending: boolean;
  onSend: () => void;
  onClose: () => void;
}

/**
 * Footer section for notification dialog
 * Contains cancel and send buttons with loading states
 */
export const NotificationDialogFooter = memo(function NotificationDialogFooter({
  canSend,
  isPending,
  onSend,
  onClose,
}: NotificationDialogFooterProps) {
  return (
    <DialogFooter className="flex flex-row">
      <Button
        type="button"
        variant="monochrome"
        onClick={onClose}
        disabled={isPending}
        className="min-w-[80px] h-11 min-h-[44px]"
      >
        Hủy
      </Button>

      <Button
        type="button"
        variant="warning"
        onClick={onSend}
        disabled={!canSend || isPending}
        className="min-w-[120px] h-11 min-h-[44px] transition-all duration-200 hover:shadow-sm hover:shadow-warning/30"
      >
        {isPending ? (
          <>
            <Loader2 className="w-4 h-4 mr-2 animate-spin" />
            Đang gửi...
          </>
        ) : (
          <>
            <Users className="w-4 h-4 mr-2" />
            Gửi
          </>
        )}
      </Button>
    </DialogFooter>
  );
});
