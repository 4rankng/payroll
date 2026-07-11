import {
  AlertCircle,
  BadgeCheck,
  BriefcaseBusiness,
  DoorOpen,
  Loader2,
  WalletCards,
} from "lucide-react";
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

  return (
    <div
      className="fixed inset-x-0 bottom-0 z-40 border-t border-slate-200 bg-white lg:hidden"
      style={{ paddingBottom: "env(safe-area-inset-bottom)" }}
      role="toolbar"
      aria-label="Hành động nhân viên"
    >
      <div className="mx-auto grid max-w-lg grid-cols-[minmax(0,2fr)_minmax(0,3fr)] gap-2 px-3 py-2">
        <Button
          type="button"
          variant="outline"
          className="employee-type-action h-11 min-w-0 rounded-[10px] border-[#07883F] px-3 text-sm font-semibold text-[#067647] hover:bg-[#ECFDF3] hover:text-[#067647]"
          onClick={onAdvanceRequest}
        >
          <WalletCards className="h-4 w-4" aria-hidden="true" />
          Ứng lương
        </Button>

        <Button
          type="button"
          className="employee-type-action h-11 min-w-0 rounded-[10px] bg-[#07883F] px-3 text-sm font-semibold text-white shadow-none hover:bg-[#067647] disabled:bg-slate-200 disabled:text-slate-500"
          disabled={disabled}
          onClick={onAttendanceAction}
        >
          <ActionIcon
            className={`h-4 w-4 ${action === "loading" ? "animate-spin" : ""}`}
            aria-hidden="true"
          />
          <span className="truncate">{actionLabel}</span>
        </Button>
      </div>
    </div>
  );
}
