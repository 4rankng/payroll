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
    <div className={className} style={style} role="region" aria-label="Thông tin ngân hàng">
      {hasBankInfo ? (
        <div>
          <div className="flex items-center justify-between border-b border-slate-100 px-4 py-3">
            <div>
              <p className="employee-type-label-caps text-slate-500">
                Tài khoản nhận tiền
              </p>
              <h2 className="employee-type-card-title text-slate-950">
                Thông tin ngân hàng
              </h2>
            </div>
            <EmployeeIconFrame icon={CreditCard} />
          </div>
          <div className="grid grid-cols-2 border-slate-100">
            {fields.map(({ label, getValue, mono }, index) => {
              const value = getValue(profile);
              const isAccountOwner = label === "Chủ tài khoản";
              return (
                <div
                  key={label}
                  className={
                    isAccountOwner
                      ? "col-span-2 border-t border-slate-100"
                      : index === 0
                        ? "border-r border-slate-100"
                        : ""
                  }
                >
                  <div className="min-w-0 px-4 py-3.5">
                    <div className="min-w-0">
                      <p className="employee-type-label truncate text-slate-500">
                        {label}
                      </p>
                      <p
                        className={`employee-type-bank-value mt-1 text-slate-950 ${
                          mono ? "whitespace-nowrap font-mono" : "whitespace-normal break-words"
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
          <p className="employee-type-card-title text-gray-500">
            Chưa có thông tin ngân hàng
          </p>
          <p className="employee-type-body mt-1 text-gray-400">
            Liên hệ quản lý để cập nhật
          </p>
        </div>
      )}
    </div>
  );
}
