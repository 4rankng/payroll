import type { CSSProperties, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";
import { Skeleton } from "@/components/ui/skeleton";

const employeeMobileBackground: CSSProperties = {
  backgroundColor: "var(--employee-page)",
  paddingBottom: "env(safe-area-inset-bottom)",
};

export const employeeCardShadow = {
  boxShadow: "0 1px 2px rgba(16, 24, 40, 0.04)",
} as const;

interface EmployeeMobileShellBaseProps {
  employeeName?: string;
  unreadCount?: number;
  children: ReactNode;
  className?: string;
  contentClassName?: string;
  style?: CSSProperties;
  hasActionToolbar?: boolean;
}

interface EmployeeMobileShellInteractiveProps extends EmployeeMobileShellBaseProps {
  chrome?: "interactive";
  onNotificationClick: () => void;
  onChangePassword: () => void;
  onLogout: () => void;
}

interface EmployeeMobileShellStaticProps extends EmployeeMobileShellBaseProps {
  chrome: "skeleton" | "error";
  onNotificationClick?: never;
  onChangePassword?: never;
  onLogout?: never;
}

type EmployeeMobileShellProps =
  | EmployeeMobileShellInteractiveProps
  | EmployeeMobileShellStaticProps;

export function EmployeeMobileShell({
  employeeName,
  unreadCount,
  onNotificationClick,
  onChangePassword,
  onLogout,
  children,
  className,
  contentClassName,
  style,
  hasActionToolbar = false,
  chrome = "interactive",
}: EmployeeMobileShellProps) {
  return (
    <div
      className={cn("employee-mobile-page min-h-[100dvh] overflow-x-hidden", className)}
      style={{ ...employeeMobileBackground, ...style }}
      data-employee-ui="true"
      data-has-action-toolbar={hasActionToolbar}
    >
      {chrome === "interactive" ? (
        <EmployeePortalHeader
          employeeName={employeeName}
          unreadCount={unreadCount}
          onNotificationClick={onNotificationClick}
          onChangePassword={onChangePassword}
          onLogout={onLogout}
        />
      ) : chrome === "skeleton" ? (
        <div className="border-b border-[var(--employee-border)] bg-[var(--employee-surface)]" aria-label="Đang tải thông tin nhân viên">
          <div className="mx-auto flex max-w-lg items-center justify-between px-4 pb-3" style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}>
            <div className="space-y-2"><Skeleton className="h-7 w-44" /><Skeleton className="h-4 w-28" /></div>
            <div className="flex gap-2"><Skeleton className="h-11 w-11 rounded-xl" /><Skeleton className="h-11 w-11 rounded-xl" /></div>
          </div>
        </div>
      ) : (
        <div className="border-b border-[var(--employee-border)] bg-[var(--employee-surface)]">
          <div className="mx-auto max-w-lg px-4 pb-3" style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)" }}>
            <p className="employee-type-header-name text-[var(--employee-text)]">Cổng nhân viên</p>
          </div>
        </div>
      )}

      <main
        className={cn(
          "employee-portal-card-stack mx-auto max-w-6xl space-y-6 px-4 py-4 pb-[calc(var(--employee-action-toolbar-block-size)+2rem+env(safe-area-inset-bottom))] lg:px-6 lg:py-6",
          contentClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
