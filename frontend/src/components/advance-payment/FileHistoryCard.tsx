import { useState } from "react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileText, FileSpreadsheet, Download, Loader2 } from "lucide-react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { showErrorNotification } from "@/utils/error-handler";
import { advancePaymentService } from "@/services/api/advance-payment.service";
import type { AdvancePaymentFileHistoryItem } from "@/types/api/advance-payment.types";

const TYPE_CONFIG = {
  flex_pay_import: {
    label: "Nhập bảng lương",
    badgeClass: "bg-blue-100 text-blue-800 border-blue-200",
    Icon: FileSpreadsheet,
  },
  advance_payment_result: {
    label: "Kết quả ứng lương",
    badgeClass: "bg-emerald-100 text-emerald-800 border-emerald-200",
    Icon: FileText,
  },
  advance_payment_export: {
    label: "Xuất chuyển lô",
    badgeClass: "bg-amber-100 text-amber-800 border-amber-200",
    Icon: Download,
  },
  advance_payment_sao_ke_export: {
    label: "Xuất sao kê",
    badgeClass: "bg-purple-100 text-purple-800 border-purple-200",
    Icon: FileSpreadsheet,
  },
  advance_payment_sao_ke_result: {
    label: "Kết quả thanh toán sao kê",
    badgeClass: "bg-rose-100 text-rose-800 border-rose-200",
    Icon: FileText,
  },
} as const;

export function FileHistoryCard({ file }: { file: AdvancePaymentFileHistoryItem }) {
  const [downloading, setDownloading] = useState(false);
  const cfg = TYPE_CONFIG[file.uploadType] ?? TYPE_CONFIG.advance_payment_result;
  const { Icon } = cfg;

  const handleDownload = async () => {
    setDownloading(true);
    try {
      await advancePaymentService.downloadUploadedFile(file.id, file.filename);
    } catch (err) {
      showErrorNotification(err);
    } finally {
      setDownloading(false);
    }
  };

  return (
    <Card className="p-3">
      <div className="flex items-start gap-3">
        <Icon className="h-5 w-5 text-muted-foreground mt-0.5 shrink-0" />
        <div className="flex-1 min-w-0">
          <p className="font-medium truncate text-sm">{file.filename}</p>
          <div className="flex items-center gap-2 mt-1">
            <Badge className={cfg.badgeClass} variant="outline">
              {cfg.label}
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            {file.createdAt &&
              format(new Date(file.createdAt), "dd/MM/yyyy HH:mm", { locale: vi })}
          </p>
        </div>
        <button
          onClick={handleDownload}
          disabled={downloading}
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg transition-colors hover:bg-muted disabled:opacity-50"
          aria-label="Tải xuống"
        >
          {downloading
            ? <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            : <Download className="h-4 w-4 text-muted-foreground" />
          }
        </button>
      </div>
    </Card>
  );
}
