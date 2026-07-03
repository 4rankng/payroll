import { useState, memo, useCallback, useMemo, useEffect } from 'react';
import { Dialog, DialogContent, DialogClose } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Mail, Loader2, AlertCircle, Calendar as CalendarIcon, X, Plus, FileSpreadsheet } from 'lucide-react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { cn } from '@/lib/utils';
import { formatDateForAPI } from '@/utils/formatters';
import { apiClient, buildQueryString } from '@/services/api/client';
import { API_ENDPOINTS } from '@/config/api.config';
import { getErrorMessage } from '@/utils/error-handler';
import { DEFAULT_SAOKE_RECIPIENTS, DEFAULT_SAOKE_CC } from '@/constants/emailDefaults';

export interface PayrollReportEmailParams {
  reportAtDate: string;
  recipients: string[];
  cc: string[];
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSendEmail: (params: PayrollReportEmailParams) => void;
  isLoading?: boolean;
}

const DEFAULT_RECIPIENTS = [...DEFAULT_SAOKE_RECIPIENTS];
const DEFAULT_CC = [...DEFAULT_SAOKE_CC];
const isValidEmail = (v: string) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v);

/* ── pill tag ── */
const Tag = memo(function Tag({ email, onRemove, primary }: { email: string; onRemove: () => void; primary?: boolean }) {
  return (
    <span className={cn(
      'inline-flex min-h-8 items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-medium',
      primary ? 'bg-primary/10 text-primary border-primary/20' : 'bg-muted text-muted-foreground border-border',
    )}>
      <span className="max-w-[180px] break-all leading-snug sm:truncate">{email}</span>
      <button type="button" onClick={onRemove} className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full hover:bg-background/70 hover:opacity-80" aria-label={`Xóa ${email}`}>
        <X className="h-3 w-3" />
      </button>
    </span>
  );
});

/* ── email column (To or CC) ── */
interface ColProps {
  label: string;
  sublabel: string;
  placeholder: string;
  value: string;
  emails: string[];
  error: string;
  primary?: boolean;
  onChange: (v: string) => void;
  onAdd: () => void;
  onRemove: (i: number) => void;
  onKeyDown: (e: React.KeyboardEvent) => void;
}

const EmailCol = memo(function EmailCol({ label, sublabel, placeholder, value, emails, error, primary, onChange, onAdd, onRemove, onKeyDown }: ColProps) {
  return (
    <div className="flex flex-col gap-1.5 min-w-0">
      <div className="flex items-center gap-1.5">
        <span className="text-xs font-semibold">{label}</span>
        <span className="text-xs text-muted-foreground">{sublabel}</span>
        <span className="ml-auto rounded-full bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground tabular-nums">{emails.length}</span>
      </div>
      <div className="flex gap-2">
        <Input
          type="email"
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onKeyDown={onKeyDown}
          className={cn('min-h-11 text-sm', error && 'border-destructive')}
        />
        <Button type="button" onClick={onAdd} size="icon" variant="outline" className="h-11 w-11 shrink-0 border-dashed" aria-label={`Thêm ${label}`}>
          <Plus className="h-4 w-4" />
        </Button>
      </div>
      {error && <p className="flex items-center gap-1 text-xs text-destructive"><AlertCircle className="h-3 w-3" />{error}</p>}
      <div className={cn(
        'flex flex-wrap gap-1 rounded-xl border border-border/50 bg-muted/30 px-2 py-1.5 min-h-[32px]',
        emails.length === 0 && 'opacity-40',
      )}>
        {emails.length === 0
          ? <span className="text-xs text-muted-foreground italic">Chưa có</span>
          : emails.map((e, i) => <Tag key={i} email={e} onRemove={() => onRemove(i)} primary={primary} />)
        }
      </div>
    </div>
  );
});

