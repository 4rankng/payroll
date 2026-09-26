import { useCallback, useEffect, useState } from "react";
import {
  AlertCircle,
  BadgeCheck,
  CircleHelp,
  Loader2,
  SearchCheck,
  TriangleAlert,
  X,
} from "lucide-react";

import { EmployeeSingleSelector } from "@/components/ui/EmployeeSingleSelector";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SearchableSelect } from "@/components/ui/searchable-select";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  useEmployeeAccountLookup,
  useManualDisbursementBanks,
  useVerifyManualDisbursementAccount,
} from "@/hooks/api/useManualDisbursement";
import { cn } from "@/lib/utils";
import { createAppError } from "@/utils/error-handler";
import type { Employee } from "@/types/api/employee.types";
import type {
  EmployeeAccountLookupOutcome,
  EmployeeAccountLookupResponse,
  VerifyAccountResponse,
} from "@/types/api/manual-disbursement.types";

interface EmployeeAccountLookupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const OUTCOME_COPY: Record<
  EmployeeAccountLookupOutcome,
  {
    title: string;
    description: string;
    className: string;
    icon: typeof BadgeCheck;
  }
> = {
  valid: {
    title: "Tài khoản hợp lệ",
    description: "Nhà cung cấp đã xác nhận tài khoản có thể đối chiếu.",
    className: "border-success/30 bg-success/10 text-success",
    icon: BadgeCheck,
  },
  invalid: {
    title: "Tài khoản không hợp lệ",
    description:
      "Nhà cung cấp xác nhận tài khoản không hợp lệ hoặc không tồn tại.",
    className: "border-error/30 bg-error/10 text-error",
    icon: AlertCircle,
  },
  name_mismatch: {
    title: "Tên chủ tài khoản không khớp",
    description: "Tên đã lưu khác với tên được nhà cung cấp xác nhận.",
    className: "border-warning/40 bg-warning/10 text-warning-content",
    icon: TriangleAlert,
  },
  unverified: {
    title: "Chưa thể xác minh",
    description:
      "Nhà cung cấp chưa thể đưa ra kết quả xác nhận. Vui lòng thử lại sau.",
    className: "border-info/30 bg-info/10 text-info-content",
    icon: CircleHelp,
  },
};

type LookupMode = "employee" | "custom";

interface CustomAccountFields {
  bankCode: string;
  accountNumber: string;
  accountName: string;
}

const EMPTY_CUSTOM_ACCOUNT: CustomAccountFields = {
  bankCode: "",
  accountNumber: "",
  accountName: "",
};

function classifyCustomResult(
  result: VerifyAccountResponse,
): EmployeeAccountLookupOutcome {
  if (result.Valid) return "valid";
  if (result.RawErrorCode === "name_mismatch") return "name_mismatch";
  if (["12", "13", "19"].includes(result.RawErrorCode)) return "unverified";
  return "invalid";
}

function validateCustomAccount(fields: CustomAccountFields) {
  const bankCode = fields.bankCode.trim().toUpperCase();
  const accountNumber = fields.accountNumber.trim();
  const accountName = fields.accountName.trim();
  return {
    bankCode:
      bankCode && !/^[A-Z0-9]{8,11}$/.test(bankCode)
        ? "Mã SWIFT phải có 8–11 ký tự chữ hoặc số."
        : "",
    accountNumber:
      accountNumber && !/^\d{6,20}$/.test(accountNumber)
        ? "Số tài khoản phải có 6–20 chữ số."
        : "",
    accountName:
      accountName && (accountName.length < 3 || accountName.length > 100)
        ? "Tên chủ tài khoản phải có 3–100 ký tự."
        : "",
  };
}

