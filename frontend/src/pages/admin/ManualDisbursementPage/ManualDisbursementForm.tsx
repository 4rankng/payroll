import { useEffect, useMemo, useState } from "react";
import {
  CheckCircle2,
  Loader2,
  ShieldCheck,
  AlertCircle,
  ChevronsUpDown,
  Check,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { useDisbursementFeeSchedules } from "@/hooks/api/useDisbursementFeeSchedules";
import { useVerifyManualDisbursementAccount } from "@/hooks/api/useManualDisbursement";
import { vietnameseIncludes } from "@/utils/vietnameseNormalization";
import { stripVietnameseDiacritics } from "@/utils/vietnameseNormalization";

import {
  type BankOption,
  type ManualDisbursementFormState,
  type VerifiedAccount,
  formatAmount,
  isVerificationValid,
  parseAmountInput,
  validateForm,
} from "./helpers";

interface Props {
  banks: BankOption[];
  banksLoading?: boolean;
  state: ManualDisbursementFormState;
  onChange: (next: ManualDisbursementFormState) => void;
  onSubmit: () => void;
  verified: VerifiedAccount | null;
  onVerified: (v: VerifiedAccount | null) => void;
  disabled: boolean;
}

const ACCOUNT_TYPE_LABELS: Record<"0" | "1", string> = {
  "0": "Tài khoản ngân hàng",
  "1": "Thẻ ATM / Ví điện tử",
};

// ─── Searchable bank combobox ────────────────────────────────────────────────

interface BankComboboxProps {
  banks: BankOption[];
  loading: boolean;
  value: string;              // swift code value
  isManualEntry: boolean;     // controlled from parent
  onDropdownSelect: (swiftCode: string) => void;
  disabled: boolean;
  hasError: boolean;
}

function BankCombobox({ banks, loading, value, isManualEntry, onDropdownSelect, disabled, hasError }: BankComboboxProps) {
  const [open, setOpen] = useState(false);
  // Track which bank was selected by its unique code (not swift code, which can be shared).
  const [selectedCode, setSelectedCode] = useState("");

  // Reset internal code when parent signals manual typing or clears the value.
  useEffect(() => {
    if (isManualEntry || !value) setSelectedCode("");
  }, [isManualEntry, value]);

  // Show bank name only when selected via dropdown.
  const selected = isManualEntry ? null : banks.find((b) => b.code === selectedCode);

  const handleDropdownSelect = (bank: BankOption) => {
    setSelectedCode(bank.code);
    // Prefer swiftCode; fall back to bank code so the input is never blank.
    onDropdownSelect(bank.swiftCode || bank.code);
    setOpen(false);
  };

  if (loading) {
    return <Skeleton className="h-10 w-full" />;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn(
            "h-10 w-full justify-between font-normal",
            !selected && "text-muted-foreground",
            hasError && "border-destructive focus-visible:ring-destructive",
          )}
        >
          <span className="truncate">
            {selected ? selected.label : "Chọn ngân hàng..."}
          </span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[var(--radix-popover-trigger-width)] p-0"
        align="start"
        sideOffset={4}
      >
        <Command
          filter={(value, search) => {
            if (!search) return 1;
            return vietnameseIncludes(value, search) ? 1 : 0;
          }}
        >
          <CommandInput placeholder="Tìm ngân hàng..." />
          <CommandList className="max-h-60">
            <CommandEmpty className="py-6 text-center text-sm text-muted-foreground">
              Không tìm thấy ngân hàng.
            </CommandEmpty>
            <CommandGroup>
              {banks.map((b, idx) => (
                <CommandItem
                  key={`${b.code}-${b.swiftCode}-${idx}`}
                  value={`${b.label} ${b.code} ${b.swiftCode}`}
                  onSelect={() => handleDropdownSelect(b)}
                  className="cursor-pointer"
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4 shrink-0",
                      !isManualEntry && selectedCode === b.code ? "opacity-100 text-primary" : "opacity-0",
                    )}
                  />
                  <span className="truncate">{b.label}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

// ─── Main form ───────────────────────────────────────────────────────────────

export function ManualDisbursementForm(props: Props) {
  const { banks, banksLoading = false, state, onChange, onSubmit, verified, onVerified, disabled } = props;
  const [touched, setTouched] = useState(false);
  const [isManualBankEntry, setIsManualBankEntry] = useState(false);
  const errors = validateForm(state);

  // Reset manual-entry flag when the parent clears bankCode (form reset).
  useEffect(() => {
    if (!state.bankCode) setIsManualBankEntry(false);
  }, [state.bankCode]);
  const formIsValid = Object.keys(errors).length === 0;
  const verifyMutation = useVerifyManualDisbursementAccount();
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [nameMismatch, setNameMismatch] = useState<{ typed: string; verified: string } | null>(null);

  const feeQuery = useDisbursementFeeSchedules();
  const activeFee = useMemo(() => {
    const entries = feeQuery.data ?? [];
    return entries.find((e) => e.isCurrentlyActive)?.feeVnd ?? null;
  }, [feeQuery.data]);

  const verificationGood = isVerificationValid(verified, state);

  const setField = <K extends keyof ManualDisbursementFormState>(
    key: K,
    value: ManualDisbursementFormState[K],
  ) => {
    if (key === "bankCode" || key === "accountNo") {
      onVerified(null);
      setVerifyError(null);
      setNameMismatch(null);
    }
    onChange({ ...state, [key]: value });
  };

  const handleVerify = async () => {
    setTouched(true);
    setVerifyError(null);
    setNameMismatch(null);
    try {
      const result = await verifyMutation.mutateAsync({
        bank_code: state.bankCode,
        account_no: state.accountNo,
        account_type: state.accountType,
        account_name: state.accountName,
      });
      if (!result || !result.Valid) {
        // Backend signals name mismatch via RawErrorCode="name_mismatch" with
        // the bank-confirmed name in AccountName. Surface the friendly
        // side-by-side comparison instead of the raw "[code] message" dump.
        if (result?.RawErrorCode === "name_mismatch" && result.AccountName) {
          setNameMismatch({
            typed: state.accountName.trim(),
            verified: result.AccountName.trim(),
          });
          onVerified(null);
          return;
        }
        // For everything else, show the human-readable message only — the
        // technical error code is noise in the UI (logged for debugging).
        const msg = result?.RawMessage ?? "Không tìm thấy tài khoản";
        setVerifyError(msg);
        onVerified(null);
        return;
      }
      const verifiedName = result.AccountName?.trim() ?? "";
      onVerified({
        bankCode: state.bankCode,
        accountNo: state.accountNo,
        verifiedName,
        verifiedAt: Date.now(),
      });
      const typed = state.accountName.trim();
      if (verifiedName && typed && verifiedName.toUpperCase() !== typed.toUpperCase()) {
        setNameMismatch({ typed, verified: verifiedName });
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Xác thực thất bại";
      setVerifyError(message);
      onVerified(null);
    }
  };

  const handleAdoptVerifiedName = () => {
    if (nameMismatch) {
      onChange({ ...state, accountName: nameMismatch.verified });
      setNameMismatch(null);
    }
  };

  const canVerify =
    formIsValid &&
    !verifyMutation.isPending &&
    !banksLoading;
  const submitReady = formIsValid && verificationGood;

  const totalWithFee = useMemo(() => {
    const amt = Number(state.amount);
    if (!Number.isFinite(amt) || amt <= 0) return null;
    if (activeFee === null) return null;
    return amt + activeFee;
  }, [state.amount, activeFee]);

  return (
    <form
      className="space-y-3"
      onSubmit={(e) => {
        e.preventDefault();
        setTouched(true);
        if (submitReady && !disabled) onSubmit();
      }}
    >
      {/* ── Row 1: Bank combobox + Swift Code (stacked on mobile, two columns at sm+) ──── */}
      <div className="grid gap-x-2 gap-y-2 grid-cols-1 sm:grid-cols-[1fr_9rem]">
        <div className="grid gap-1.5">
          <Label>Ngân hàng</Label>
          <BankCombobox
            banks={banks}
            loading={banksLoading}
            value={state.bankCode}
            isManualEntry={isManualBankEntry}
            onDropdownSelect={(swiftCode) => {
              setIsManualBankEntry(false);
              setField("bankCode", swiftCode);
            }}
            disabled={disabled}
            hasError={touched && !!errors.bankCode}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="md-swift-code">Swift Code</Label>
          <Input
            id="md-swift-code"
            value={state.bankCode}
            onChange={(e) => {
              setIsManualBankEntry(true);
              setField("bankCode", e.target.value.toUpperCase());
            }}
            placeholder="VD: VPBVVNVX"
            disabled={disabled}
            className={cn(
              "h-10",
              touched && !!errors.bankCode && "border-destructive focus-visible:ring-destructive",
            )}
            maxLength={11}
            autoComplete="off"
          />
        </div>
        {touched && errors.bankCode && (
          <p className="sm:col-span-2 text-xs text-destructive">{errors.bankCode}</p>
        )}
      </div>

      {/* ── Row 2: Account number + Account name (2-col) ────────────────── */}
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="grid gap-1.5">
          <Label htmlFor="md-account-no">Số tài khoản</Label>
          <Input
            id="md-account-no"
            value={state.accountNo}
            inputMode="numeric"
            autoComplete="off"
            maxLength={20}
            onChange={(e) => setField("accountNo", e.target.value.replace(/[^\d]/g, ""))}
            disabled={disabled}
            className="h-10"
          />
          {touched && errors.accountNo && (
            <p className="text-xs text-destructive">{errors.accountNo}</p>
          )}
        </div>

        <div className="grid gap-1.5">
          <Label htmlFor="md-account-name">Tên chủ tài khoản</Label>
          <Input
            id="md-account-name"
            value={state.accountName}
            autoComplete="off"
            maxLength={100}
            onChange={(e) => setField("accountName", e.target.value)}
            disabled={disabled}
            className="h-10"
          />
          {touched && errors.accountName && (
            <p className="text-xs text-destructive">{errors.accountName}</p>
          )}
        </div>
      </div>

      {/* Name mismatch — side-by-side comparison with a one-click "use bank's name" affordance.
          Triggered both by Valid=true+name-differs (rare) and Valid=false+code=name_mismatch
          (the common case the backend returns). */}
      {nameMismatch && (
        <div className="rounded-xl border border-amber-200 bg-amber-50/60 p-3.5 space-y-3 shadow-sm">
          <div className="flex items-start gap-2.5">
            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-amber-100 border border-amber-200">
              <AlertCircle className="h-3.5 w-3.5 text-amber-700" />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold text-amber-900 leading-tight">
                Tên chủ tài khoản không khớp
              </p>
              <p className="text-[11.5px] text-amber-800/80 mt-0.5 leading-snug">
                Ngân hàng trả về một tên khác với tên bạn nhập. Kiểm tra lại hoặc dùng tên ngân hàng xác nhận.
              </p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <div className="rounded-lg border border-amber-200/70 bg-white px-3 py-2">
              <p className="text-[10px] font-bold uppercase tracking-wider text-amber-700/70 mb-0.5">
                Bạn nhập
              </p>
              <p className="font-mono text-xs font-semibold text-foreground break-words leading-snug">
                {nameMismatch.typed || "—"}
              </p>
            </div>
            <div className="rounded-lg border border-emerald-200 bg-white px-3 py-2">
              <p className="text-[10px] font-bold uppercase tracking-wider text-emerald-700/70 mb-0.5">
                Ngân hàng xác nhận
              </p>
              <p className="font-mono text-xs font-semibold text-foreground break-words leading-snug">
                {nameMismatch.verified || "—"}
              </p>
            </div>
          </div>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAdoptVerifiedName}
            className="h-8 text-xs gap-1.5 border-amber-300 hover:bg-amber-100 hover:border-amber-400 text-amber-900"
          >
            <Check className="h-3.5 w-3.5" />
            Dùng tên ngân hàng xác nhận
          </Button>
        </div>
      )}

      {/* ── Row 3: Account type (pill segmented control, full width) ───── */}
      <div className="grid gap-1.5">
        <Label>Loại tài khoản</Label>
        <div className="flex p-1 bg-slate-100 rounded-xl">
          {(["0", "1"] as const).map((opt) => (
            <button
              key={opt}
              type="button"
              disabled={disabled}
              onClick={() => setField("accountType", opt)}
              className={cn(
                "flex-1 flex items-center justify-center py-2 text-xs font-semibold rounded-lg transition-all",
                state.accountType === opt
                  ? "bg-white text-slate-900 shadow-sm"
                  : "text-slate-500 hover:text-slate-700",
                disabled && "cursor-not-allowed opacity-50",
              )}
            >
              {ACCOUNT_TYPE_LABELS[opt]}
            </button>
          ))}
        </div>
      </div>

      {/* ── Row 4: Amount + fee info ────────────────────────────────────── */}
      <div className="grid gap-1.5">
        <Label htmlFor="md-amount">Số tiền (VNĐ)</Label>
        <div className="relative">
          <Input
            id="md-amount"
            value={state.amount ? formatAmount(state.amount) : ""}
            inputMode="numeric"
            autoComplete="off"
            onChange={(e) => setField("amount", parseAmountInput(e.target.value))}
            disabled={disabled}
            className="h-10 pr-14 text-right font-semibold text-lg tabular-nums"
          />
          <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-muted-foreground">
            VNĐ
          </span>
        </div>
        {touched && errors.amount && (
          <p className="text-xs text-destructive">{errors.amount}</p>
        )}
        {activeFee !== null && state.amount && !errors.amount && (
          <p className="text-xs text-muted-foreground">
            Phí:{" "}
            <span className="font-semibold text-foreground">{formatAmount(activeFee)} đ</span>
            {totalWithFee !== null && (
              <>
                {" "}· Tổng:{" "}
                <span className="font-bold text-foreground">{formatAmount(totalWithFee)} đ</span>
              </>
            )}
          </p>
        )}
      </div>

      {/* ── Row 5: Description ──────────────────────────────────────────── */}
      <div className="grid gap-1.5">
        <Label htmlFor="md-description">Nội dung chuyển khoản</Label>
        <Textarea
          id="md-description"
          value={state.description}
          rows={2}
          maxLength={100}
          onChange={(e) => setField("description", stripVietnameseDiacritics(e.target.value).replace(/[^A-Za-z0-9 ]/g, ""))}
          disabled={disabled}
          className="resize-none min-h-0 h-[4.5rem]"
        />
        {touched && errors.description && (
          <p className="text-xs text-destructive">{errors.description}</p>
        )}
      </div>

      {/* ── Footer: verify status + actions (stacked on mobile) ─────────── */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-t pt-3 mt-1">
        <div className="min-w-0 sm:flex-1">
          {verificationGood ? (
            <div className="flex items-center gap-2 text-sm text-success">
              <CheckCircle2 className="h-4 w-4 shrink-0" />
              <span className="truncate">
                Tài khoản hợp lệ —{" "}
                <span className="font-medium">{verified?.verifiedName}</span>
              </span>
            </div>
          ) : verifyError ? (
            <div className="flex items-start gap-2 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{verifyError}</span>
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">
              Bạn cần xác thực tài khoản trước khi chuyển tiền.
            </p>
          )}
        </div>
        <div className="flex shrink-0 flex-row items-center justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={handleVerify}
            disabled={!canVerify || disabled}
            className="h-9 text-xs font-bold uppercase tracking-tight px-3"
          >
            {verifyMutation.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <ShieldCheck className="mr-2 h-4 w-4" />
            )}
            Xác thực
          </Button>
          <Button
            type="submit"
            disabled={!submitReady || disabled}
            className="h-9 bg-[#2a3b58] text-white hover:bg-[#1e293b] text-xs font-bold uppercase tracking-tight px-4"
          >
            {disabled && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Chuyển tiền
          </Button>
        </div>
      </div>
    </form>
  );
}
