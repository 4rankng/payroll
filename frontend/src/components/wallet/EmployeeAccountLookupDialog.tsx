import { useCallback, useEffect, useState } from 'react';
import { AlertCircle, BadgeCheck, CircleHelp, Loader2, SearchCheck, TriangleAlert, X } from 'lucide-react';

import { EmployeeSingleSelector } from '@/components/ui/EmployeeSingleSelector';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
} from '@/components/ui/dialog';
import { useEmployeeAccountLookup } from '@/hooks/api/useManualDisbursement';
import { cn } from '@/lib/utils';
import { createAppError } from '@/utils/error-handler';
import type { Employee } from '@/types/api/employee.types';
import type { EmployeeAccountLookupOutcome, EmployeeAccountLookupResponse } from '@/types/api/manual-disbursement.types';

interface EmployeeAccountLookupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const OUTCOME_COPY: Record<EmployeeAccountLookupOutcome, { title: string; description: string; className: string; icon: typeof BadgeCheck }> = {
  valid: {
    title: 'Tài khoản hợp lệ',
    description: 'Nhà cung cấp đã xác nhận tài khoản có thể đối chiếu.',
    className: 'border-emerald-200 bg-emerald-50 text-emerald-900',
    icon: BadgeCheck,
  },
  invalid: {
    title: 'Tài khoản không hợp lệ',
    description: 'Nhà cung cấp xác nhận tài khoản không hợp lệ hoặc không tồn tại.',
    className: 'border-rose-200 bg-rose-50 text-rose-900',
    icon: AlertCircle,
  },
  name_mismatch: {
    title: 'Tên chủ tài khoản không khớp',
    description: 'Tên đã lưu khác với tên được nhà cung cấp xác nhận.',
    className: 'border-amber-200 bg-amber-50 text-amber-950',
    icon: TriangleAlert,
  },
  unverified: {
    title: 'Chưa thể xác minh',
    description: 'Nhà cung cấp chưa thể đưa ra kết quả xác nhận. Vui lòng thử lại sau.',
    className: 'border-sky-200 bg-sky-50 text-sky-950',
    icon: CircleHelp,
  },
};

function lookupErrorMessage(error: unknown): string {
  const { code, message } = createAppError(error);
  switch (code) {
    case 'employee_bank_data_incomplete':
      return 'Nhân viên chưa có đủ thông tin ngân hàng đã lưu để tra cứu.';
    case 'employee_not_found':
      return 'Không tìm thấy nhân viên. Vui lòng chọn lại từ danh sách.';
    case 'account_verifier_unavailable':
      return 'Dịch vụ xác minh tài khoản hiện chưa sẵn sàng. Vui lòng thử lại sau.';
    case 'account_verification_provider_error':
      return 'Nhà cung cấp không thể xác minh tài khoản lúc này. Vui lòng thử lại sau.';
    default:
      return message || 'Không thể tra cứu tài khoản. Vui lòng thử lại sau.';
  }
}

function DetailRow({ label, value, emphasize = false }: { label: string; value?: string | null; emphasize?: boolean }) {
  return (
    <div className="grid grid-cols-[minmax(6.5rem,0.8fr)_minmax(0,1.2fr)] gap-3 px-3 py-2.5 sm:grid-cols-[9rem_minmax(0,1fr)]">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className={cn('min-w-0 break-words text-sm leading-snug text-foreground', emphasize && 'font-semibold')}>
        {value || '—'}
      </dd>
    </div>
  );
}

function LookupResult({ result }: { result: EmployeeAccountLookupResponse }) {
  const outcome = OUTCOME_COPY[result.outcome];
  const OutcomeIcon = outcome.icon;
  const provider = result.provider_result;

  return (
    <div className="flex flex-col gap-3" aria-live="polite">
      <div className="border-b pb-2">
        <p className="text-sm font-semibold text-foreground">Thông tin ngân hàng đã lưu</p>
        <p className="mt-0.5 text-xs text-muted-foreground">Dữ liệu chỉ đọc từ hồ sơ của {result.employee.fullname}.</p>
      </div>
      <dl className="divide-y rounded-lg border border-border bg-background">
        <DetailRow label="Ngân hàng" value={result.stored_bank.bank_name} emphasize />
        <DetailRow label="Mã ngân hàng" value={result.stored_bank.bank_code || result.stored_bank.swift_code} />
        <DetailRow label="Số tài khoản" value={result.stored_bank.account_number} />
        <DetailRow label="Tên đã lưu" value={result.stored_bank.account_name} emphasize />
      </dl>

      <section className={cn('rounded-lg border px-3 py-3', outcome.className)} aria-label="Kết quả xác minh">
        <div className="flex items-start gap-2">
          <OutcomeIcon className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <div className="min-w-0">
            <p className="text-sm font-semibold">{outcome.title}</p>
            <p className="mt-0.5 text-xs leading-snug opacity-90">{outcome.description}</p>
          </div>
        </div>
      </section>

      <div className="border-t pt-2">
        <p className="text-sm font-semibold text-foreground">Kết quả từ nhà cung cấp</p>
        <dl className="mt-2 divide-y rounded-lg border border-border bg-background">
          <DetailRow label="Tên xác nhận" value={provider.AccountName} emphasize />
          {provider.RawErrorCode && <DetailRow label="Mã phản hồi" value={provider.RawErrorCode} />}
          {provider.RawMessage && <DetailRow label="Phản hồi" value={provider.RawMessage} />}
        </dl>
      </div>
    </div>
  );
}

