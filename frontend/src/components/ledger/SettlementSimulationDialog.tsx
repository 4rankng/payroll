import { useState } from 'react';
import {
  AlertTriangle,
  CheckCircle2,
  ClipboardCheck,
  Clock,
  Loader2,
  AlertCircle,
  Info,
} from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useSimulateSettlement } from '@/hooks/api/usePayrolls';
import { formatCurrency, formatDateTime } from '@/utils/formatters';
import type {
  SettlementSimulationResult,
  SettlementVerdict,
  CycleProjection,
} from '@/types/api/settlement-simulation.types';

interface SettlementSimulationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const VERDICT_META: Record<
  SettlementVerdict,
  { label: string; tone: 'green' | 'amber' | 'red'; icon: typeof CheckCircle2; description: string }
> = {
  AN_TOAN_DE_XUAT: {
    label: 'An toàn để xuất',
    tone: 'green',
    icon: CheckCircle2,
    description: 'Tất cả giao dịch chưa thanh toán sẽ được bao phủ bởi các kỳ mô phỏng.',
  },
  CAN_KIEM_TRA: {
    label: 'Cần kiểm tra',
    tone: 'amber',
    icon: AlertTriangle,
    description: 'Có giao dịch chưa được bao phủ, có cảnh báo, hoặc đối soát lệch.',
  },
  KHONG_THE_TAT_TOAN: {
    label: 'Không thể tất toán',
    tone: 'red',
    icon: AlertCircle,
    description: 'Có rủi ro cao — liên hệ kỹ thuật trước khi xuất.',
  },
};

const TONE_CLASS: Record<'green' | 'amber' | 'red', string> = {
  green: 'border-green-500 bg-green-50 text-green-900',
  amber: 'border-amber-500 bg-amber-50 text-amber-900',
  red: 'border-red-500 bg-red-50 text-red-900',
};

