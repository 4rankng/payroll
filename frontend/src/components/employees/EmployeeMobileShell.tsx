import type { CSSProperties, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";

const employeeMobileBackground: CSSProperties = {
  backgroundColor: "var(--employee-page)",
  paddingBottom: "env(safe-area-inset-bottom)",
};

export const employeeCardShadow = {
  boxShadow: "0 1px 2px rgba(16, 24, 40, 0.04)",
} as const;

interface EmployeeMobileShellProps {
  employeeName?: string;
  unreadCount?: number;
  onNotificationClick: () => void;
  onChangePassword: () => void;
  onLogout: () => void;
  children: ReactNode;
  className?: string;
  contentClassName?: string;
  style?: CSSProperties;
}

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
}: EmployeeMobileShellProps) {
  return (
    <div
      className={cn("employee-mobile-page min-h-[100dvh] overflow-x-hidden", className)}
      style={{ ...employeeMobileBackground, ...style }}
    >
      <EmployeePortalHeader
        employeeName={employeeName}
        unreadCount={unreadCount}
        onNotificationClick={onNotificationClick}
        onChangePassword={onChangePassword}
        onLogout={onLogout}
      />

      <main
        className={cn(
          "employee-portal-card-stack mx-auto max-w-6xl space-y-6 px-4 py-4 pb-[calc(2rem+env(safe-area-inset-bottom))] lg:px-6 lg:py-6",
          contentClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
