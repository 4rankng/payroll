import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { memo, useCallback, useMemo } from "react";
import {
  Star,
  ChevronRight,
  ChevronDown,
  Clock,
  Building2,
  Moon,
  SunMedium,
  TimerReset,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Timesheet } from "@/types/api/timesheet.types";
import {
  TIMESHEET_STRIP_COLORS,
  PAYMENT_STRIP_COLORS,
} from "../utils/timesheetStatusColors";
import {
  groupTimesheetsByEmployee,
  groupEmployeeEntriesByProject,
  getPaytypeDetail,
  EmployeeGroupedTimesheet,
  ProjectTimesheetSection,
} from "../utils/timesheetGrouping";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "@/components/ui/table";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { UserAvatar } from "@/components/ui/user-avatar";
import { cn } from "@/lib/utils";
import { EmptyState } from "@/components/shared/EmptyState";

export interface TimesheetGroupedTableProps {
  timesheets: Timesheet[];
  userRole: "admin" | "partner";
  expandedGroups: Set<string>;
  onToggleGroup: (groupKey: string) => void;
  onRowClick: (timesheet: Timesheet) => void;
}

function getStatusStripBg(status: string, paymentStatus?: string): string {
  if (paymentStatus === "paid" || status === "paid") return "bg-green-500";
  switch (status) {
    case "approved": return "bg-blue-500";
    case "pending_approval": return "bg-amber-400";
    case "rejected": return "bg-red-500";
    default: return "bg-gray-300";
  }
}