export function SettlementSimulationDialog({ open, onOpenChange }: SettlementSimulationDialogProps) {
  const [cycleCount, setCycleCount] = useState(4);
  const [result, setResult] = useState<SettlementSimulationResult | null>(null);

  const simulation = useSimulateSettlement();

  const handleSubmit = async () => {
    const data = await simulation.mutateAsync({ projected_cycle_count: cycleCount });
    setResult(data);
  };

  const handleClose = () => {
    setResult(null);
    simulation.reset();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={(o) => (o ? onOpenChange(true) : handleClose())}>
      <DialogContent className="sm:max-w-3xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ClipboardCheck className="w-5 h-5" />
            Mô phỏng đối soát (dry-run)
          </DialogTitle>
          <p className="text-sm text-muted-foreground">
            Xem trước lần xuất sao kê này và các kỳ tiếp theo. Không thay đổi dữ liệu.
          </p>
        </DialogHeader>

        <ScrollArea className="flex-1 pr-4">
          <div className="space-y-4">
            {/* Controls */}
            <div className="flex items-end gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Số kỳ mô phỏng</label>
                <Select value={String(cycleCount)} onValueChange={(v) => setCycleCount(Number(v))}>
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {[1, 2, 3, 4, 5, 6].map((n) => (
                      <SelectItem key={n} value={String(n)}>
                        {n} kỳ
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button onClick={handleSubmit} disabled={simulation.isPending}>
                {simulation.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin mr-2" />
                    Đang mô phỏng...
                  </>
                ) : (
                  'Chạy mô phỏng'
                )}
              </Button>
            </div>

            {simulation.isError && (
              <Alert variant="destructive">
                <AlertCircle className="w-4 h-4" />
                <AlertTitle>Lỗi mô phỏng</AlertTitle>
                <AlertDescription>{(simulation.error as Error).message}</AlertDescription>
              </Alert>
            )}

            {result && <ResultView result={result} />}
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Đóng
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ResultView({ result }: { result: SettlementSimulationResult }) {
  const meta = VERDICT_META[result.verdict];
  const VerdictIcon = meta.icon;
  const { summary, reconciliation, cycles, remainders, warnings } = result;

  return (
    <div className="space-y-4">
      {/* Verdict banner */}
      <Alert className={TONE_CLASS[meta.tone]}>
        <VerdictIcon className="w-5 h-5" />
        <AlertTitle className="text-base font-semibold">{meta.label}</AlertTitle>
        <AlertDescription>{meta.description}</AlertDescription>
      </Alert>

      {/* Summary stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatCard label="Tổng đủ điều kiện" value={`${summary.total_eligible_count}`} sub={formatCurrency(summary.total_eligible_amount)} />
        <StatCard label="Đã bao phủ" value={`${summary.total_included_count}`} sub={formatCurrency(summary.total_included_amount)} />
        <StatCard
          label="Còn lại sau các kỳ"
          value={`${summary.remaining_after_all_count}`}
          sub={formatCurrency(summary.remaining_after_all_amount)}
          tone={summary.remaining_after_all_count > 0 ? 'amber' : 'green'}
        />
        <StatCard
          label="Đối soát sổ cái"
          value={reconciliation.reconciled ? 'Khớp' : 'Lệch'}
          sub={reconciliation.reconciled ? `${formatCurrency(reconciliation.exported_total)}` : `Δ ${formatCurrency(reconciliation.delta)}`}
          tone={reconciliation.reconciled ? 'green' : 'red'}
        />
      </div>

      {/* Per-cycle table */}
      <div>
        <h3 className="text-sm font-semibold mb-2">Chi tiết từng kỳ</h3>
        <CycleTable cycles={cycles} />
      </div>

      {/* Remainders */}
      {remainders.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold mb-2 flex items-center gap-2">
            <Clock className="w-4 h-4 text-amber-600" />
            Còn lại (chưa bao phủ) — {remainders.length} giao dịch, {formatCurrency(remainders.reduce((s, r) => s + r.amount, 0))}
          </h3>
          <p className="text-xs text-muted-foreground mb-2">
            Các giao dịch thuộc kỳ trước vẫn chưa thanh toán sẽ <strong>KHÔNG</strong> tự động được bao phủ.
            Xem danh sách dưới đây để xử lý thủ công.
          </p>
          <RemaindersTable remainders={remainders} />
        </div>
      )}

      {/* Warnings */}
      {warnings.length > 0 && (
        <div className="space-y-2">
          {warnings.map((w, i) => (
            <Alert key={i} variant="default">
              <Info className="w-4 h-4" />
              <AlertDescription className="text-sm">{w.message}</AlertDescription>
            </Alert>
          ))}
        </div>
      )}

      {/* Snapshot epoch */}
      <p className="text-xs text-muted-foreground">
        Dữ liệu chốt tại {formatDateTime(result.snapshot_epoch)}. Nếu có thay đổi, chạy lại mô phỏng trước khi xuất.
      </p>
    </div>
  );
}

function StatCard({
  label,
  value,
  sub,
  tone = 'default',
}: {
  label: string;
  value: string;
  sub?: string;
  tone?: 'default' | 'green' | 'amber' | 'red';
}) {
  const toneClass =
    tone === 'green'
      ? 'text-green-700'
      : tone === 'amber'
        ? 'text-amber-700'
        : tone === 'red'
          ? 'text-red-700'
          : 'text-foreground';
  return (
    <div className="border rounded-lg p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`text-lg font-semibold ${toneClass}`}>{value}</div>
      {sub && <div className={`text-xs ${toneClass}`}>{sub}</div>}
    </div>
  );
}

function CycleTable({ cycles }: { cycles: CycleProjection[] }) {
  return (
    <div className="border rounded-lg overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-32">Kỳ</TableHead>
            <TableHead>Từ − Đến</TableHead>
            <TableHead className="text-right">Bao gồm</TableHead>
            <TableHead className="text-right">Loại trừ</TableHead>
            <TableHead className="text-right">Số tiền</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {cycles.map((c) => (
            <TableRow key={c.sequence}>
              <TableCell className="font-medium">{c.label}</TableCell>
              <TableCell className="text-xs text-muted-foreground">
                {c.from_date} → {c.to_date}
                <div className="text-[10px]">Trả lương: {c.pay_date}</div>
              </TableCell>
              <TableCell className="text-right">{c.included_count}</TableCell>
              <TableCell className="text-right">{c.excluded_count}</TableCell>
              <TableCell className="text-right font-mono">{formatCurrency(c.included_amount)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function RemaindersTable({ remainders }: { remainders: SettlementSimulationResult['remainders'] }) {
  return (
    <div className="border rounded-lg overflow-hidden max-h-72 overflow-y-auto">
      <Table>
        <TableHeader className="sticky top-0 bg-background">
          <TableRow>
            <TableHead>Nhân viên</TableHead>
            <TableHead>Dự án</TableHead>
            <TableHead className="text-right">Số tiền</TableHead>
            <TableHead className="text-right">Timesheets</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {remainders.map((r, i) => (
            <TableRow key={`${r.employee_id}-${r.project_id}-${i}`}>
              <TableCell>
                <div className="font-medium">{r.employee_name}</div>
                <div className="text-xs text-muted-foreground">{r.bank_account_masked || '—'}</div>
              </TableCell>
              <TableCell className="text-sm">{r.project_name}</TableCell>
              <TableCell className="text-right font-mono">{formatCurrency(r.amount)}</TableCell>
              <TableCell className="text-right text-xs text-muted-foreground">
                {r.timesheet_ids.join(', ')}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

// Re-export for callers that want to show the badge in finding lists.
export function FindingBadge({ inProduction }: { inProduction: boolean }) {
  return inProduction ? (
    <Badge variant="secondary">Sản xuất cũng kiểm tra</Badge>
  ) : (
    <Badge variant="outline" className="border-amber-500 text-amber-700">
      Sản xuất không kiểm tra
    </Badge>
  );
}
