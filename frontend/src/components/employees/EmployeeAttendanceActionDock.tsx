import {
  AlertCircle,
  BadgeCheck,
  BriefcaseBusiness,
  DoorOpen,
  Loader2,
  WalletCards,
} from "lucide-react";
import { useLayoutEffect, useRef } from "react";
import { Button } from "@/components/ui/button";

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
      className="employee-attendance-action-dock fixed inset-x-0 bottom-0 z-40 min-h-[calc(4.25rem+env(safe-area-inset-bottom))] border-t border-[var(--employee-border)] bg-white"
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      role="toolbar"
      aria-label="Hành động nhân viên"
    >
      <div className="mx-auto grid min-h-[4.25rem] max-w-lg grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-stretch gap-2 px-3 py-2">
        <Button
          type="button"
          variant="outline"
          className="employee-type-action h-auto min-h-11 min-w-0 whitespace-normal rounded-[var(--employee-radius-control)] border-[var(--employee-accent)] px-3 py-2 text-center text-sm font-semibold leading-tight text-[var(--employee-accent)] hover:bg-[var(--employee-accent-soft)] hover:text-[var(--employee-accent)]"
          onClick={onAdvanceRequest}
        >
          <WalletCards className="h-4 w-4" aria-hidden="true" />
          Ứng lương
        </Button>

        <Button
          type="button"
          className="employee-type-action h-auto min-h-11 min-w-0 whitespace-normal rounded-[var(--employee-radius-control)] bg-[var(--employee-accent)] px-3 py-2 text-center text-sm font-semibold leading-tight text-white shadow-none hover:bg-[var(--employee-accent-strong)] disabled:bg-[var(--employee-border)] disabled:text-[var(--employee-text-secondary)]"
          disabled={disabled}
          onClick={onAttendanceAction}
        >
          <ActionIcon
            className={`h-4 w-4 ${action === "loading" ? "animate-spin" : ""}`}
            aria-hidden="true"
          />
          <span>{actionLabel}</span>
        </Button>
      </div>
    </div>
  );
}