export function EmployeeAccountLookupDialog({ open, onOpenChange }: EmployeeAccountLookupDialogProps) {
  const [employee, setEmployee] = useState<Employee | null>(null);
  const lookupMutation = useEmployeeAccountLookup();
  const { reset: resetLookup } = lookupMutation;

  const reset = useCallback(() => {
    setEmployee(null);
    resetLookup();
  }, [resetLookup]);

  useEffect(() => {
    if (!open) reset();
  }, [open, reset]);

  const handleOpenChange = useCallback((nextOpen: boolean) => {
    if (!nextOpen) reset();
    onOpenChange(nextOpen);
  }, [onOpenChange, reset]);

  const handleEmployeeSelect = useCallback((nextEmployee: Employee) => {
    setEmployee(nextEmployee);
    resetLookup();
  }, [resetLookup]);

  const handleLookup = useCallback(() => {
    if (!employee) return;
    lookupMutation.mutate({ employee_id: employee.id });
  }, [employee, lookupMutation]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        title="Tra cứu tài khoản"
        description="Chọn nhân viên để đối chiếu thông tin tài khoản đã lưu với nhà cung cấp."
        contentPadding="none"
        hideCloseButton
        className="gap-0 p-0 sm:max-w-xl"
      >
        <header className="flex items-start gap-3 bg-emerald-950 px-4 py-4 text-white sm:px-5">
          <div className="min-w-0 flex-1">
            <DialogTitle>Tra cứu tài khoản</DialogTitle>
            <DialogDescription>Chỉ đọc thông tin đã lưu; thao tác này không tạo giao dịch.</DialogDescription>
          </div>
          <DialogClose
            aria-label="Đóng hộp thoại tra cứu tài khoản"
            className="flex size-11 shrink-0 items-center justify-center rounded-full bg-white/10 transition-colors hover:bg-emerald-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-200"
          >
            <X className="size-4" />
          </DialogClose>
        </header>
        <div className="min-h-0 overflow-y-auto px-4 py-4 sm:px-5">
          <div className="flex flex-col gap-2">
            <label htmlFor="employee-account-lookup-selector" className="text-sm font-medium text-foreground">
              Nhân viên
            </label>
            <EmployeeSingleSelector
              id="employee-account-lookup-selector"
              value={employee}
              onSelect={handleEmployeeSelect}
              placeholder="Chọn nhân viên cần tra cứu"
              ariaLabel="Chọn nhân viên cần tra cứu"
              disabled={lookupMutation.isPending}
            />
            <p className="text-xs leading-snug text-muted-foreground">
              Hệ thống tự lấy ngân hàng, số tài khoản và tên chủ tài khoản từ hồ sơ đã lưu.
            </p>
          </div>

          {lookupMutation.isPending && (
            <div className="mt-4 flex items-center gap-2 border-y py-3 text-sm text-muted-foreground" role="status">
              <Loader2 className="size-4 animate-spin" />
              Đang tra cứu với nhà cung cấp...
            </div>
          )}
          {lookupMutation.isError && (
            <div className="mt-4 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-3 text-sm text-destructive" role="alert">
              {lookupErrorMessage(lookupMutation.error)}
            </div>
          )}
          {lookupMutation.data && <div className="mt-4"><LookupResult result={lookupMutation.data} /></div>}
        </div>
        <DialogFooter className="border-t bg-background px-4 py-3 sm:px-5">
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
            className="min-h-11 sm:min-h-9"
          >
            Đóng
          </Button>
          <Button
            type="button"
            onClick={handleLookup}
            disabled={!employee || lookupMutation.isPending}
            className="min-h-11 gap-2 sm:min-h-9"
          >
            {lookupMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : <SearchCheck className="size-4" />}
            Tra cứu
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
