import { useState, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Download } from "lucide-react";

const MONTH_OPTIONS = [1, 2, 3, 6, 12, 24];

interface ExportPaidNoMobileModalProps {
  open: boolean;
  onClose: () => void;
  onExport: (months: number) => void;
  isExporting?: boolean;
}

/**
 * Chooses the look-back window for the admin export of employees who received
 * salary or FlexPay payment recently but have no mobile number on file.
 */
export function ExportPaidNoMobileModal({
  open,
  onClose,
  onExport,
  isExporting = false,
}: ExportPaidNoMobileModalProps) {
  const [months, setMonths] = useState(3);

  const handleClose = useCallback(() => onClose(), [onClose]);

  const handleExport = useCallback(() => {
    onExport(months);
    handleClose();
  }, [onExport, months, handleClose]);

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md gap-0">
        <DialogHeader className="pb-1">
          <DialogTitle className="text-sm font-semibold">
            Xuất nhân viên chưa có số điện thoại
          </DialogTitle>
          <DialogDescription className="text-xs">
            Nhân viên đã nhận lương hoặc tạm ứng trong khoảng đã chọn nhưng chưa
            có số điện thoại trong hệ thống.
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center justify-between gap-3 px-1 py-2">
          <span className="text-xs text-muted-foreground">Khoảng thời gian</span>
          <Select value={String(months)} onValueChange={(v) => setMonths(Number(v))}>
            <SelectTrigger className="h-8 w-44 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MONTH_OPTIONS.map((m) => (
                <SelectItem key={m} value={String(m)} className="text-xs">
                  {m} tháng gần nhất
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="outline" size="sm" onClick={handleClose} disabled={isExporting}>
            Huỷ
          </Button>
          <Button size="sm" onClick={handleExport} disabled={isExporting}>
            <Download className="mr-1.5 h-3.5 w-3.5" />
            {isExporting ? "Đang xuất..." : "Xuất Excel"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