function lookupErrorMessage(error: unknown): string {
  const { code, message } = createAppError(error);
  switch (code) {
    case "employee_bank_data_incomplete":
      return "Nhân viên chưa có đủ thông tin ngân hàng đã lưu để tra cứu.";
    case "employee_not_found":
      return "Không tìm thấy nhân viên. Vui lòng chọn lại từ danh sách.";
    case "account_verifier_unavailable":
      return "Dịch vụ xác minh tài khoản hiện chưa sẵn sàng. Vui lòng thử lại sau.";
    case "account_verification_provider_error":
      return "Nhà cung cấp không thể xác minh tài khoản lúc này. Vui lòng thử lại sau.";
    default:
      return message || "Không thể tra cứu tài khoản. Vui lòng thử lại sau.";
  }
}

function DetailRow({
  label,
  value,
  emphasize = false,
}: {
  label: string;
  value?: string | null;
  emphasize?: boolean;
}) {
  return (
    <div className="grid grid-cols-[minmax(6.5rem,0.8fr)_minmax(0,1.2fr)] gap-3 px-3 py-2.5 sm:grid-cols-[9rem_minmax(0,1fr)]">
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "min-w-0 break-words text-sm leading-snug text-foreground",
          emphasize && "font-semibold",
        )}
      >
        {value || "—"}
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
        <p className="text-sm font-semibold text-foreground">
          Thông tin ngân hàng đã lưu
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Dữ liệu chỉ đọc từ hồ sơ của {result.employee.fullname}.
        </p>
      </div>
      <dl className="divide-y rounded-lg border border-border bg-card">
        <DetailRow
          label="Ngân hàng"
          value={result.stored_bank.bank_name}
          emphasize
        />
        <DetailRow
          label="Mã ngân hàng"
          value={result.stored_bank.bank_code || result.stored_bank.swift_code}
        />
        <DetailRow
          label="Số tài khoản"
          value={result.stored_bank.account_number}
        />
        <DetailRow
          label="Tên đã lưu"
          value={result.stored_bank.account_name}
          emphasize
        />
      </dl>

      <section
        className={cn("rounded-lg border px-3 py-3", outcome.className)}
        aria-label="Kết quả xác minh"
      >
        <div className="flex items-start gap-2">
          <OutcomeIcon className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <div className="min-w-0">
            <p className="text-sm font-semibold">{outcome.title}</p>
            <p className="mt-0.5 text-xs leading-snug opacity-90">
              {outcome.description}
            </p>
          </div>
        </div>
      </section>

      <div className="border-t pt-2">
        <p className="text-sm font-semibold text-foreground">
          Kết quả từ nhà cung cấp
        </p>
        <dl className="mt-2 divide-y rounded-lg border border-border bg-card">
          <DetailRow
            label="Tên xác nhận"
            value={provider.AccountName}
            emphasize
          />
          {provider.RawErrorCode && (
            <DetailRow label="Mã phản hồi" value={provider.RawErrorCode} />
          )}
          {provider.RawMessage && (
            <DetailRow label="Phản hồi" value={provider.RawMessage} />
          )}
        </dl>
      </div>
    </div>
  );
}

