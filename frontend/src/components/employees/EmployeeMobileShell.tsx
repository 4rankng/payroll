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
  /**
   * Full-bleed replacement for the default white app bar — e.g. an emerald
   * canopy that merges identity chrome with a hero metric. When omitted, the
   * shell falls back to the plain EmployeePortalHeader.
   */
  canopy?: ReactNode;
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
  canopy,
}: EmployeeMobileShellProps & { canopy?: ReactNode }) {
  const renderChrome = () => {
    if (chrome === "interactive") {
      if (canopy) return canopy;

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
          className="overflow-hidden rounded-b-[32px] bg-gradient-to-br from-emerald-500 to-emerald-700"
          style={{ paddingTop: "calc(env(safe-area-inset-top, 0px) + 1rem)" }}
          aria-label="Đang tải thông tin nhân viên"
        >
          <div className="mx-auto flex max-w-lg items-center justify-between px-5 pb-5">
            <div className="space-y-2.5">
              <Skeleton className="h-4 w-40 bg-white/20" />
              <Skeleton className="h-7 w-48 bg-white/25" />
            </div>
            <div className="flex gap-2.5">
              <Skeleton className="h-11 w-11 rounded-2xl bg-white/20" />
              <Skeleton className="h-11 w-11 rounded-2xl bg-white/20" />
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
          "employee-portal-card-stack mx-auto max-w-6xl space-y-5 px-3 py-5 pb-[calc(var(--employee-action-toolbar-block-size)+2.5rem+env(safe-area-inset-bottom))] sm:px-6 sm:py-6 lg:px-8 lg:py-8",
          contentClassName
        )}
      >
        {children}
      </main>
    </div>
  );
}
