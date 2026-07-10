import { Building2, Copy, CreditCard } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";
import { toast } from "@/components/ui/sonner";

interface EmployeeBankInfoCardProps {
  profile: EmployeeProfile;
  className?: string;
  style?: React.CSSProperties;
}

const bankFields: readonly {
  label: string;
  getValue: (p: EmployeeProfile) => string | undefined;
  mono: boolean;
  copyable?: boolean;
}[] = [
  {
    label: "Ngân hàng",
    getValue: (p) => p.bank?.branch_name,
    mono: false,
  },
  {
    label: "Số tài khoản",
    getValue: (p) => p.bank_account_number,
    mono: true,
    copyable: true,
  },
];

const normalizeName = (value?: string) =>
  value?.trim().replace(/\s+/g, " ").toLocaleLowerCase("vi-VN") ?? "";

export function EmployeeBankInfoCard({
  profile,
  className,
  style,
}: EmployeeBankInfoCardProps) {
  if (!profile) return null;

  const accountOwner = profile.bank_account_name?.trim();
  const shouldShowAccountOwner =
    !!accountOwner && normalizeName(accountOwner) !== normalizeName(profile.fullname);
  const fields = shouldShowAccountOwner
    ? [
        ...bankFields,
        {
          label: "Chủ tài khoản",
          getValue: (p: EmployeeProfile) => p.bank_account_name,
          mono: false,
          copyable: false,
        },
      ]
    : bankFields;

  const hasBankInfo = !!(
    profile.bank_account_number ||
    profile.bank ||
    accountOwner
  );

  const handleCopyAccountNumber = async (accountNumber: string) => {
    try {
      await navigator.clipboard.writeText(accountNumber);
      toast({ title: "Đã sao chép số tài khoản" });
    } catch {
      toast({
        title: "Chưa thể sao chép số tài khoản",
        description: "Vui lòng thử lại.",
        variant: "destructive",
      });
    }
  };

  return (
    <div
      className={className}
      role="region"
      aria-labelledby="employee-bank-title"
    >
      <div className="mb-3 flex items-center gap-2.5 px-0.5">
        <EmployeeIconFrame icon={CreditCard} size="row" />
        <span className="min-w-0">
          <span id="employee-bank-title" className="employee-type-section-title block text-[#101828]">
            Thông tin ngân hàng
          </span>
          <span className="employee-type-body mt-0.5 block text-[var(--employee-text-secondary)]">
            {hasBankInfo
              ? "Tài khoản sẽ nhận tiền ứng lương"
              : "Chưa cập nhật tài khoản nhận tiền"}
          </span>
        </span>
      </div>

      <div className="overflow-hidden rounded-xl border border-[var(--employee-border)] bg-white shadow-[var(--employee-shadow)]" style={style}>
        {hasBankInfo ? (
          <dl className="divide-y divide-[#EAECF0]">
            {fields.map(({ label, getValue, mono, copyable }) => {
              const value = getValue(profile);
              return (
                <div key={label} className="grid grid-cols-[minmax(84px,0.7fr)_minmax(0,1.3fr)] items-center gap-3 px-4 py-3 lg:grid-cols-1 lg:items-start lg:gap-1">
                  <dt className="employee-type-bank-label text-[var(--employee-text-secondary)]">{label}</dt>
                  <dd
                    className={`employee-type-bank-value min-w-0 text-right text-[var(--employee-text)] lg:text-left ${
                      mono ? "break-all font-mono tabular-nums" : "break-words"
                    }`}
                  >
                    {copyable && value ? (
                      <span className="flex min-w-0 items-center justify-end gap-1.5 lg:justify-start">
                        <span className="min-w-0 break-all">{value}</span>
                        <button
                          type="button"
                          onClick={() => handleCopyAccountNumber(value)}
                          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-[10px] text-[var(--employee-text-secondary)] transition-colors duration-200 hover:bg-[#F2F4F7] active:bg-[#EAECF0] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent)]"
                          aria-label="Sao chép số tài khoản"
                          title="Sao chép số tài khoản"
                        >
                          <Copy className="h-4 w-4" aria-hidden="true" />
                        </button>
                      </span>
                    ) : (
                      value || "—"
                    )}
                  </dd>
                </div>
              );
            })}
          </dl>
        ) : (
          <div className="px-4 py-6 text-center">
            <div className="mx-auto mb-2.5 flex h-11 w-11 items-center justify-center rounded-xl bg-[#F2F4F7] text-[#667085]">
              <Building2 className="h-5 w-5" />
            </div>
            <p className="employee-type-strong text-[#344054]">
              Chưa có thông tin ngân hàng
            </p>
            <p className="employee-type-body-sm mt-1 text-[#667085]">
              Liên hệ quản lý để cập nhật
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
