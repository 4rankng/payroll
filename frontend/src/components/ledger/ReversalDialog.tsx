import { format } from "date-fns";
import React, { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { ledgerService } from '@/services/api/ledger.service';
import type { LedgerEntry } from '@/types/api/financial.types';

interface ReversalDialogProps {
  isOpen: boolean;
  onClose: () => void;
  entry: LedgerEntry | null;
  onConfirm: (entryId: number, reason: string) => Promise<void>;
  isLoading?: boolean;
}

export function ReversalDialog({
  isOpen,
  onClose,
  entry,
  onConfirm,
  isLoading = false,
}: ReversalDialogProps) {
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!reason.trim()) {
      setError('Vui lòng nhập lý do đảo ngược');
      return;
    }

    if (!entry) return;

    try {
      setError('');
      await onConfirm(entry.id, reason.trim());
      setReason('');
      onClose();
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Có lỗi xảy ra');
    }
  };

  const handleClose = () => {
    setReason('');
    setError('');
    onClose();
  };

  if (!entry) return null;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Xác Nhận Đảo Ngược Bút Toán</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Entry Details */}
          <div className="bg-muted/50 p-4 rounded-xl space-y-2">
            <div className="typography-body-small text-muted-foreground mb-2">Thông tin bút toán:</div>
            <div className="space-y-1">
              <div className="flex justify-between">
                <span className="typography-body-medium">Ngày:</span>
                <span className="typography-body-medium font-medium">
                  {format(new Date(entry.date), 'dd/MM/yyyy')}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="typography-body-medium">Đối tượng:</span>
                <span className="typography-body-medium font-medium">{entry.party}</span>
              </div>
              <div className="flex justify-between">
                <span className="typography-body-medium">Tài khoản:</span>
                <span className="typography-body-medium font-medium">
                  {ledgerService.getAccountDisplayName(entry.account)}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="typography-body-medium">Số tiền:</span>
                <span className="typography-body-medium font-medium">
                  {entry.debit > 0
                    ? `${ledgerService.formatCurrency(entry.debit)} (Nợ)`
                    : `${ledgerService.formatCurrency(entry.credit)} (Có)`
                  }
                </span>
              </div>
              <div className="pt-2">
                <div className="typography-body-medium">Diễn giải:</div>
                <div className="typography-body-medium font-medium text-foreground mt-1">
                  {entry.description}
                </div>
              </div>
            </div>
          </div>

          {/* Warning */}
          <div className="bg-amber-50 border border-amber-200 p-3 rounded-xl">
            <div className="typography-body-small text-amber-800">
              <strong>Lưu ý:</strong> Thao tác này sẽ tạo một bút toán đảo ngược với số tiền và tài khoản ngược lại
              để hủy bỏ tác động của bút toán gốc. Bút toán gốc sẽ được giữ lại để đảm bảo tính minh bạch.
            </div>
          </div>

          {/* Reason Input */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="reason">Lý do đảo ngược *</Label>
              <Textarea
                id="reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Nhập lý do tại sao cần đảo ngược bút toán này..."
                rows={3}
                className={error ? 'border-red-300 focus-visible:ring-red-500' : ''}
              />
              {error && (
                <div className="typography-body-small text-red-600">{error}</div>
              )}
            </div>

            {/* Actions */}
            <div className="flex justify-end gap-3 pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={handleClose}
                disabled={isLoading}
              >
                Đóng
              </Button>
              <Button
                type="submit"
                variant="destructive"
                disabled={isLoading}
              >
                {isLoading ? 'Đang xử lý...' : 'Đảo Ngược'}
              </Button>
            </div>
          </form>
        </div>
      </DialogContent>
    </Dialog>
  );
}
