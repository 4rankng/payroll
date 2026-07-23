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
        "ct-card employee-surface-card h-full min-h-[228px] overflow-hidden bg-[var(--employee-surface)] text-[var(--employee-text)]",
        className
      )}
      style={style}
      role="region"
      aria-labelledby="employee-bank-title"
    >
      <div className="flex items-center gap-3 border-b border-[var(--employee-border)] px-4 py-4 sm:px-5">
        <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]">
          <Landmark className="h-5 w-5" aria-hidden="true" />
        </span>
        <div className="min-w-0">
          <p className="employee-type-label-caps text-[var(--employee-text-secondary)]">
            Cổng nhận lương
          </p>
          <h2
            id="employee-bank-title"
            className="employee-type-card-title mt-0.5 text-[var(--employee-text)]"
          >
            Tài khoản nhận tiền
          </h2>
        </div>
      </div>

      {hasBankInfo ? (
        <dl>
          <div className="border-b border-[var(--employee-border)] bg-[var(--employee-accent-soft)] px-4 py-4 sm:px-5">
            <dt className="employee-type-label-caps text-[var(--employee-text-secondary)]">
              Số tài khoản
            </dt>
            <dd className="mt-1.5 min-w-0 break-all font-financial text-xl font-semibold leading-tight tracking-[0.035em] text-[var(--employee-text)] tabular-nums sm:text-2xl">
              {profile.bank_account_number || "—"}
            </dd>
          </div>

          <div className="grid grid-cols-2 divide-x divide-[var(--employee-border)]">
            <div className="min-w-0 px-4 py-4 sm:px-5">
              <dt className="employee-type-label-caps text-[var(--employee-text-secondary)]">
                Ngân hàng
              </dt>
              <dd className="employee-type-bank-value mt-1 break-words text-[var(--employee-text)]">
                {profile.bank?.branch_name || "—"}
              </dd>
            </div>
            <div className="min-w-0 px-4 py-4 text-right sm:px-5">
              <dt className="employee-type-label-caps text-[var(--employee-text-secondary)]">
                Chủ tài khoản
              </dt>
              <dd className="employee-type-bank-value mt-1 break-words text-[var(--employee-text)]">
                {accountOwner || "—"}
              </dd>
            </div>
          </div>
        </dl>
      ) : (
        <div className="flex min-h-[150px] flex-col items-center justify-center px-4 py-6 text-center">
          <span className="inline-flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--employee-surface-muted)] text-[var(--employee-text-secondary)]">
            <Landmark className="h-5 w-5" aria-hidden="true" />
          </span>
          <p className="employee-type-strong mt-3 text-[var(--employee-text)]">
            Chưa có thông tin ngân hàng
          </p>
          <p className="employee-type-body-sm mt-1 max-w-xs text-[var(--employee-text-secondary)]">
            Liên hệ quản lý để cập nhật
          </p>
        </div>
      )}
    </div>
  );
}
