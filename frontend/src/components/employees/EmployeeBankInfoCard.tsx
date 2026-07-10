import { Building2, CreditCard } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";

interface EmployeeBankInfoCardProps {
  profile: EmployeeProfile;
  className?: string;
  style?: React.CSSProperties;
}

const bankFields: readonly {
  label: string;
  getValue: (p: EmployeeProfile) => string | undefined;
  mono: boolean;
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
        },
      ]
    : bankFields;

  const hasBankInfo = !!(
    profile.bank_account_number ||
    profile.bank ||
    accountOwner
  );

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
          <span className="employee-type-body-sm mt-0.5 block text-[#667085]">
            {hasBankInfo
              ? "Tài khoản sẽ nhận tiền ứng lương"
              : "Chưa cập nhật tài khoản nhận tiền"}
          </span>
        </span>
      </div>

      <div className="overflow-hidden rounded-xl border border-[#E4E7EC] bg-white" style={style}>
        {hasBankInfo ? (
          <dl className="divide-y divide-[#EAECF0]">
            {fields.map(({ label, getValue, mono }) => {
              const value = getValue(profile);
              return (
                <div key={label} className="grid grid-cols-[minmax(92px,0.75fr)_minmax(0,1.25fr)] items-start gap-4 px-4 py-3">
                  <dt className="employee-type-label text-[#667085]">{label}</dt>
                  <dd
                    className={`employee-type-bank-value min-w-0 text-right text-[#101828] ${
                      mono ? "break-all font-mono tabular-nums" : "break-words"
                    }`}
                  >
                    {value || "—"}
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