function CustomLookupResult({
  fields,
  result,
}: {
  fields: CustomAccountFields;
  result: VerifyAccountResponse;
}) {
  const outcome = OUTCOME_COPY[classifyCustomResult(result)];
  const OutcomeIcon = outcome.icon;

  return (
    <div className="flex flex-col gap-3" aria-live="polite">
      <div className="border-b pb-2">
        <p className="text-sm font-semibold text-foreground">
          Thông tin đã nhập
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Dữ liệu chỉ dùng cho lần tra cứu này và không được lưu.
        </p>
      </div>
      <dl className="divide-y rounded-lg border border-border bg-card">
        <DetailRow label="Mã SWIFT" value={fields.bankCode} />
        <DetailRow label="Số tài khoản" value={fields.accountNumber} />
        <DetailRow label="Tên đã nhập" value={fields.accountName} emphasize />
      </dl>
      <section
        className={cn("rounded-lg border px-3 py-3", outcome.className)}
        aria-label="Kết quả xác minh"
      >
        <div className="flex items-start gap-2">
          <OutcomeIcon className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <div className="min-w-0">
            <p className="text-sm font-semibold">{outcome.title}</p>
            <p className="mt-0.5 text-xs leading-snug opacity-90">
              {outcome.description}
            </p>
          </div>
        </div>
      </section>
      <div className="border-t pt-2">
        <p className="text-sm font-semibold text-foreground">
          Kết quả từ nhà cung cấp
        </p>
        <dl className="mt-2 divide-y rounded-lg border border-border bg-card">
          <DetailRow
            label="Tên xác nhận"
            value={result.AccountName}
            emphasize
          />
          {result.RawErrorCode && (
            <DetailRow label="Mã phản hồi" value={result.RawErrorCode} />
          )}
          {result.RawMessage && (
            <DetailRow label="Phản hồi" value={result.RawMessage} />
          )}
        </dl>
      </div>
    </div>
  );
}

