import { useId, useState } from "react";
import { Building2, ChevronDown, CreditCard } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";
import { cn } from "@/lib/utils";

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
  const [isOpen, setIsOpen] = useState(false);
  const panelId = useId();

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
      className={className ?? "overflow-hidden rounded-2xl bg-white"}
      style={style}
      role="region"
      aria-label="Thông tin ngân hàng"
    >
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((current) => !current)}
        className="flex min-h-14 w-full items-center justify-between gap-3 px-4 py-3 text-left transition-colors hover:bg-slate-50/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500"
      >
        <span className="flex min-w-0 items-center gap-2.5">
          <EmployeeIconFrame icon={CreditCard} size="row" />
          <span className="min-w-0">
            <span className="employee-type-card-title block text-slate-950">
              Thông tin ngân hàng
            </span>
            <span className="employee-type-body-sm mt-0.5 block truncate text-slate-500">
              {hasBankInfo
                ? profile.bank?.branch_name || "Tài khoản nhận tiền đã cập nhật"
                : "Chưa cập nhật tài khoản nhận tiền"}
            </span>
          </span>
        </span>
        <ChevronDown
          className={cn(
            "h-5 w-5 shrink-0 text-slate-400 transition-transform duration-200",
            isOpen && "rotate-180"
          )}
        />
      </button>

      {isOpen && (
        <div id={panelId} className="border-t border-slate-100">
          {hasBankInfo ? (
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
          ) : (
          <div className="px-4 py-7 text-center">
            <div className="mx-auto mb-2.5 flex h-11 w-11 items-center justify-center rounded-2xl bg-slate-100 text-slate-500">
              <Building2 className="h-5 w-5" />
            </div>
            <p className="employee-type-card-title text-slate-700">
              Chưa có thông tin ngân hàng
            </p>
            <p className="employee-type-body mt-1 text-slate-600">
              Liên hệ quản lý để cập nhật
            </p>
          </div>
          )}
        </div>
      )}
    </div>
  );
}
