import type { EmployeeProfile } from "@/types/api/auth.types";
import { cn } from "@/lib/utils";
import { hasEmployeeBankInfo } from "@/utils/employeePortal/mobileHome";

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
  const hasBankInfo = hasEmployeeBankInfo(profile);

  return (
    <div
      className={cn(
        "employee-surface-card group relative isolate min-h-[228px] overflow-hidden p-5 text-[var(--employee-text)] sm:min-h-[250px] sm:p-6",
        className
      )}
      style={style}
      role="region"
      aria-labelledby="employee-bank-title"
    >
      <img
        src="/employee/bank-security-pattern-v2.webp"
        alt=""
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 -z-20 h-full w-full object-cover object-right opacity-90 saturate-75 transition-transform duration-700 motion-safe:group-hover:scale-[1.025]"
      />
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-r from-[var(--employee-surface)]/45 via-[var(--employee-surface)]/10 to-transparent" />

      <div className="flex min-h-[188px] flex-col sm:min-h-[202px]">
        <h2 id="employee-bank-title" className="employee-type-card-title text-[var(--employee-text)]">
          Tài khoản nhận tiền
        </h2>

        {hasBankInfo ? (
          <div className="mt-5 flex flex-1 flex-col sm:mt-6">
            <dl className="min-w-0">
              <dt className="employee-type-label-caps text-[var(--employee-text)]">Số tài khoản</dt>
              <dd className="employee-type-bank-value mt-1.5 min-w-0 break-all text-[var(--employee-text)] tabular-nums">
                {profile.bank_account_number || "—"}
              </dd>
            </dl>

            <div className="mt-auto grid grid-cols-2 gap-4 border-t border-[var(--employee-border)] pt-4">
              <dl className="min-w-0">
                <dt className="employee-type-label-caps text-[var(--employee-text)]">Ngân hàng</dt>
                <dd className="employee-type-bank-value mt-1 break-words text-[var(--employee-text)]">
                  {profile.bank?.branch_name || "—"}
                </dd>
              </dl>
              <dl className="min-w-0 text-right">
                <dt className="employee-type-label-caps text-[var(--employee-text)]">Chủ tài khoản</dt>
                <dd className="employee-type-bank-value mt-1 break-words text-[var(--employee-text)]">
                  {accountOwner || "—"}
                </dd>
              </dl>
            </div>
          </div>
        ) : (
          <div className="flex flex-1 flex-col items-center justify-center px-4 text-center">
            <p className="employee-type-strong text-[var(--employee-text)]">
              Chưa có thông tin ngân hàng
            </p>
            <p className="employee-type-body-sm mt-1 max-w-xs text-[var(--employee-text-secondary)]">
              Liên hệ quản lý để cập nhật
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