export const TimesheetGroupedTable = memo(function TimesheetGroupedTable({
  timesheets,
  userRole,
  expandedGroups,
  onToggleGroup,
  onRowClick,
}: TimesheetGroupedTableProps) {
  const grouped = useMemo(
    () => groupTimesheetsByEmployee(timesheets),
    [timesheets],
  );

  if (grouped.length === 0) {
    return (
      <div className="rounded-2xl border border-border/60 p-20 text-center">
        <EmptyState icon={Clock} title="Không có dữ liệu bảng công" />
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-border/50 overflow-hidden">
      {/* Scroll only when the desktop workspace is genuinely too narrow. */}
      <div className="overflow-x-auto">
        <Table className="table-fixed min-w-[860px]">
          <colgroup>
            <col style={{ width: "4px" }} />
            <col style={{ width: "170px" }} />
            <col style={{ width: "130px" }} />
            <col style={{ width: "150px" }} />
            <col style={{ width: "80px" }} />
            <col style={{ width: "125px" }} />
            <col style={{ width: "145px" }} />
            <col style={{ width: "36px" }} />
          </colgroup>
          <TableHeader>
            <TableRow className="bg-muted/30 hover:bg-muted/30 border-b border-border/40">
              <TableHead className="p-0" />
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground whitespace-nowrap">
                {userRole === "partner" ? "Nhân viên · Ngày" : "Nhân viên"}
              </TableHead>
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground whitespace-nowrap">
                {userRole === "partner" ? "Dự án · Loại ngày" : "Loại ngày"}
              </TableHead>
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground whitespace-nowrap">
                Ca làm · Trạng thái
              </TableHead>
              <TableHead className="border-l border-border/30 py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground text-right whitespace-nowrap">
                Giờ
              </TableHead>
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground text-right whitespace-nowrap">
                {userRole === "admin" ? "Dự án · Đơn giá" : "Đơn giá"}
              </TableHead>
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground text-right whitespace-nowrap">
                Thành tiền
              </TableHead>
              <TableHead className="py-3 px-2" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {grouped.map((group, idx) => (
              <GroupRow
                key={group.groupKey}
                group={group}
                idx={idx}
                userRole={userRole}
                isExpanded={expandedGroups.has(group.groupKey)}
                onToggle={() => onToggleGroup(group.groupKey)}
                onRowClick={onRowClick}
              />
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
});

// ── Group row (collapsible header + children) ──────────────────────────────

interface GroupRowProps {
  group: EmployeeGroupedTimesheet;
  idx: number;
  userRole: "admin" | "partner";
  isExpanded: boolean;
  onToggle: () => void;
  onRowClick: (timesheet: Timesheet) => void;
}

const GroupRow = memo(function GroupRow({
  group,
  idx,
  userRole,
  isExpanded,
  onToggle,
  onRowClick,
}: GroupRowProps) {
  const groupStripBg = getStatusStripBg(group.aggregatedStatus);
  const projectSections = useMemo(
    () => groupEmployeeEntriesByProject(group.entries),
    [group.entries],
  );

  return (
    <Collapsible asChild open={isExpanded} onOpenChange={onToggle}>
      <>
        <CollapsibleTrigger asChild>
          <TableRow
            className={cn(
              "group cursor-pointer transition-colors hover:bg-muted/30",
              idx !== 0 && "border-t border-border/40",
              isExpanded && "bg-muted/20",
            )}
          >
            {/* [0] status strip */}
            <TableCell className="p-0">
              <div className={cn("w-1 h-full min-h-[52px] rounded-r-full transition-colors", groupStripBg)} />
            </TableCell>
            {/* [1] employee */}
            <TableCell className="py-3.5 px-3">
              <div className="flex items-center gap-2.5 min-w-0">
                <UserAvatar email={group.employeeCode} name={group.employeeName} size="sm" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-semibold text-foreground leading-tight truncate">
                    {group.employeeName || "-"}
                  </p>
                  <p className="text-xs text-muted-foreground leading-tight mt-0.5 tabular-nums truncate">
                    {group.employeeCode || "-"}
                  </p>
                </div>
              </div>
            </TableCell>
            {/* [2] role-specific column */}
            <TableCell className="py-3.5 px-3">
              {userRole === "partner" ? (
                <PartnerGroupColumn group={group} />
              ) : (
                <span className="inline-flex items-center gap-1 px-2.5 h-6 rounded-full bg-muted text-xs font-semibold text-muted-foreground tabular-nums">
                  {group.entries.length} ca
                </span>
              )}
            </TableCell>
            {/* [3] status summary */}
            <TableCell className="py-3.5 px-3">
              <GroupStatusSummary group={group} />
            </TableCell>
            {/* [4] total hours */}
            <TableCell className="border-l border-border/30 py-3.5 px-3 text-right">
              <span className="inline-flex h-7 min-w-14 items-center justify-end rounded-md bg-background/80 px-2 typography-body-medium font-bold text-foreground tabular-nums whitespace-nowrap">
                {group.totalHours}h
              </span>
            </TableCell>
            {/* [5] Project preview for Admin summaries; rate lives in expanded rows. */}
            <TableCell className="py-3.5 px-3 text-right">
              {userRole === "admin" ? (
                <AdminGroupProjectColumn group={group} />
              ) : (
                <span className="text-xs text-muted-foreground/35">—</span>
              )}
            </TableCell>
            {/* [6] total amount */}
            <TableCell className="py-3.5 px-3 text-right">
              <span className="inline-flex h-7 items-center justify-end rounded-md bg-emerald-50 px-2 typography-body-medium font-bold text-emerald-800 tabular-nums whitespace-nowrap">
                {group.totalAmount.toLocaleString("vi-VN")}
                <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
              </span>
            </TableCell>
            {/* [7] chevron */}
            <TableCell className="py-3.5 px-2">
              <div className="flex justify-center text-muted-foreground group-hover:text-foreground transition-colors">
                {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
              </div>
            </TableCell>
          </TableRow>
        </CollapsibleTrigger>

        <CollapsibleContent asChild>
          <>
            {projectSections.map((project, projectIndex) => (
              <ProjectSection
                key={project.projectId}
                project={project}
                showHeader={projectSections.length > 1}
                isFirstProject={projectIndex === 0}
                userRole={userRole}
                onRowClick={onRowClick}
              />
            ))}
          </>
        </CollapsibleContent>
      </>
    </Collapsible>
  );
});

// ── Partner-specific group column ──────────────────────────────────────────

const PartnerGroupColumn = memo(function PartnerGroupColumn({ group }: { group: EmployeeGroupedTimesheet }) {
  return (
    <div className="flex flex-col gap-1 min-w-0">
      {(() => {
        const projects = [...new Set(group.entries.map(e => e.projectName).filter(Boolean))];
        return (
          <p className="text-xs font-medium text-foreground truncate" title={projects.join(", ")}>
            {projects.length === 1 ? projects[0] : projects.length > 1 ? `${projects[0]} +${projects.length - 1}` : "—"}
          </p>
        );
      })()}
    </div>
  );
});

const AdminGroupProjectColumn = memo(function AdminGroupProjectColumn({ group }: { group: EmployeeGroupedTimesheet }) {
  const projects = [...new Set(group.entries.map((entry) => entry.projectName).filter(Boolean))];
  const projectLabel = projects.length === 1
    ? projects[0]
    : projects.length > 1
      ? `${projects[0]} +${projects.length - 1}`
      : "—";

  return (
    <div
      className="ml-auto flex w-full min-w-0 items-center justify-end gap-1.5 text-left"
      title={projects.join(", ")}
    >
      <Building2 aria-hidden="true" className="h-3.5 w-3.5 shrink-0 text-primary/70" />
      <span className="min-w-0 truncate text-xs font-semibold text-foreground">
        {projectLabel}
      </span>
    </div>
  );
});

const GroupStatusSummary = memo(function GroupStatusSummary({ group }: { group: EmployeeGroupedTimesheet }) {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
        {group.statusBreakdown.pending_approval > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-amber-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-400 shrink-0" />
            {group.statusBreakdown.pending_approval} chờ
          </span>
        )}
        {group.statusBreakdown.approved > 0 && group.aggregatedStatus !== "paid" && (
          <span className="inline-flex items-center gap-1 text-xs text-blue-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-500 shrink-0" />
            {group.statusBreakdown.approved} duyệt
          </span>
        )}
        {group.aggregatedStatus === "paid" && (
          <span className="inline-flex items-center gap-1 text-xs text-green-700 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500 shrink-0" />
            {group.entries.length} đã TT
          </span>
        )}
        {group.statusBreakdown.rejected > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-red-600 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />
            {group.statusBreakdown.rejected} loại
          </span>
        )}
        {group.statusBreakdown.draft > 0 && (
          <span className="inline-flex items-center gap-1 text-xs text-slate-600 whitespace-nowrap">
            <span className="w-1.5 h-1.5 rounded-full bg-slate-400 shrink-0" />
            {group.statusBreakdown.draft} nháp
          </span>
        )}
        {group.entries.some(e => !e.payrate || e.payrate === 0) && (
          <span className="inline-flex items-center gap-1 text-xs text-red-600 font-semibold whitespace-nowrap">
            ⚠ đơn giá
          </span>
        )}
    </div>
  );
});

const ProjectSection = memo(function ProjectSection({
  project,
  showHeader,
  isFirstProject,
  userRole,
  onRowClick,
}: {
  project: ProjectTimesheetSection;
  showHeader: boolean;
  isFirstProject: boolean;
  userRole: "admin" | "partner";
  onRowClick: (timesheet: Timesheet) => void;
}) {
  return (
    <>
      {showHeader && (
        <TableRow className={cn("bg-primary/[0.035] hover:bg-primary/[0.035]", isFirstProject && "border-t border-dashed border-border/50")}>
          <TableCell className="p-0" />
          <TableCell colSpan={3} className="px-3 py-2.5 pl-6">
            <div className="flex min-w-0 items-center gap-2.5">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Building2 className="h-3.5 w-3.5" />
              </span>
              <div className="min-w-0">
                <p className="truncate text-xs font-bold text-foreground">{project.projectName}</p>
                <p className="mt-0.5 truncate text-[10.5px] text-muted-foreground">
                  {[project.projectCode, project.positions.join(', '), `${project.entries.length} mục`].filter(Boolean).join(' · ')}
                </p>
              </div>
            </div>
          </TableCell>
          <TableCell className="px-4 py-2.5 text-right text-xs font-bold tabular-nums text-foreground">{project.totalHours}h</TableCell>
          <TableCell className="px-4 py-2.5 text-right text-xs text-muted-foreground/35">—</TableCell>
          <TableCell className="px-4 py-2.5 text-right text-xs font-bold tabular-nums text-foreground">
            {project.totalAmount.toLocaleString("vi-VN")}
            <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
          </TableCell>
          <TableCell className="p-0" />
        </TableRow>
      )}
      {project.entries.map((entry, entryIndex) => (
        <EntryRow
          key={entry.id}
          entry={entry}
          entryIdx={entryIndex}
          showDate={entryIndex === 0 || project.entries[entryIndex - 1]?.date !== entry.date}
          startsAfterProjectHeader={showHeader && entryIndex === 0}
          userRole={userRole}
          onClick={() => onRowClick(entry)}
        />
      ))}
    </>
  );
});

// ── Entry row (expanded child) ─────────────────────────────────────────────

interface EntryRowProps {
  entry: Timesheet;
  entryIdx: number;
  showDate: boolean;
  startsAfterProjectHeader: boolean;
  userRole: "admin" | "partner";
  onClick: () => void;
}

const EntryRow = memo(function EntryRow({ entry, entryIdx, showDate, startsAfterProjectHeader, userRole, onClick }: EntryRowProps) {
  const navigate = useNavigate();
  const entryDate = new Date(entry.date);
  const hoursWorked = entry.hours_worked || 0;
  const dayTypeLabel = entry.day_type || "—";
  const normalizedDayType = dayTypeLabel.toLocaleLowerCase("vi-VN");
  const shiftLabel = entry.hour_type || getPaytypeDetail(entry.paytype) || "—";
  const normalizedShift = shiftLabel.toLocaleLowerCase("vi-VN");
  const isOvertime = normalizedShift.includes("tăng ca");
  const isNightShift = normalizedShift.includes("đêm");
  const isShortShift = hoursWorked > 0 && hoursWorked < 8;

  return (
    <TableRow
      onClick={onClick}
      className={cn(
        "cursor-pointer transition-colors hover:bg-primary/[0.045]",
        entryIdx % 2 === 0 ? "bg-background" : "bg-slate-50/55",
        entryIdx === 0 && !startsAfterProjectHeader ? "border-t border-dashed border-border/50" : "border-t border-border/20",
      )}
    >
      <TableCell className="p-0 relative">
        {(!entry.payrate || entry.payrate === 0) && <span className="absolute left-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500" />}
      </TableCell>
      <TableCell className="py-2 px-3 pl-6">
        <div className="flex min-h-9 items-center gap-2">
          {showDate ? (
            <div className="flex items-center gap-2">
              <span className="inline-flex h-7 min-w-9 items-center justify-center rounded-md bg-slate-100 px-1.5 text-[10px] font-bold uppercase text-slate-600">
                {format(entryDate, "EEE", { locale: vi }).replace("Th ", "T")}
              </span>
              <p className="text-xs font-bold tabular-nums text-foreground">{format(entryDate, "dd/MM/yyyy")}</p>
            </div>
          ) : (
            <span className="ml-4 flex items-center gap-1.5 text-[10px] font-medium text-muted-foreground" aria-label="Cùng ngày">
              <span className="h-4 w-px bg-border" />
              <span className="h-1.5 w-1.5 rounded-full bg-border" />
            </span>
          )}
          {entry.force_payroll && <Star className="h-3 w-3 text-yellow-500 fill-yellow-500 shrink-0" />}
        </div>
      </TableCell>
      <TableCell className="py-2 px-3">
        <span
          className={cn(
            "inline-flex h-7 items-center rounded-md px-2.5 text-xs font-semibold capitalize ring-1 ring-inset",
            normalizedDayType.includes("lễ")
              ? "bg-rose-50 text-rose-800 ring-rose-200"
              : normalizedDayType.includes("nghỉ")
                ? "bg-teal-50 text-teal-800 ring-teal-200"
                : "bg-slate-50 text-slate-700 ring-slate-200",
          )}
        >
          {dayTypeLabel}
        </span>
      </TableCell>
      <TableCell className="py-2 px-3">
        <span
          className={cn(
            "inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-semibold capitalize",
            isOvertime
              ? "bg-amber-50 text-amber-800 ring-1 ring-inset ring-amber-200"
              : isNightShift
                ? "bg-teal-50 text-teal-800 ring-1 ring-inset ring-teal-200"
                : "bg-sky-50 text-sky-800 ring-1 ring-inset ring-sky-200",
          )}
        >
          {isOvertime ? (
            <TimerReset className="h-3.5 w-3.5" />
          ) : isNightShift ? (
            <Moon className="h-3.5 w-3.5" />
          ) : (
            <SunMedium className="h-3.5 w-3.5" />
          )}
          {shiftLabel}
        </span>
      </TableCell>
      <TableCell className="border-l border-border/30 py-2 px-3 text-right">
        <span
          className={cn(
            "inline-flex h-7 min-w-12 items-center justify-end rounded-md px-2 text-xs font-bold tabular-nums",
            isShortShift ? "bg-amber-50 text-amber-800" : "bg-emerald-50 text-emerald-800",
          )}
        >
          {hoursWorked}h
        </span>
      </TableCell>
      <TableCell className="py-2 px-3 text-right">
        <p className={cn(
          "inline-flex h-7 items-center justify-end rounded-md bg-slate-50 px-2 text-xs tabular-nums",
          (!entry.payrate || entry.payrate === 0) ? "text-red-500 font-semibold" : "font-medium text-muted-foreground",
        )}>
          {(!entry.payrate || entry.payrate === 0) && userRole === "partner" ? (
            <span
              className="cursor-pointer hover:underline"
              onClick={(e) => {
                e.stopPropagation();
                navigate(`/partner/projects/${entry.project_id}/payrates/new/edit`);
              }}
            >
              ⚠ chưa có
            </span>
          ) : `${entry.payrate?.toLocaleString("vi-VN")}${entry.projectIsFlexible ? "₫/ca" : "₫/giờ"}`}
        </p>
      </TableCell>
      <TableCell className="py-2 px-3 text-right">
        <div className="flex items-center justify-end gap-1">
          <span className="inline-flex h-7 items-center justify-end rounded-md bg-emerald-50 px-2 text-xs font-bold text-emerald-800 tabular-nums">
            {(entry.amount || 0).toLocaleString("vi-VN")}
            <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
          </span>
        </div>
      </TableCell>
      <TableCell className="py-2 px-2">
        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/40 mx-auto" />
      </TableCell>
    </TableRow>
  );
});
