import { useCallback, useEffect, useMemo, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { AlertCircle, ArrowLeft, Banknote, Landmark, Loader2 } from 'lucide-react';
import { useLoans, useRepayLoan } from '@/hooks/api/useLoans';
import { useBank } from '@/hooks/api/useBanks';
import { QueryKeys } from '@/lib/queryKeys';
import { formatVND } from '@/utils/loanHelpers';
import type { LenderWithSummary } from '@/types/api/loan.types';

interface SendMoneyToLenderModalProps {
  isOpen: boolean;
  onClose: () => void;
  lender: LenderWithSummary;
}

type FormData = {
  loan_id: string;
  amount: string;
  payment_date: string;
  payment_reference: string;
  notes: string;
};

const parseAmount = (value: string): number => Number(value.replace(/[^\d]/g, ''));

/**
 * Records a principal repayment ("send money") to a lender from the lender
 * drawer. The destination is the lender's saved bank account; the payment is
 * posted against one of their loans with outstanding principal via the loan
 * repay endpoint.
 */
export function SendMoneyToLenderModal({ isOpen, onClose, lender }: SendMoneyToLenderModalProps) {
  const [step, setStep] = useState<1 | 2>(1);
  const [form, setForm] = useState<FormData>({
    loan_id: '',
    amount: '',
    payment_date: new Date().toISOString().split('T')[0],
    payment_reference: '',
    notes: '',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

  const queryClient = useQueryClient();
  const repayLoan = useRepayLoan();

  const { data: loansResponse, isLoading: isLoansLoading } = useLoans(
    { lender_id: lender.id, pageSize: 100 },
    { enabled: isOpen }
  );

  const repayableLoans = useMemo(
    () => (loansResponse?.data ?? []).filter((loan) => loan.outstanding_principal > 0),
    [loansResponse]
  );

  const selectedLoan = useMemo(
    () => repayableLoans.find((loan) => loan.id === Number(form.loan_id)) ?? null,
    [repayableLoans, form.loan_id]
  );

  const { data: bank } = useBank(lender.bank_id || 0, {
    enabled: isOpen && !!lender.bank_id,
  });

  const hasBankInfo = !!(lender.bank_account_number && lender.bank_account_name);

  // Seed the form once the loan list has loaded
  useEffect(() => {
    if (isOpen && !isLoansLoading) {
      setStep(1);
      setForm({
        loan_id: repayableLoans.length > 0 ? repayableLoans[0].id.toString() : '',
        amount: repayableLoans.length > 0 ? repayableLoans[0].outstanding_principal.toString() : '',
        payment_date: new Date().toISOString().split('T')[0],
        payment_reference: '',
        notes: '',
      });
      setErrors({});
    }
  }, [isOpen, isLoansLoading, repayableLoans]);

  const updateField = useCallback((key: keyof FormData, value: string) => {
    setForm(prev => ({ ...prev, [key]: value }));
    if (errors[key]) {
      setErrors(prev => ({ ...prev, [key]: '' }));
    }
  }, [errors]);

  const handleLoanChange = useCallback((loanId: string) => {
    const loan = repayableLoans.find(l => l.id === Number(loanId));
    setForm(prev => ({
      ...prev,
      loan_id: loanId,
      amount: loan ? loan.outstanding_principal.toString() : prev.amount,
    }));
    setErrors(prev => ({ ...prev, loan_id: '', amount: '' }));
  }, [repayableLoans]);

  const handleAmountChange = useCallback((raw: string) => {
    updateField('amount', raw.replace(/[^\d]/g, ''));
  }, [updateField]);

  const validate = useCallback(() => {
    const nextErrors: Record<string, string> = {};

    if (!form.loan_id) {
      nextErrors.loan_id = 'Vui lòng chọn khoản vay';
    }
    const amount = parseAmount(form.amount);
    if (!form.amount.trim() || !Number.isFinite(amount) || amount <= 0) {
      nextErrors.amount = 'Số tiền phải lớn hơn 0';
    } else if (selectedLoan && amount > selectedLoan.outstanding_principal) {
      nextErrors.amount = `Số tiền vượt quá nợ gốc còn lại (${formatVND(selectedLoan.outstanding_principal)})`;
    }
    if (!form.payment_date) {
      nextErrors.payment_date = 'Vui lòng chọn ngày thanh toán';
    }

    setErrors(nextErrors);
    return Object.keys(nextErrors).length === 0;
  }, [form, selectedLoan]);

  const handleNext = useCallback(() => {
    if (validate()) {
      setStep(2);
    }
  }, [validate]);

  const handleSubmit = useCallback(async () => {
    if (!selectedLoan) return;

    try {
      await repayLoan.mutateAsync({
        id: selectedLoan.id,
        data: {
          amount: parseAmount(form.amount),
          payment_date: form.payment_date,
          payment_reference: form.payment_reference.trim() || undefined,
          notes: form.notes.trim() || undefined,
        },
      });
      // Refresh the lender drawer: outstanding principal and summary changed
      queryClient.invalidateQueries({ queryKey: QueryKeys.lenders.detail(lender.id) });
      onClose();
    } catch (error) {
      // Error handled by mutation hook
      setStep(1);
    }
  }, [selectedLoan, form, repayLoan, queryClient, lender.id, onClose]);

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-h-[92dvh] overflow-y-auto sm:max-w-[520px]">
        <DialogHeader>
          <DialogTitle>Gửi tiền cho chủ nợ</DialogTitle>
          <DialogDescription>
            {lender.name} — {step === 1 ? 'Nhập thông tin chuyển tiền' : 'Xác nhận thông tin chuyển tiền'}
          </DialogDescription>
        </DialogHeader>

        {/* Destination bank account */}
        <div className="rounded-lg border bg-muted/30 p-3 space-y-1.5">
          <p className="typography-label-medium flex items-center gap-1.5">
            <Landmark className="w-4 h-4" />
            Tài khoản nhận
          </p>
          {hasBankInfo ? (
            <div className="typography-body-medium space-y-0.5">
              <p>{bank?.branch_name || (lender.bank_id ? 'Ngân hàng' : 'Không rõ ngân hàng')}</p>
              <p className="font-mono">{lender.bank_account_number}</p>
              <p>{lender.bank_account_name}</p>
            </div>
          ) : (
            <p className="typography-body-small text-financial-negative flex items-start gap-1.5">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              Chưa lưu thông tin ngân hàng của chủ nợ. Hãy kiểm tra kỹ thông tin người nhận trước khi chuyển tiền.
            </p>
          )}
        </div>

        {isLoansLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        ) : repayableLoans.length === 0 ? (
          <p className="typography-body-medium text-muted-foreground py-4">
            Chủ nợ này không còn khoản vay chưa tất toán.
          </p>
        ) : step === 1 ? (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label className="typography-label-medium">Khoản vay *</Label>
              <Select value={form.loan_id} onValueChange={handleLoanChange}>
                <SelectTrigger className={errors.loan_id ? 'h-11 border-red-500' : 'h-11'}>
                  <SelectValue placeholder="Chọn khoản vay" />
                </SelectTrigger>
                <SelectContent>
                  {repayableLoans.map((loan) => (
                    <SelectItem key={loan.id} value={loan.id.toString()}>
                      {loan.loan_code} — còn lại {formatVND(loan.outstanding_principal)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {errors.loan_id && <p className="typography-body-small text-financial-negative">{errors.loan_id}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="send-amount" className="typography-label-medium">Số tiền (VNĐ) *</Label>
              <Input
                id="send-amount"
                inputMode="numeric"
                value={form.amount ? new Intl.NumberFormat('vi-VN').format(parseAmount(form.amount)) : ''}
                onChange={(e) => handleAmountChange(e.target.value)}
                className={errors.amount ? 'h-11 border-red-500' : 'h-11'}
                placeholder="0"
              />
              {errors.amount && <p className="typography-body-small text-financial-negative">{errors.amount}</p>}
              {selectedLoan && !errors.amount && (
                <p className="typography-body-small text-muted-foreground">
                  Nợ gốc còn lại: {formatVND(selectedLoan.outstanding_principal)}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="send-date" className="typography-label-medium">Ngày thanh toán *</Label>
              <Input
                id="send-date"
                type="date"
                value={form.payment_date}
                onChange={(e) => updateField('payment_date', e.target.value)}
                className={errors.payment_date ? 'h-11 border-red-500' : 'h-11'}
              />
              {errors.payment_date && <p className="typography-body-small text-financial-negative">{errors.payment_date}</p>}
            </div>

            <div className="space-y-2">
              <Label htmlFor="send-reference" className="typography-label-medium">Số tham chiếu</Label>
              <Input
                id="send-reference"
                value={form.payment_reference}
                onChange={(e) => updateField('payment_reference', e.target.value)}
                className="h-11"
                placeholder="Mã giao dịch ngân hàng (nếu có)"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="send-notes" className="typography-label-medium">Ghi chú</Label>
              <Textarea
                id="send-notes"
                value={form.notes}
                onChange={(e) => updateField('notes', e.target.value)}
                placeholder="Nội dung chuyển khoản (nếu có)"
                rows={3}
              />
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={onClose} className="min-h-11">
                Hủy
              </Button>
              <Button onClick={handleNext} className="min-h-11">
                <Banknote className="w-4 h-4 mr-2" />
                Tiếp tục
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="rounded-lg border p-3 space-y-2 typography-body-medium">
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Khoản vay</span>
                <span className="font-medium">{selectedLoan?.loan_code}</span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Số tiền</span>
                <span className="font-semibold text-financial-positive">{form.amount ? formatVND(parseAmount(form.amount)) : '-'}</span>
              </div>
              <div className="flex justify-between gap-4">
                <span className="text-muted-foreground">Ngày thanh toán</span>
                <span>{form.payment_date}</span>
              </div>
              {form.payment_reference.trim() && (
                <div className="flex justify-between gap-4">
                  <span className="text-muted-foreground">Số tham chiếu</span>
                  <span>{form.payment_reference.trim()}</span>
                </div>
              )}
              {form.notes.trim() && (
                <div className="flex justify-between gap-4">
                  <span className="text-muted-foreground">Ghi chú</span>
                  <span className="text-right">{form.notes.trim()}</span>
                </div>
              )}
              <div className="flex justify-between gap-4 border-t pt-2">
                <span className="text-muted-foreground">Nhận bởi</span>
                <span className="text-right">
                  {lender.name}
                  {hasBankInfo && (
                    <>
                      <br />
                      <span className="font-mono text-sm">{lender.bank_account_number}</span>
                    </>
                  )}
                </span>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setStep(1)} disabled={repayLoan.isPending} className="min-h-11">
                <ArrowLeft className="w-4 h-4 mr-2" />
                Quay lại
              </Button>
              <Button onClick={handleSubmit} disabled={repayLoan.isPending} className="min-h-11">
                {repayLoan.isPending ? (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                ) : (
                  <Banknote className="w-4 h-4 mr-2" />
                )}
                {repayLoan.isPending ? 'Đang xử lý...' : 'Xác nhận gửi tiền'}
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default SendMoneyToLenderModal;
