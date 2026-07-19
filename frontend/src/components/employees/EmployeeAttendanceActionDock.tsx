import {
  AlertCircle,
  BadgeCheck,
  BriefcaseBusiness,
  DoorOpen,
  Loader2,
  WalletCards,
} from "lucide-react";
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
  check_in: BriefcaseBusiness,
  check_out: DoorOpen,
  loading: Loader2,
  completed: BadgeCheck,
  attention: AlertCircle,
} satisfies Record<AttendanceDockAction, typeof BriefcaseBusiness>;

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
      className="employee-attendance-action-dock fixed inset-x-0 bottom-0 z-40 min-h-[calc(4.25rem+env(safe-area-inset-bottom))] border-t border-base-300 bg-base-100/95 shadow-[0_-12px_32px_-24px_rgba(16,24,40,0.35)] backdrop-blur"
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      role="toolbar"
      aria-label="Hành động nhân viên"
    >
      <div className="mx-auto grid min-h-[4.25rem] max-w-lg grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-stretch gap-2 px-3 py-2">
        <button
          type="button"
          className="ct-btn ct-btn-outline ct-btn-primary employee-type-action h-auto min-h-11 min-w-0 whitespace-normal rounded-[var(--employee-radius-control)] px-3 py-2 text-center text-sm font-semibold leading-tight normal-case"
          onClick={onAdvanceRequest}
        >
          <WalletCards className="h-4 w-4" aria-hidden="true" />
          Ứng lương
        </button>

        <button
          type="button"
          className="ct-btn ct-btn-primary employee-type-action h-auto min-h-11 min-w-0 whitespace-normal rounded-[var(--employee-radius-control)] px-3 py-2 text-center text-sm font-semibold leading-tight normal-case shadow-[var(--employee-cta-shadow)]"
          disabled={disabled}
          onClick={onAttendanceAction}
        >
          <ActionIcon
            className={`h-4 w-4 ${action === "loading" ? "animate-spin" : ""}`}
            aria-hidden="true"
          />
          <span>{actionLabel}</span>
        </button>
      </div>
    </div>
  );
}
