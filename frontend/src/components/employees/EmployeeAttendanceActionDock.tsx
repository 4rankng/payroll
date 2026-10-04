import {
  AlertCircle,
  Briefcase01,
  CheckVerified01,
  LogOut02,
  RefreshCw05,
  Wallet01,
} from "@untitledui/icons";
import { useLayoutEffect, useRef } from "react";

export type AttendanceDockAction =
  | "check_in"
  | "check_out"
  | "loading"
  | "completed"
  | "attention";

interface EmployeeAttendanceActionDockProps {
  action: AttendanceDockAction;
  actionLabel: string;
  actionDisabled?: boolean;
  onAdvanceRequest: () => void;
  onAttendanceAction?: () => void;
}

const ACTION_ICONS = {
  check_in: Briefcase01,
  check_out: LogOut02,
  loading: RefreshCw05,
  completed: CheckVerified01,
  attention: AlertCircle,
} satisfies Record<AttendanceDockAction, typeof Briefcase01>;

export function EmployeeAttendanceActionDock({
  action,
  actionLabel,
  actionDisabled = false,
  onAdvanceRequest,
  onAttendanceAction,
}: EmployeeAttendanceActionDockProps) {
  const ActionIcon = ACTION_ICONS[action];
  const disabled = actionDisabled || !onAttendanceAction;
  const dockRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const dock = dockRef.current;
    const employeePage = dock?.closest<HTMLElement>(".employee-mobile-page");
    if (!dock || !employeePage) return;

    const updateReservedSpace = () => {
      const measuredHeight = Math.ceil(dock.getBoundingClientRect().height);
      employeePage.style.setProperty(
        "--employee-action-toolbar-block-size",
        measuredHeight > 0 ? `${measuredHeight}px` : "5.5rem"
      );
    };
    updateReservedSpace();
    const observer = typeof ResizeObserver === "undefined" ? null : new ResizeObserver(updateReservedSpace);
    observer?.observe(dock);
    return () => {
      observer?.disconnect();
      employeePage.style.removeProperty("--employee-action-toolbar-block-size");
    };
  }, []);

  return (
    <div
      ref={dockRef}
      className="employee-attendance-action-dock fixed inset-x-0 bottom-0 z-40 min-h-[calc(4.25rem+env(safe-area-inset-bottom))] border-t border-[var(--employee-border)] bg-[var(--employee-surface)]"
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      role="toolbar"
      aria-label="Hành động nhân viên"
    >
      <div className="mx-auto grid min-h-[4.25rem] max-w-lg grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-stretch gap-2 px-3 py-2">
        <button
          type="button"
          className="employee-type-action inline-flex h-auto min-h-11 min-w-0 items-center justify-center gap-1.5 whitespace-normal rounded-[var(--employee-radius-control)] border border-[var(--employee-accent)] px-3 py-2 text-center text-sm font-semibold leading-tight text-[var(--employee-accent)] transition-colors duration-100 ease-linear hover:bg-[var(--employee-accent-soft)] focus-visible:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--employee-focus-ring)]"
          onClick={onAdvanceRequest}
        >
          <Wallet01 className="h-4 w-4" aria-hidden="true" />
          Ứng lương
        </button>

        <button
          type="button"
          className="employee-type-action inline-flex h-auto min-h-11 min-w-0 items-center justify-center gap-1.5 whitespace-normal rounded-[var(--employee-radius-control)] bg-[var(--employee-accent)] px-3 py-2 text-center text-sm font-semibold leading-tight text-white transition-colors duration-100 ease-linear hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--employee-focus-ring)] disabled:pointer-events-none disabled:opacity-60"
          disabled={disabled}
          onClick={onAttendanceAction}
        >
          <ActionIcon
            className={`h-4 w-4 ${action === "loading" ? "animate-spin motion-reduce:animate-none" : ""}`}
            aria-hidden="true"
          />
          <span>{actionLabel}</span>
        </button>
      </div>
    </div>
  );
}
