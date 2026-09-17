import { useState, useCallback } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  FileText, FileSpreadsheet, Download, Upload, CheckCircle2,
  Receipt, MoreHorizontal, Eye, Loader2,
} from 'lucide-react';
import { format, parseISO, subHours } from 'date-fns';
import { vi } from 'date-fns/locale';
import { showErrorNotification } from '@/utils/error-handler';
import { advancePaymentService } from '@/services/api/advance-payment.service';
import { cn } from '@/lib/utils';

export type AdvancePaymentFileType =
  | 'flex_pay_import'
  | 'advance_payment_result'
  | 'advance_payment_export'
  | 'advance_payment_sao_ke_export'
  | 'advance_payment_sao_ke_result';

export interface UnifiedFileItem {
  id: number;
  filename: string;
  type: AdvancePaymentFileType;
  date: string;
  uploadedBy: string;
}

const TYPE_CONFIG = {
  flex_pay_import: {
    label: 'Nhập bảng lương',
    iconboxClass: 'bg-blue-100 text-blue-600',
    textClass: 'text-blue-600',
    dotClass: 'bg-blue-500',
    Icon: Upload,
  },
  advance_payment_result: {
    label: 'Kết quả ứng lương',
    iconboxClass: 'bg-emerald-100 text-emerald-600',
    textClass: 'text-emerald-600',
    dotClass: 'bg-emerald-500',
    Icon: CheckCircle2,
  },
  advance_payment_export: {
    label: 'Xuất chuyển lô',
    iconboxClass: 'bg-amber-100 text-amber-600',
    textClass: 'text-amber-600',
    dotClass: 'bg-amber-500',
    Icon: Download,
  },
  advance_payment_sao_ke_export: {
    label: 'Xuất sao kê',
    iconboxClass: 'bg-teal-100 text-teal-600',
    textClass: 'text-teal-600',
    dotClass: 'bg-teal-500',
    Icon: FileSpreadsheet,
  },
  advance_payment_sao_ke_result: {
    label: 'Kết quả thanh toán sao kê',
    iconboxClass: 'bg-rose-100 text-rose-600',
    textClass: 'text-rose-600',
    dotClass: 'bg-rose-500',
    Icon: FileText,
  },
} as const;

const DEFAULT_CONFIG = {
  label: 'File',
  iconboxClass: 'bg-gray-100 text-gray-600',
  textClass: 'text-gray-600',
  dotClass: 'bg-gray-500',
  Icon: FileText,
};

function isRecent(dateStr: string): boolean {
  try {
    return parseISO(dateStr) > subHours(new Date(), 24);
  } catch {
    return false;
  }
}

interface UnifiedFileCardProps {
  file: UnifiedFileItem;
  isNew?: boolean;
  alwaysShowActions?: boolean;
}

export function UnifiedFileCard({ file, isNew, alwaysShowActions }: UnifiedFileCardProps) {
  const [downloading, setDownloading] = useState(false);
  const cfg = TYPE_CONFIG[file.type as keyof typeof TYPE_CONFIG] ?? DEFAULT_CONFIG;
  const { Icon } = cfg;
  const showNewBadge = isNew ?? isRecent(file.date);

  const handleDownload = useCallback(async () => {
    setDownloading(true);
    try {
      await advancePaymentService.downloadUploadedFile(file.id, file.filename);
    } catch (err) {
      showErrorNotification(err);
    } finally {
      setDownloading(false);
    }
  }, [file.id, file.filename]);

  return (
    <div className={cn(
      'group grid grid-cols-[40px_1fr] gap-3 rounded-lg px-3 py-3 transition-colors sm:grid-cols-[40px_1fr_auto] sm:items-center',
      'hover:bg-muted/50'
    )}>
      {/* Iconbox */}
      <div className={cn('w-10 h-10 rounded-[10px] flex items-center justify-center shrink-0', cfg.iconboxClass)}>
        <Icon className="w-[18px] h-[18px]" />
      </div>

      {/* Metadata */}
      <div className="min-w-0">
        <div className="flex min-w-0 flex-wrap items-center gap-1.5">
          <p className="min-w-0 break-words text-sm font-semibold leading-snug sm:truncate" title={file.filename}>{file.filename}</p>
          {showNewBadge && (
            <Badge className="bg-emerald-100 text-emerald-700 text-xs px-1.5 py-0 rounded-sm font-bold shrink-0 border-0 leading-4">
              MỚI
            </Badge>
          )}
        </div>
        <div className="mt-1 flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs text-muted-foreground sm:truncate">
          <span className={cn('font-medium', cfg.textClass)}>{cfg.label}</span>
          <span className="text-border">·</span>
          <span className="min-w-0 break-words sm:truncate">{file.uploadedBy}</span>
          <span className="text-border">·</span>
          <span className="font-mono text-xs shrink-0">
            {file.date ? format(parseISO(file.date), 'dd/MM · HH:mm', { locale: vi }) : ''}
          </span>
        </div>
      </div>

      {/* Actions */}
      <div className={cn(
        'col-span-2 ml-12 flex items-center gap-1 shrink-0 sm:col-span-1 sm:ml-0',
        alwaysShowActions
          ? 'opacity-100'
          : 'opacity-0 group-hover:opacity-100 translate-x-1 group-hover:translate-x-0 transition-all duration-200'
      )}>
        <Button
          variant="ghost"
          size="icon"
          className="h-11 w-11 text-muted-foreground hover:text-foreground"
          aria-label="Xem"
        >
          <Eye className="w-4 h-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-11 w-11 text-foreground hover:bg-foreground hover:text-background"
          onClick={handleDownload}
          disabled={downloading}
          aria-label="Tải xuống"
        >
          {downloading
            ? <Loader2 className="w-4 h-4 animate-spin" />
            : <Download className="w-4 h-4" />
          }
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-11 w-11 text-muted-foreground hover:text-foreground"
          aria-label="Thêm"
        >
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
