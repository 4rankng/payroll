import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Calendar } from '@/components/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { CalendarIcon, Wallet as WalletIcon } from 'lucide-react';
import { format } from 'date-fns';
import { cn } from '@/lib/utils';
import { walletService } from '@/services/api/wallet.service';
import type { CreateWalletTopupRequest } from '@/types/api/wallet.types';
import { showErrorNotification } from '@/utils/error-handler';

interface CreateTopupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

const initialForm: CreateWalletTopupRequest = {
  amount: 0,
  bank_ref: '',
  occurred_at: format(new Date(), "yyyy-MM-dd'T'00:00:00XXX"),
};

export default function CreateTopupDialog({
  open,
  onOpenChange,
  onSuccess,
}: CreateTopupDialogProps) {
  const [loading, setLoading] = useState(false);
  const [formData, setFormData] = useState<CreateWalletTopupRequest>({ ...initialForm });
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());

  const [amountError, setAmountError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (formData.amount % 1000 !== 0) {
      setAmountError('Số tiền phải là bội số của 1.000 ₫');
      return;
    }
    setAmountError(null);
    setLoading(true);

    try {
      await walletService.createTopup(formData);
      onSuccess();
      onOpenChange(false);
      setFormData({ ...initialForm });
      setSelectedDate(new Date());
    } catch (error) {
      showErrorNotification(error, 'Tạo bản ghi nạp tiền thất bại');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[calc(100vw-1rem)] max-w-[calc(100vw-1rem)] sm:max-w-[460px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <WalletIcon className="h-5 w-5 text-white/80" />
            Nạp tiền vào ví
          </DialogTitle>
          <DialogDescription>
            Nhập thông tin giao dịch nạp tiền từ ngân hàng
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2 space-y-2">
              <Label htmlFor="amount">
                Số tiền nạp <span className="text-red-600">*</span>
              </Label>
              <Input
                id="amount"
                type="text"
                inputMode="numeric"
                required
                value={formData.amount ? new Intl.NumberFormat('vi-VN').format(formData.amount) : ''}
                onChange={(e) => {
                  const raw = e.target.value.replace(/[^\d]/g, '');
                  setFormData({ ...formData, amount: parseInt(raw) || 0 });
                  setAmountError(null);
                }}
                className="text-right font-semibold tabular-nums"
              />
              {amountError && (
                <p className="text-xs text-destructive">{amountError}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label>
                Ngày nạp <span className="text-red-600">*</span>
              </Label>
              <Popover>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    className={cn(
                      'w-full justify-start text-left font-normal',
                      !selectedDate && 'text-muted-foreground'
                    )}
                  >
                    <CalendarIcon className="mr-2 h-4 w-4" />
                    {selectedDate ? format(selectedDate, 'dd/MM/yyyy') : 'Chọn ngày'}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-auto p-0">
                  <Calendar
                    mode="single"
                    selected={selectedDate}
                    onSelect={(date) => {
                      if (date) {
                        setSelectedDate(date);
                        setFormData({
                          ...formData,
                          occurred_at: format(date, "yyyy-MM-dd'T'00:00:00XXX"),
                        });
                      }
                    }}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="bank_ref">
              Mã giao dịch ngân hàng <span className="text-red-600">*</span>
            </Label>
            <Input
              id="bank_ref"
              required
              value={formData.bank_ref}
              onChange={(e) =>
                setFormData({ ...formData, bank_ref: e.target.value })
              }
              placeholder="VD: FT24050123456789"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="note">Ghi chú</Label>
            <Textarea
              id="note"
              value={formData.note || ''}
              onChange={(e) =>
                setFormData({ ...formData, note: e.target.value || undefined })
              }
              placeholder="Ngân hàng, số tài khoản, lý do nạp..."
              rows={3}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              Hủy
            </Button>
            <Button type="submit" disabled={loading} className="bg-primary hover:bg-primary/90 text-primary-foreground">
              {loading ? 'Đang nạp...' : 'Nạp tiền'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