export function EmployeeAccountLookupDialog({
  open,
  onOpenChange,
}: EmployeeAccountLookupDialogProps) {
  const [mode, setMode] = useState<LookupMode>("employee");
  const [employee, setEmployee] = useState<Employee | null>(null);
  const [customFields, setCustomFields] =
    useState<CustomAccountFields>(EMPTY_CUSTOM_ACCOUNT);
  const lookupMutation = useEmployeeAccountLookup();
  const customLookupMutation = useVerifyManualDisbursementAccount();
  const banksQuery = useManualDisbursementBanks();
  const { reset: resetLookup } = lookupMutation;
  const { reset: resetCustomLookup } = customLookupMutation;

  const reset = useCallback(() => {
    setEmployee(null);
    setMode("employee");
    setCustomFields(EMPTY_CUSTOM_ACCOUNT);
    resetLookup();
    resetCustomLookup();
  }, [resetCustomLookup, resetLookup]);

  useEffect(() => {
    if (!open) reset();
  }, [open, reset]);

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) reset();
      onOpenChange(nextOpen);
    },
    [onOpenChange, reset],
  );

  const handleEmployeeSelect = useCallback(
    (nextEmployee: Employee) => {
      setEmployee(nextEmployee);
      resetLookup();
    },
    [resetLookup],
  );

  const handleModeChange = useCallback(
    (nextMode: LookupMode) => {
      setMode(nextMode);
      setEmployee(null);
      resetLookup();
      resetCustomLookup();
    },
    [resetCustomLookup, resetLookup],
  );

  const handleLookup = useCallback(() => {
    if (mode === "employee") {
      if (!employee) return;
      lookupMutation.mutate({ employee_id: employee.id });
      return;
    }
    customLookupMutation.mutate({
      bank_code: customFields.bankCode.trim().toUpperCase(),
      account_no: customFields.accountNumber.trim(),
      account_name: customFields.accountName.trim(),
      account_type: "0",
    });
  }, [customFields, customLookupMutation, employee, lookupMutation, mode]);

  const customErrors = validateCustomAccount(customFields);
  const customFormValid = Boolean(
    customFields.bankCode.trim() &&
    customFields.accountNumber.trim() &&
    customFields.accountName.trim() &&
    !customErrors.bankCode &&
    !customErrors.accountNumber &&
    !customErrors.accountName,
  );
  const pending =
    mode === "employee"
      ? lookupMutation.isPending
      : customLookupMutation.isPending;
  const lookupDisabled =
    pending || (mode === "employee" ? !employee : !customFormValid);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent
        contentPadding="none"
        hideCloseButton
        className="gap-0 p-0 sm:max-w-xl"
      >
        <header className="flex items-start gap-3 bg-emerald-950 px-4 py-4 text-white sm:px-5">
          <div className="min-w-0 flex-1">
            <DialogTitle>Tra cứu tài khoản</DialogTitle>
            <DialogDescription>
              Chỉ đọc thông tin đã lưu; thao tác này không tạo giao dịch.
            </DialogDescription>
          </div>
          <DialogClose
            aria-label="Đóng hộp thoại tra cứu tài khoản"
            className="flex size-11 shrink-0 items-center justify-center rounded-full bg-white/10 transition-colors hover:bg-emerald-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-200"
          >
            <X className="size-4" />
          </DialogClose>
        </header>
        <div className="min-h-0 overflow-y-auto px-4 py-4 sm:px-5">
          <div
            className="grid grid-cols-2 rounded-lg bg-muted p-1"
            role="tablist"
            aria-label="Nguồn thông tin tra cứu"
          >
            <button
              type="button"
              role="tab"
              aria-selected={mode === "employee"}
              onClick={() => handleModeChange("employee")}
              className={cn(
                "min-h-11 rounded-md px-3 text-sm font-semibold",
                mode === "employee"
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground",
              )}
            >
              Theo nhân viên
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={mode === "custom"}
              onClick={() => handleModeChange("custom")}
              className={cn(
                "min-h-11 rounded-md px-3 text-sm font-semibold",
                mode === "custom"
                  ? "bg-card text-foreground shadow-sm"
                  : "text-muted-foreground",
              )}
            >
              Nhập thủ công
            </button>
          </div>

          {mode === "employee" ? (
            <div className="mt-4 flex flex-col gap-2">
              <label
                htmlFor="employee-account-lookup-selector"
                className="text-sm font-medium text-foreground"
              >
                Nhân viên
              </label>
              <EmployeeSingleSelector
                id="employee-account-lookup-selector"
                value={employee}
                onSelect={handleEmployeeSelect}
                placeholder="Chọn nhân viên cần tra cứu"
                ariaLabel="Chọn nhân viên cần tra cứu"
                disabled={pending}
              />
              <p className="text-xs leading-snug text-muted-foreground">
                Hệ thống tự lấy ngân hàng, số tài khoản và tên chủ tài khoản từ
                hồ sơ đã lưu.
              </p>
            </div>
          ) : (
            <div className="mt-4 grid gap-3">
              <div className="grid gap-1.5 sm:grid-cols-[minmax(0,1fr)_9rem] sm:gap-3">
                <div className="grid gap-1.5">
                  <Label htmlFor="custom-bank-select">Ngân hàng</Label>
                  <SearchableSelect
                    triggerId="custom-bank-select"
                    value={customFields.bankCode}
                    onChange={(value) => {
                      setCustomFields((current) => ({
                        ...current,
                        bankCode: value,
                      }));
                      resetCustomLookup();
                    }}
                    options={[
                      ...new Map(
                        (banksQuery.data ?? [])
                          .filter((bank) => bank.swift_code)
                          .map((bank) => [bank.swift_code, bank]),
                      ).values(),
                    ].map((bank) => ({
                      value: bank.swift_code,
                      label: bank.bank_name,
                      searchText: bank.swift_code || undefined,
                    }))}
                    disabled={pending || banksQuery.isLoading}
                    placeholder={
                      banksQuery.isLoading ? "Đang tải..." : "Chọn ngân hàng"
                    }
                    searchPlaceholder="Tìm ngân hàng / mã SWIFT..."
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="custom-bank-code">Mã SWIFT</Label>
                  <Input
                    id="custom-bank-code"
                    value={customFields.bankCode}
                    onChange={(event) => {
                      setCustomFields((current) => ({
                        ...current,
                        bankCode: event.target.value.toUpperCase(),
                      }));
                      resetCustomLookup();
                    }}
                    maxLength={11}
                    autoComplete="off"
                    placeholder="VD: VCBVNVX"
                    disabled={pending}
                    aria-invalid={Boolean(customErrors.bankCode)}
                    aria-describedby={
                      customErrors.bankCode
                        ? "custom-bank-code-error"
                        : undefined
                    }
                    className={cn(
                      "min-h-11",
                      customErrors.bankCode && "border-destructive",
                    )}
                  />
                  {customErrors.bankCode && (
                    <p
                      id="custom-bank-code-error"
                      className="text-xs text-destructive"
                    >
                      {customErrors.bankCode}
                    </p>
                  )}
                </div>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label htmlFor="custom-account-number">Số tài khoản</Label>
                  <Input
                    id="custom-account-number"
                    value={customFields.accountNumber}
                    onChange={(event) => {
                      setCustomFields((current) => ({
                        ...current,
                        accountNumber: event.target.value.replace(/[^\d]/g, ""),
                      }));
                      resetCustomLookup();
                    }}
                    inputMode="numeric"
                    maxLength={20}
                    autoComplete="off"
                    disabled={pending}
                    aria-invalid={Boolean(customErrors.accountNumber)}
                    aria-describedby={
                      customErrors.accountNumber
                        ? "custom-account-number-error"
                        : undefined
                    }
                    className={cn(
                      "min-h-11",
                      customErrors.accountNumber && "border-destructive",
                    )}
                  />
                  {customErrors.accountNumber && (
                    <p
                      id="custom-account-number-error"
                      className="text-xs text-destructive"
                    >
                      {customErrors.accountNumber}
                    </p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="custom-account-name">Tên chủ tài khoản</Label>
                  <Input
                    id="custom-account-name"
                    value={customFields.accountName}
                    onChange={(event) => {
                      setCustomFields((current) => ({
                        ...current,
                        accountName: event.target.value,
                      }));
                      resetCustomLookup();
                    }}
                    maxLength={100}
                    autoComplete="off"
                    disabled={pending}
                    aria-invalid={Boolean(customErrors.accountName)}
                    aria-describedby={
                      customErrors.accountName
                        ? "custom-account-name-error"
                        : undefined
                    }
                    className={cn(
                      "min-h-11",
                      customErrors.accountName && "border-destructive",
                    )}
                  />
                  {customErrors.accountName && (
                    <p
                      id="custom-account-name-error"
                      className="text-xs text-destructive"
                    >
                      {customErrors.accountName}
                    </p>
                  )}
                </div>
              </div>
              <p className="text-xs leading-snug text-muted-foreground">
                Thông tin nhập thủ công chỉ được gửi tới OnePay để tra cứu và
                không được lưu vào hồ sơ nhân viên.
              </p>
            </div>
          )}

          {pending && (
            <div
              className="mt-4 flex items-center gap-2 border-y py-3 text-sm text-muted-foreground"
              role="status"
            >
              <Loader2 className="size-4 animate-spin" />
              Đang tra cứu với nhà cung cấp...
            </div>
          )}
          {mode === "employee" && lookupMutation.isError && (
            <div
              className="mt-4 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-3 text-sm text-destructive"
              role="alert"
            >
              {lookupErrorMessage(lookupMutation.error)}
            </div>
          )}
          {mode === "custom" && customLookupMutation.isError && (
            <div
              className="mt-4 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-3 text-sm text-destructive"
              role="alert"
            >
              {lookupErrorMessage(customLookupMutation.error)}
            </div>
          )}
          {mode === "employee" && lookupMutation.data && (
            <div className="mt-4">
              <LookupResult result={lookupMutation.data} />
            </div>
          )}
          {mode === "custom" && customLookupMutation.data && (
            <div className="mt-4">
              <CustomLookupResult
                fields={customFields}
                result={customLookupMutation.data}
              />
            </div>
          )}
        </div>
        <DialogFooter className="border-t bg-card px-4 py-3 sm:px-5">
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
            disabled={lookupDisabled}
            className="min-h-11 gap-2 sm:min-h-9"
          >
            {pending ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <SearchCheck className="size-4" />
            )}
            Tra cứu
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
