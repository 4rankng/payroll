import { Building2, CreditCard, User } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";

interface EmployeeBankInfoCardProps {
  profile: EmployeeProfile;
  className?: string;
  style?: React.CSSProperties;
}

const bankFields: readonly {
  icon: typeof Building2;
  iconBg: string;
  iconColor: string;
  label: string;
  getValue: (p: EmployeeProfile) => string | undefined;
  mono: boolean;
}[] = [
  {
    icon: Building2,
    iconBg: "bg-sky-50",
    iconColor: "text-sky-600",
    label: "Ngân hàng",
    getValue: (p) => p.bank?.branch_name,
    mono: false,
  },
  {
    icon: CreditCard,
    iconBg: "bg-violet-50",
    iconColor: "text-violet-600",
    label: "Số tài khoản",
    getValue: (p) => p.bank_account_number,
    mono: true,
  },
  {
    icon: User,
    iconBg: "bg-emerald-50",
    iconColor: "text-emerald-600",
    label: "Chủ tài khoản",
    getValue: (p) => p.bank_account_name,
    mono: false,
  },
];

export function EmployeeBankInfoCard({
  profile,
  className,
  style,
}: EmployeeBankInfoCardProps) {
  if (!profile) return null;

  const hasBankInfo = !!(
    profile.bank_account_number || profile.bank
  );

  return (
    <div className={className} style={style} role="region" aria-label="Thông tin ngân hàng">
      {hasBankInfo ? (
        <div className="grid grid-cols-2 divide-x divide-gray-100">
          {bankFields.map(
            ({ icon: Icon, iconBg, iconColor, label, getValue, mono }, idx) => {
              const value = getValue(profile);
              return (
                <div
                  key={label}
                  className={idx === 0 ? "col-span-2 border-b border-gray-100" : ""}
                >
                  <div className="flex items-center gap-2.5 px-4 py-3">
                    <div className={`p-1.5 rounded-xl ${iconBg} shrink-0`}>
                      <Icon className={`h-3.5 w-3.5 ${iconColor}`} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-gray-400 font-medium">
                        {label}
                      </p>
                      <p
                        className={`text-sm font-bold text-gray-800 truncate ${mono ? "font-mono" : ""}`}
                      >
                        {value || "—"}
                      </p>
                    </div>
                  </div>
                </div>
              );
            }
          )}
        </div>
      ) : (
        <div className="px-4 pb-3 text-center py-8">
          <div className="w-10 h-10 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-2.5">
            <Building2 className="h-4 w-4 text-gray-300" />
          </div>
          <p className="text-sm font-medium text-gray-400">
            Chưa có thông tin ngân hàng
          </p>
          <p className="text-xs text-gray-300 mt-0.5">
            Liên hệ quản lý để cập nhật
          </p>
        </div>
      )}
    </div>
  );
}
