import type { CSSProperties, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { EmployeePortalHeader } from "@/components/employees/EmployeePortalHeader";

const employeeMobileBackground: CSSProperties = {
  backgroundImage:
    "linear-gradient(180deg, rgba(224, 249, 255, 0.86) 0%, rgba(241, 250, 255, 0.92) 42%, rgba(248, 250, 252, 0.98) 100%), url('/employee-bg.avif')",
  backgroundSize: "cover",
  backgroundPosition: "center top",
  backgroundAttachment: "fixed",
  paddingBottom: "env(safe-area-inset-bottom)",
};

export const employeeCardShadow = {
  boxShadow: "0 12px 30px rgba(15, 23, 42, 0.08), 0 1px 0 rgba(255,255,255,0.75) inset",
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
          "employee-portal-card-stack mx-auto max-w-2xl space-y-4 p-4 pb-[calc(7rem+env(safe-area-inset-bottom))]",
          contentClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
