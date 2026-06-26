import { Building2, CreditCard, User, type LucideIcon } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";

interface EmployeeBankInfoCardProps {
  profile: EmployeeProfile;
  className?: string;
  style?: React.CSSProperties;
}

const bankFields: readonly {
  icon: LucideIcon;
  label: string;
  getValue: (p: EmployeeProfile) => string | undefined;
  mono: boolean;
}[] = [
  {
    icon: Building2,
    label: "Ngân hàng",
    getValue: (p) => p.bank?.branch_name,
    mono: false,
  },
  {
    icon: CreditCard,
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
          icon: User,
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
    <div className={className} style={style} role="region" aria-label="Thông tin ngân hàng">
      {hasBankInfo ? (
        <div>
          <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
            <div>
              <p className="text-[12px] font-semibold uppercase leading-4 tracking-wide text-slate-500">
                Tài khoản nhận tiền
              </p>
              <h2 className="text-[17px] font-bold leading-6 text-slate-950">
                Thông tin ngân hàng
              </h2>
            </div>
            <EmployeeIconFrame icon={CreditCard} />
          </div>
          <div className="divide-y divide-slate-100">
            {fields.map(({ icon: Icon, label, getValue, mono }) => {
              const value = getValue(profile);
              return (
                <div key={label}>
                  <div className="flex items-center gap-3 px-4 py-3.5">
                    <EmployeeIconFrame icon={Icon} size="row" tone="slate" />
                    <div className="min-w-0 flex-1">
                      <p className="text-[14px] font-medium leading-5 text-slate-500">
                        {label}
                      </p>
                      <p
                        className={`mt-0.5 text-[18px] font-bold leading-7 text-slate-950 ${
                          mono ? "break-all font-mono" : "whitespace-normal break-words"
                        }`}
                      >
                        {value || "—"}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      ) : (
        <div className="px-4 pb-3 text-center py-8">
          <div className="mx-auto mb-2.5 flex h-11 w-11 items-center justify-center rounded-2xl bg-gray-100 text-gray-300">
            <Building2 className="h-5 w-5" />
          </div>
          <p className="text-base font-semibold text-gray-500">
            Chưa có thông tin ngân hàng
          </p>
          <p className="mt-1 text-[15px] text-gray-400">
            Liên hệ quản lý để cập nhật
          </p>
        </div>
      )}
    </div>
  );
}
