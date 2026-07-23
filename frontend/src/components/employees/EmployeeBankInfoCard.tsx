import { Landmark } from "lucide-react";
import type { EmployeeProfile } from "@/types/api/auth.types";
import { cn } from "@/lib/utils";

interface EmployeeBankInfoCardProps {
  profile: EmployeeProfile;
  className?: string;
  style?: React.CSSProperties;
}

export function EmployeeBankInfoCard({
  profile,
  className,
  style,
}: EmployeeBankInfoCardProps) {
  if (!profile) return null;

  const accountOwner = profile.bank_account_name?.trim();
  const hasBankInfo = !!(
    profile.bank_account_number ||
    profile.bank ||
    accountOwner
  );

  return (
    <div
      className={cn(
        "ct-card employee-bank-card group relative isolate min-h-[228px] overflow-hidden p-5 text-white sm:min-h-[250px] sm:p-6",
        className
      )}
      style={style}
      role="region"
      aria-labelledby="employee-bank-title"
    >
      <div className="pointer-events-none absolute -right-20 -top-24 h-56 w-56 rounded-full border border-white/10 shadow-[0_0_80px_rgba(109,230,165,0.16)]" aria-hidden="true" />
      <div className="pointer-events-none absolute -bottom-24 -left-20 h-48 w-48 rounded-full border border-amber-200/10" aria-hidden="true" />

      <div className="relative z-10 flex min-h-[188px] flex-col sm:min-h-[202px]">
        <div className="flex items-start justify-between gap-4">
          <div className="flex min-w-0 items-center gap-3">
            <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-white/15 bg-white/10 text-emerald-100 backdrop-blur">
              <Landmark className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <p className="employee-type-label-caps text-white/70">Cổng nhận lương</p>
              <h2 id="employee-bank-title" className="employee-type-card-title mt-0.5 text-white">
                Tài khoản nhận tiền
              </h2>
            </div>
          </div>
          <span className="grid h-9 w-12 shrink-0 grid-cols-3 gap-1 rounded-lg border border-amber-100/35 bg-amber-200/15 p-2 shadow-inner" aria-hidden="true">
            {Array.from({ length: 6 }, (_, index) => (
              <span key={index} className="rounded-sm bg-amber-100/55" />
            ))}
          </span>
        </div>

        {hasBankInfo ? (
          <dl className="mt-5 flex flex-1 flex-col">
            <div className="min-w-0">
              <dt className="employee-type-label-caps text-white/70">Số tài khoản</dt>
              <dd className="mt-1.5 min-w-0 break-all font-financial text-2xl font-semibold leading-tight tracking-[0.075em] text-white tabular-nums sm:text-[1.7rem]">
                {profile.bank_account_number || "—"}
              </dd>
            </div>

            <div className="mt-auto grid grid-cols-2 gap-4 border-t border-white/15 pt-4">
              <div className="min-w-0">
                <dt className="employee-type-label-caps text-white/90">Ngân hàng</dt>
                <dd className="employee-type-bank-value mt-1 break-words text-white">
                  {profile.bank?.branch_name || "—"}
                </dd>
              </div>
              <div className="min-w-0 text-right">
                <dt className="employee-type-label-caps text-white/90">Chủ tài khoản</dt>
                <dd className="employee-type-bank-value mt-1 break-words text-white">
                  {accountOwner || "—"}
                </dd>
              </div>
            </div>
          </dl>
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center px-4 text-center">
            <p className="employee-type-strong text-white">
              Chưa có thông tin ngân hàng
            </p>
            <p className="employee-type-body-sm mt-1 max-w-xs text-white/75">
              Liên hệ quản lý để cập nhật
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
