import type { CSSProperties, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";

const employeeMobileBackground: CSSProperties = {
  backgroundImage:
    "linear-gradient(180deg, #e8f8ef 0%, #f2f8f5 10rem, #f5f7fa 100%)",
  paddingBottom: "env(safe-area-inset-bottom)",
};

export const employeeCardShadow = {
  boxShadow: "0 8px 24px -20px rgba(15, 23, 42, 0.48)",
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
          "employee-portal-card-stack mx-auto max-w-lg space-y-3 px-3 py-3 min-[390px]:px-4 pb-[calc(5rem+env(safe-area-inset-bottom))]",
          contentClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
