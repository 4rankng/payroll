import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { type GroupedTimesheet } from '../utils/timesheetGrouping';
import { formatDate } from '../utils/timesheetHelpers';

interface GroupBulkRejectDialogProps {
  isOpen: boolean;
  onClose: () => void;
  group: GroupedTimesheet | null;
  onConfirm: (group: GroupedTimesheet, reason: string) => Promise<void>;
  isLoading?: boolean;
}

export function GroupBulkRejectDialog({
  isOpen,
  onClose,
  group,
  onConfirm,
  isLoading = false
}: GroupBulkRejectDialogProps) {
  const [rejectionReason, setRejectionReason] = useState('');

  const handleConfirm = async () => {
    if (group && rejectionReason.trim()) {
      await onConfirm(group, rejectionReason);
      setRejectionReason('');
      onClose();
    }
  };

  const handleClose = () => {
    setRejectionReason('');
    onClose();
  };

  if (!group) return null;

  return (
    <ConfirmDialog
      open={isOpen}
      onOpenChange={handleClose}
      title="Loại tất cả bảng công trong nhóm"
      description={
        <div className="space-y-4">
          <div className="p-3 bg-muted rounded-xl">
            <p className="font-medium">{group.employeeName}</p>
            <p className="typography-body-medium text-muted-foreground">
              Ngày {formatDate(group.date)} • {group.statusBreakdown.pending_approval} mục chờ duyệt
            </p>
          </div>

          <p className="typography-body-medium">
            Bạn có chắc chắn muốn loại tất cả {group.statusBreakdown.pending_approval} bảng công
            đang chờ duyệt trong nhóm này?
          </p>

          <div className="space-y-2">
            <Label htmlFor="group-rejection-reason">Lý do loại *</Label>
            <Textarea
              id="group-rejection-reason"
              placeholder="Nhập lý do loại chung cho tất cả mục trong nhóm..."
              value={rejectionReason}
              onChange={(e) => setRejectionReason(e.target.value)}
              rows={3}
              required
            />
          </div>
        </div>
      }
      confirmText="Loại tất cả"
      cancelText="Hủy"
      onConfirm={handleConfirm}
      confirmVariant="destructive"
      disabled={!rejectionReason.trim() || isLoading}
      isLoading={isLoading}
    />
  );
}
