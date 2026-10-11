import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { ApprovalRequest } from '@/types/api/approval.types';

export type ApprovalDecisionMode = 'approve' | 'reject' | 'cancel';

interface ApprovalDecisionModalProps {
  open: boolean;
  mode: ApprovalDecisionMode | null;
  request: ApprovalRequest | null;
  isPending: boolean;
  onClose: () => void;
  onConfirm: (id: number, note: string) => void;
}

const MODE_COPY: Record<
  ApprovalDecisionMode,
  { title: string; description: string; confirm: string; confirmClass: string; noteRequired: boolean; noteLabel: string; noteHint: string }
> = {
  approve: {
    title: 'Duyệt yêu cầu',
    description: 'Duyệt sẽ thực thi hành động (ví dụ đưa email vào hàng đợi gửi) và ghi lại kết quả.',
    confirm: 'Duyệt',
    confirmClass: 'bg-emerald-600 text-white hover:bg-emerald-700',
    noteRequired: false,
    noteLabel: 'Ghi chú (không bắt buộc)',
    noteHint: 'Ghi chú nội bộ kèm quyết định.',
  },
  reject: {
    title: 'Từ chối — thử lại ngày mai',
    description:
      'Yêu cầu sẽ được chuẩn bị lại lúc 08:00 ngày mai và quay lại hàng chờ duyệt. Sau 3 lần từ chối, yêu cầu sẽ được leo thang.',
    confirm: 'Từ chối',
    confirmClass: 'bg-amber-600 text-white hover:bg-amber-700',
    noteRequired: true,
    noteLabel: 'Lý do / hướng dẫn (bắt buộc)',
    noteHint: 'Ví dụ: "gửi mai, hôm nay dữ liệu chưa đủ" — ghi chú này định hướng lần chuẩn bị lại.',
  },
  cancel: {
    title: 'Huỷ yêu cầu',
    description: 'Huỷ là chốt hạ: không thử lại, không leo thang. Yêu cầu này sẽ không được thực thi.',
    confirm: 'Huỷ vĩnh viễn',
    confirmClass: 'bg-red-600 text-white hover:bg-red-700',
    noteRequired: false,
    noteLabel: 'Ghi chú (không bắt buộc)',
    noteHint: 'Lý do huỷ, lưu lại cùng lịch sử yêu cầu.',
  },
};

/** Decision dialog for the approval engine. Reject requires guidance — the
 * note is the instruction the next automated preparation run follows. */
export const ApprovalDecisionModal = ({
  open,
  mode,
  request,
  isPending,
  onClose,
  onConfirm,
}: ApprovalDecisionModalProps) => {
  const [note, setNote] = useState('');
  const copy = mode ? MODE_COPY[mode] : null;

  useEffect(() => {
    if (open) setNote('');
  }, [open, mode]);

  if (!mode || !copy) return null;

  const noteTrimmed = note.trim();
  const canConfirm = !isPending && (!copy.noteRequired || noteTrimmed.length > 0);

  return (
    <Dialog open={open} onOpenChange={(next) => !isPending && !next && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>{copy.description}</DialogDescription>
        </DialogHeader>

        {request ? (
          <div className="rounded-lg border bg-muted/30 px-3 py-2.5 text-sm">
            <p className="font-medium text-foreground">
              #{request.id} · {request.type_label}
            </p>
            <p className="mt-0.5 text-xs text-muted-foreground">
              Trạng thái: {request.status_label}
              {request.retry_count > 0 ? ` · đã thử lại ${request.retry_count} lần` : ''}
            </p>
          </div>
        ) : null}

        <div className="space-y-2">
          <Label htmlFor="approval-decision-note" className="text-sm font-medium">
            {copy.noteLabel}
          </Label>
          <Textarea
            id="approval-decision-note"
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder={copy.noteHint}
            disabled={isPending}
            rows={3}
            className="resize-none"
          />
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button type="button" variant="outline" onClick={onClose} disabled={isPending}>
            Đóng
          </Button>
          <Button
            type="button"
            className={copy.confirmClass}
            disabled={!canConfirm || !request}
            onClick={() => request && onConfirm(request.id, noteTrimmed)}
          >
            {isPending ? <Loader2 aria-hidden="true" className="size-4 animate-spin" /> : null}
            {copy.confirm}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