/* ── dialog ── */
export const PayrollReportEmailDialog = memo(function PayrollReportEmailDialog({ open, onOpenChange, onSendEmail, isLoading = false }: Props) {
  const [date, setDate] = useState<Date>();
  const [calOpen, setCalOpen] = useState(false);
  const [recipients, setRecipients] = useState<string[]>(DEFAULT_RECIPIENTS);
  const [cc, setCc] = useState<string[]>(DEFAULT_CC);
  const [curTo, setCurTo] = useState('');
  const [curCc, setCurCc] = useState('');
  const [toErr, setToErr] = useState('');
  const [ccErr, setCcErr] = useState('');
  const [attempted, setAttempted] = useState(false);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [previewError, setPreviewError] = useState('');

  useEffect(() => {
    if (open) {
      setDate(new Date()); setCalOpen(false);
      setRecipients(DEFAULT_RECIPIENTS); setCc(DEFAULT_CC);
      setCurTo(''); setCurCc(''); setToErr(''); setCcErr(''); setAttempted(false);
    }
  }, [open]);

  const canSend = useMemo(() => !!date && recipients.length > 0, [date, recipients.length]);

  const add = useCallback((raw: string, list: string[], setList: (l: string[]) => void, setCur: (v: string) => void, setErr: (v: string) => void) => {
    const v = raw.trim();
    if (!v) { setErr('Không được để trống'); return; }
    if (!isValidEmail(v)) { setErr('Email không hợp lệ'); return; }
    if (list.includes(v)) { setErr('Đã tồn tại'); return; }
    setList([...list, v]); setCur(''); setErr('');
  }, []);

  const remove = useCallback((i: number, list: string[], setList: (l: string[]) => void) => {
    setList(list.filter((_, idx) => idx !== i));
  }, []);

  const handleSend = useCallback(() => {
    setAttempted(true);
    if (!canSend || !date) return;
    onSendEmail({ reportAtDate: formatDateForAPI(date), recipients, cc });
  }, [canSend, date, recipients, cc, onSendEmail]);

  const handlePreview = useCallback(async () => {
    if (!date) return;
    setIsPreviewing(true);
    setPreviewError('');
    try {
      const qs = buildQueryString({ atDate: formatDateForAPI(date) });
      await apiClient.download(
        `${API_ENDPOINTS.timesheets.payrollReport}${qs}`,
        `sao_ke_${formatDateForAPI(date)}.xlsx`,
      );
    } catch (err) {
      setPreviewError(getErrorMessage(err));
    } finally {
      setIsPreviewing(false);
    }
  }, [date]);

  const toErrMsg = toErr || (attempted && recipients.length === 0 ? 'Phải có ít nhất một người nhận' : '');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-full max-w-[calc(100vw-1rem)] max-h-[92dvh] gap-0 overflow-hidden p-0 sm:max-w-2xl" hideCloseButton>

        {/* ── header with actions inline ── */}
        <div className="bg-slate-900 px-4 pt-4 pb-3 text-white flex-shrink-0">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <div className="flex-1 min-w-0">
              <p className="text-sm font-semibold text-white leading-tight">Email Thanh Toán</p>
              <p className="text-xs text-slate-400 mt-0.5">Gửi sao kê qua email</p>
            </div>
            <div className="flex w-full items-center gap-2 sm:w-auto sm:shrink-0">
              <Button
                variant="ghost"
                size="sm"
                className="min-h-11 flex-1 bg-white/10 px-3 text-xs text-white shadow-none hover:bg-white/20 sm:flex-none"
                onClick={() => onOpenChange(false)}
                disabled={isLoading}
              >
                Hủy
              </Button>
              <Button
                size="sm"
                className="min-h-11 flex-1 gap-1.5 bg-white/10 px-3 text-xs text-white shadow-none hover:bg-white/20 sm:flex-none"
                onClick={handleSend}
                disabled={!canSend || isLoading}
              >
                {isLoading
                  ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Đang gửi...</>
                  : <><Mail className="h-3.5 w-3.5" />Gửi email</>}
              </Button>
              <DialogClose className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-white/10 outline-none transition-colors hover:bg-white/20" aria-label="Đóng">
                <X className="h-4 w-4 text-white" />
              </DialogClose>
            </div>
          </div>
        </div>

        {/* ── body: date | To | CC — row on desktop, stacked on mobile ── */}
        <div className="max-h-[calc(92dvh-92px)] overflow-y-auto">
        <div className="flex flex-col sm:flex-row sm:divide-x sm:divide-border">

          {/* date */}
          <div className="flex flex-col gap-2 px-4 py-3 sm:w-48 sm:shrink-0 border-b sm:border-b-0">
            <p className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              <CalendarIcon className="h-3 w-3" />Ngày báo cáo
            </p>
            <Popover open={calOpen} onOpenChange={setCalOpen}>
              <PopoverTrigger asChild>
                <Button variant="outline" className={cn('min-h-11 w-full justify-start px-3 text-left text-sm font-normal', !date && 'text-muted-foreground')}>
                  <CalendarIcon className="mr-1.5 h-3 w-3 text-muted-foreground" />
                  {date ? format(date, 'dd/MM/yyyy', { locale: vi }) : 'Chọn ngày'}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-auto p-0" align="start" sideOffset={4}>
                <Calendar mode="single" selected={date} onSelect={(d) => { if (d) { setDate(d); setPreviewError(''); } setCalOpen(false); }} initialFocus locale={vi} />
              </PopoverContent>
            </Popover>
            {date && <p className="text-xs text-muted-foreground">{format(date, 'EEEE', { locale: vi })}</p>}
            {date && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="min-h-11 w-full gap-1.5 text-sm"
                onClick={handlePreview}
                disabled={isPreviewing}
              >
                {isPreviewing
                  ? <Loader2 className="h-3 w-3 animate-spin" />
                  : <FileSpreadsheet className="h-3 w-3" />}
                {isPreviewing ? 'Đang tải...' : 'Xem sao kê'}
              </Button>
            )}
            {previewError && (
              <p className="flex items-center gap-1 text-xs text-destructive">
                <AlertCircle className="h-3 w-3 shrink-0" />{previewError}
              </p>
            )}
          </div>

          {/* To + CC — side by side on all sizes within this panel */}
          <div className="flex min-w-0 flex-1 flex-col divide-y divide-border sm:flex-row sm:divide-x sm:divide-y-0">
            <div className="flex-1 px-4 py-3 min-w-0">
              <EmailCol
                label="Người nhận"
                sublabel="(bắt buộc)"
                placeholder="email@domain.com"
                value={curTo}
                emails={recipients}
                error={toErrMsg}
                primary
                onChange={(v) => { setCurTo(v); setToErr(''); }}
                onAdd={() => add(curTo, recipients, setRecipients, setCurTo, setToErr)}
                onRemove={(i) => remove(i, recipients, setRecipients)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); add(curTo, recipients, setRecipients, setCurTo, setToErr); } }}
              />
            </div>
            <div className="flex-1 px-4 py-3 min-w-0">
              <EmailCol
                label="CC"
                sublabel="(tùy chọn)"
                placeholder="email@domain.com"
                value={curCc}
                emails={cc}
                error={ccErr}
                onChange={(v) => { setCurCc(v); setCcErr(''); }}
                onAdd={() => add(curCc, cc, setCc, setCurCc, setCcErr)}
                onRemove={(i) => remove(i, cc, setCc)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); add(curCc, cc, setCc, setCurCc, setCcErr); } }}
              />
            </div>
          </div>
        </div>
        </div>

      </DialogContent>
    </Dialog>
  );
});
