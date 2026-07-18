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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
import { formatCurrency, formatDate, formatDateTime } from '@/utils/formatters';
import { dateToString } from '@/utils/dateHelpers';
import type {
  SettlementSimulationResult,
  SettlementVerdict,
  ExportProjection,
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
    description: 'Tất cả giao dịch chưa đối soát sẽ được bao phủ bởi các lần xuất đã mô phỏng.',
  },
  CAN_KIEM_TRA: {
    label: 'Cần kiểm tra',
    tone: 'amber',
    icon: AlertTriangle,
    description: 'Có giao dịch chưa được bao phủ, cảnh báo, hoặc đối soát sổ cái lệch.',
  },
};

const TONE_CLASS: Record<'green' | 'amber' | 'red', string> = {
  green: 'border-green-500 bg-green-50 text-green-900',
  amber: 'border-amber-500 bg-amber-50 text-amber-900',
  red: 'border-red-500 bg-red-50 text-red-900',
};

export function SettlementSimulationDialog({ open, onOpenChange }: SettlementSimulationDialogProps) {
  const today = dateToString(new Date());
  const [startDate, setStartDate] = useState(today);
  const [exportCount, setExportCount] = useState(4);
  const [cadenceDays, setCadenceDays] = useState(7);
  const [result, setResult] = useState<SettlementSimulationResult | null>(null);

  const simulation = useSimulateSettlement();

  const handleSubmit = async () => {
    const data = await simulation.mutateAsync({ start_date: startDate, export_count: exportCount, cadence_days: cadenceDays });
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
            Xem trước các lần xuất sao kê kế tiếp. Không thay đổi dữ liệu. Sử dụng cùng logic chọn dữ liệu như xuất sao kê thật.
          </p>
        </DialogHeader>

        <ScrollArea className="flex-1 pr-4">
          <div className="space-y-4">
            {/* Controls */}
            <div className="flex flex-wrap items-end gap-3">
              <div className="space-y-1.5">
                <Label className="text-sm font-medium">Ngày xuất đầu tiên</Label>
                <Input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="w-40"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium">Số lần xuất</Label>
                <Select value={String(exportCount)} onValueChange={(v) => setExportCount(Number(v))}>
                  <SelectTrigger className="w-28">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {[1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map((n) => (
                      <SelectItem key={n} value={String(n)}>
                        {n} lần
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium">Khoảng cách (ngày)</Label>
                <Select value={String(cadenceDays)} onValueChange={(v) => setCadenceDays(Number(v))}>
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="7">7 ngày</SelectItem>
                    <SelectItem value="14">14 ngày</SelectItem>
                    <SelectItem value="30">30 ngày</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button onClick={handleSubmit} disabled={simulation.isPending || !startDate}>
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
  const meta = VERDICT_META[result.verdict] ?? VERDICT_META.CAN_KIEM_TRA;
  const VerdictIcon = meta.icon;
  const { summary, reconciliation, exports, remainders, warnings } = result;

  return (
    <div className="space-y-4">
      {/* Export dates preview */}
      <div className="text-xs text-muted-foreground">
        Các ngày xuất sẽ mô phỏng: <strong>{result.export_dates.map((d) => formatDate(d)).join(', ')}</strong>
      </div>

      {/* Verdict banner */}
      <Alert className={TONE_CLASS[meta.tone]}>
        <VerdictIcon className="w-5 h-5" />
        <AlertTitle className="text-base font-semibold">{meta.label}</AlertTitle>
        <AlertDescription>{meta.description}</AlertDescription>
      </Alert>

      {/* Summary stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <StatCard
          label="Tổng chưa đối soát"
          value={`${summary.total_eligible_timesheets}`}
          sub={`${summary.total_eligible_groups} nhóm · ${formatCurrency(summary.total_eligible_amount)}`}
        />
        <StatCard
          label="Đã bao phủ"
          value={`${summary.total_included_timesheets}`}
          sub={formatCurrency(summary.total_included_amount)}
          tone="green"
        />
        <StatCard
          label="Còn lại"
          value={`${summary.remaining_timesheets}`}
          sub={`${summary.remaining_groups} nhóm · ${formatCurrency(summary.remaining_amount)}`}
          tone={summary.remaining_timesheets > 0 ? 'amber' : 'green'}
        />
        <StatCard
          label="Đối soát sổ cái"
          value={reconciliation.reconciled ? 'Khớp' : 'Lệch'}
          sub={reconciliation.reconciled ? formatCurrency(reconciliation.exported_total) : `Δ ${formatCurrency(reconciliation.delta)}`}
          tone={reconciliation.reconciled ? 'green' : 'red'}
        />
      </div>

      {/* Per-export table */}
      <div>
        <h3 className="text-sm font-semibold mb-2">Chi tiết từng lần xuất</h3>
        <ExportTable exports={exports} />
      </div>

      {/* Remainders */}
      {remainders.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold mb-2 flex items-center gap-2">
            <Clock className="w-4 h-4 text-amber-600" />
            Còn lại (chưa bao phủ) — {remainders.length} nhóm, {formatCurrency(remainders.reduce((s, r) => s + r.amount, 0))}
          </h3>
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

      {/* Snapshot */}
      {result.snapshot_epoch && result.snapshot_epoch !== '0001-01-01T00:00:00Z' && (
        <p className="text-xs text-muted-foreground">
          Dữ liệu chốt tại {formatDateTime(result.snapshot_epoch)}. Nếu có thay đổi, chạy lại mô phỏng.
        </p>
      )}
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

function ExportTable({ exports }: { exports: ExportProjection[] }) {
  return (
    <div className="border rounded-lg overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-20">#</TableHead>
            <TableHead>Ngày xuất</TableHead>
            <TableHead className="text-right">Bao gồm</TableHead>
            <TableHead className="text-right">Số tiền</TableHead>
            <TableHead className="text-right">Còn lại</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {exports.map((e) => (
            <TableRow key={e.sequence}>
              <TableCell className="font-medium">{e.sequence}</TableCell>
              <TableCell className="text-sm">{formatDate(e.export_date)}</TableCell>
              <TableCell className="text-right">{e.included_count}</TableCell>
              <TableCell className="text-right font-mono">{formatCurrency(e.included_amount)}</TableCell>
              <TableCell className="text-right text-muted-foreground">{e.remaining_count || '—'}</TableCell>
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
            <TableHead>Ngày công</TableHead>
            <TableHead className="text-right">SL</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {remainders.map((r, i) => (
            <TableRow key={`${r.employee_id}-${r.project_id}-${i}`}>
              <TableCell className="font-medium">{r.employee_name}</TableCell>
              <TableCell className="text-sm">{r.project_name}</TableCell>
              <TableCell className="text-right font-mono">{formatCurrency(r.amount)}</TableCell>
              <TableCell className="text-xs text-muted-foreground">
                {r.timesheet_dates.length > 0
                  ? `${formatDate(r.timesheet_dates[0])}${r.timesheet_dates.length > 1 ? ` (+${r.timesheet_dates.length - 1})` : ''}`
                  : '—'}
              </TableCell>
              <TableCell className="text-right text-xs">{r.timesheet_ids.length}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
