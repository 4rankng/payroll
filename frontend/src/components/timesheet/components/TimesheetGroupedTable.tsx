import { format } from "date-fns";
import { memo, useCallback, useMemo } from "react";
import {
  Star,
  CheckCircle,
  ChevronRight,
  ChevronDown,
  Clock,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Timesheet } from "@/types/api/timesheet.types";
import { PaytypeHierarchy } from "./PaytypeHierarchy";
import {
  TIMESHEET_STRIP_COLORS,
  PAYMENT_STRIP_COLORS,
} from "../utils/timesheetStatusColors";
import { groupTimesheetsByEmployee, EmployeeGroupedTimesheet } from "../utils/timesheetGrouping";
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
      {/* Horizontal scroll wrapper — preserves desktop layout, scrolls when viewport is too narrow */}
      <div className="overflow-x-auto">
        <Table className="table-fixed min-w-[760px]">
          <colgroup>
            <col style={{ width: "4px" }} />
            <col style={{ width: userRole === "partner" ? "200px" : "190px" }} />
            <col style={{ minWidth: "200px" }} />
            <col style={{ width: "80px" }} />
            <col style={{ width: userRole === "partner" ? "130px" : "120px" }} />
            <col style={{ width: "36px" }} />
          </colgroup>
          <TableHeader>
            <TableRow className="bg-muted/30 hover:bg-muted/30 border-b border-border/40">
              <TableHead className="p-0" />
              <TableHead className="py-3 px-4 text-xs font-semibold uppercase tracking-wider text-muted-foreground whitespace-nowrap">
                Nhân viên
              </TableHead>
              <TableHead className="py-3 px-4 text-xs font-semibold uppercase tracking-wider text-muted-foreground whitespace-nowrap">
                {userRole === "partner" ? "Dự án · Trạng thái" : "Loại ca"}
              </TableHead>
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground text-right whitespace-nowrap">
                {userRole === "partner" ? "Giờ" : "Tổng giờ"}
              </TableHead>
              <TableHead className="py-3 px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground text-right whitespace-nowrap">
                {userRole === "partner" ? "Lương" : "Tổng tiền"}
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
            <TableCell className="py-3.5 px-4">
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
            <TableCell className="py-3.5 px-4">
              {userRole === "partner" ? (
                <PartnerGroupColumn group={group} />
              ) : (
                <span className="inline-flex items-center gap-1 px-2.5 h-6 rounded-full bg-muted text-xs font-semibold text-muted-foreground tabular-nums">
                  {group.entries.length} ca
                </span>
              )}
            </TableCell>
            {/* [3] total hours */}
            <TableCell className="py-3.5 px-3 text-right">
              <span className="typography-body-medium font-bold text-foreground tabular-nums whitespace-nowrap">
                {group.totalHours}h
              </span>
            </TableCell>
            {/* [4] total amount */}
            <TableCell className="py-3.5 px-3 text-right">
              <span className="typography-body-medium font-bold text-foreground tabular-nums whitespace-nowrap">
                {group.totalAmount.toLocaleString("vi-VN")}đ
              </span>
            </TableCell>
            {/* [5] chevron */}
            <TableCell className="py-3.5 px-2">
              <div className="flex justify-center text-muted-foreground group-hover:text-foreground transition-colors">
                {isExpanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
              </div>
            </TableCell>
          </TableRow>
        </CollapsibleTrigger>

        <CollapsibleContent asChild>
          <>
            {group.entries.map((entry, entryIdx) => (
              <EntryRow
                key={entry.id}
                entry={entry}
                entryIdx={entryIdx}
                groupStripBg={groupStripBg}
                userRole={userRole}
                onClick={() => onRowClick(entry)}
              />
            ))}
            <SubtotalRow entryCount={group.entries.length} totalHours={group.totalHours} totalAmount={group.totalAmount} />
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
        {group.entries.some(e => !e.payrate || e.payrate === 0) && (
          <span className="inline-flex items-center gap-1 text-xs text-red-600 font-semibold whitespace-nowrap">
            ⚠ đơn giá
          </span>
        )}
      </div>
    </div>
  );
});

