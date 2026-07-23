import type { CSSProperties, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";
import { Skeleton } from "@/components/ui/skeleton";

const employeeMobileBackground: CSSProperties = {
  paddingBottom: "env(safe-area-inset-bottom)",
};

const employeeShellHeaderStyle = {
  paddingTop: "calc(env(safe-area-inset-top, 0px) + 0.75rem)",
} as const;

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
  const renderChrome = () => {
    if (chrome === "interactive") {
      return (
        <EmployeePortalHeader
          employeeName={employeeName}
          unreadCount={unreadCount}
          onNotificationClick={onNotificationClick}
          onChangePassword={onChangePassword}
          onLogout={onLogout}
        />
      );
    }

    if (chrome === "skeleton") {
      return (
        <div
          className="border-b border-[var(--employee-border)] bg-[var(--employee-surface)]"
          aria-label="Đang tải thông tin nhân viên"
        >
          <div
            className="mx-auto flex max-w-lg items-center justify-between px-4 pb-3"
            style={employeeShellHeaderStyle}
          >
            <div className="space-y-2">
              <Skeleton className="h-7 w-44" />
              <Skeleton className="h-4 w-28" />
            </div>
            <div className="flex gap-2">
              <Skeleton className="h-11 w-11 rounded-xl" />
              <Skeleton className="h-11 w-11 rounded-xl" />
            </div>
          </div>
        </div>
      );
    }

    return null;
  };

  return (
    <div
      className={cn("employee-mobile-page min-h-[100dvh] overflow-x-hidden bg-[var(--employee-page)] text-[var(--employee-text)]", className)}
      style={{ ...employeeMobileBackground, ...style }}
      data-employee-ui=""
      data-theme="employee"
      data-has-action-toolbar={hasActionToolbar}
    >
      {renderChrome()}

      <main
        className={cn(
          "employee-portal-card-stack mx-auto max-w-6xl space-y-5 px-3.5 py-5 pb-[calc(var(--employee-action-toolbar-block-size)+2.5rem+env(safe-area-inset-bottom))] sm:px-5 sm:py-6 lg:px-8 lg:py-8",
          contentClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