// ── Entry row (expanded child) ─────────────────────────────────────────────

interface EntryRowProps {
  entry: Timesheet;
  entryIdx: number;
  groupStripBg: string;
  userRole: "admin" | "partner";
  onClick: () => void;
}

const EntryRow = memo(function EntryRow({ entry, entryIdx, groupStripBg, userRole, onClick }: EntryRowProps) {
  const navigate = useNavigate();

  return (
    <TableRow
      onClick={onClick}
      className={cn(
        "cursor-pointer transition-colors bg-muted/[0.03] hover:bg-muted/10",
        entryIdx === 0 ? "border-t border-dashed border-border/50" : "border-t border-border/20",
      )}
    >
      <TableCell className="p-0 relative">
        <span className={cn("absolute top-1.5 left-1.5 w-2 h-2 rounded-full", groupStripBg)} />
      </TableCell>
      <TableCell className="py-2.5 px-4 pl-12">
        <div className="min-w-0">
          <div className="flex items-center gap-1">
            <p className="text-xs font-medium text-foreground truncate">{entry.projectName || "-"}</p>
            {entry.force_payroll && <Star className="h-3 w-3 text-yellow-500 fill-yellow-500 shrink-0" />}
          </div>
          <p className="text-xs text-muted-foreground tabular-nums mt-0.5">
            {format(new Date(entry.date), 'dd/MM/yyyy')}
          </p>
        </div>
      </TableCell>
      <TableCell className="py-2.5 px-4">
        {entry.paytype ? (
          <PaytypeHierarchy paytype={entry.paytype} />
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell className="py-2.5 px-4 text-right">
        <p className="text-xs font-semibold text-foreground tabular-nums">{entry.hours_worked || 0}h</p>
        {userRole === "partner" ? (
          <p className={cn(
            "text-xs tabular-nums",
            (!entry.payrate || entry.payrate === 0) ? "text-red-500 font-semibold" : "text-muted-foreground",
          )}>
            {(!entry.payrate || entry.payrate === 0) ? (
              <span
                className="cursor-pointer hover:underline"
                onClick={(e) => {
                  e.stopPropagation();
                  navigate(`/partner/projects/${entry.project_id}/payrates/new/edit`);
                }}
              >
                ⚠ chưa có đơn giá
              </span>
            ) : `${entry.payrate?.toLocaleString("vi-VN")}đ/h`}
          </p>
        ) : (
          <p className="text-xs text-muted-foreground tabular-nums">
            {entry.payrate?.toLocaleString("vi-VN")}đ/h
          </p>
        )}
      </TableCell>
      <TableCell className="py-2.5 px-4 text-right">
        <div className="flex items-center justify-end gap-1">
          {entry.payment_status === "paid" && <CheckCircle className="h-3 w-3 text-green-600 shrink-0" />}
          <span className="text-xs font-semibold text-foreground tabular-nums">
            {(entry.amount || 0).toLocaleString("vi-VN")}đ
          </span>
        </div>
      </TableCell>
      <TableCell className="py-2.5 px-2">
        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/40 mx-auto" />
      </TableCell>
    </TableRow>
  );
});

// ── Sub-total row ──────────────────────────────────────────────────────────

const SubtotalRow = memo(function SubtotalRow({
  entryCount,
  totalHours,
  totalAmount,
}: {
  entryCount: number;
  totalHours: number;
  totalAmount: number;
}) {
  return (
    <TableRow className="border-t border-border/40 bg-muted/10">
      <TableCell className="p-0" />
      <TableCell className="py-2 px-4 pl-12">
        <span className="text-xs text-muted-foreground">{entryCount} mục</span>
      </TableCell>
      <TableCell className="py-2 px-4" />
      <TableCell className="py-2 px-4 text-right">
        <span className="text-xs font-bold text-foreground tabular-nums">{totalHours}h</span>
      </TableCell>
      <TableCell className="py-2 px-4 text-right">
        <span className="text-xs font-bold text-foreground tabular-nums">
          {totalAmount.toLocaleString("vi-VN")}đ
        </span>
      </TableCell>
      <TableCell className="py-2 px-2" />
    </TableRow>
  );
});
