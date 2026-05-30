import { useState, useCallback } from "react";
import { showErrorNotification } from "@/utils/error-handler";
import { Timesheet } from "@/types/api/timesheet.types";
import { TimesheetEntryModal } from "./components/TimesheetEntryModal";
import {
  TIMESHEET_STRIP_COLORS,
  PAYMENT_STRIP_COLORS,
} from "./utils/timesheetStatusColors";
import { cn } from "@/lib/utils";
import { TimesheetGroupedTable } from "./components/TimesheetGroupedTable";
import { TimesheetTablePagination } from "./components/TimesheetTablePagination";
import { useTimesheetContext } from "./TimesheetContext";

export function TimesheetListTable() {
  const { state, actions, meta } = useTimesheetContext();
  const [selectedTimesheet, setSelectedTimesheet] = useState<Timesheet | null>(null);
  const [isEntryModalOpen, setIsEntryModalOpen] = useState(false);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

  const toggleGroup = useCallback((groupKey: string) => {
    setExpandedGroups((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(groupKey)) newSet.delete(groupKey); else newSet.add(groupKey);
      return newSet;
    });
  }, []);

  const handleRowClick = useCallback((timesheet: Timesheet) => {
    setSelectedTimesheet(timesheet);
    setIsEntryModalOpen(true);
  }, []);

  const handleEntryModalClose = useCallback(() => {
    setIsEntryModalOpen(false);
    setSelectedTimesheet(null);
  }, []);

  const handleRequestEditWithClose = useCallback(
    (timesheet: Timesheet) => {
      if (!actions.requestEdit) return;
      void Promise.resolve(actions.requestEdit(timesheet, handleEntryModalClose)).catch((err) => {
        showErrorNotification(err);
      });
    },
    [actions, handleEntryModalClose],
  );

  return (
    <div className="space-y-5">
      {state.userRole === "admin" && (
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          {[
            { bg: TIMESHEET_STRIP_COLORS.pending_approval.bg, label: "Chờ duyệt" },
            { bg: TIMESHEET_STRIP_COLORS.approved.bg, label: "Đã duyệt" },
            { bg: PAYMENT_STRIP_COLORS.paid.bg, label: "Đã thanh toán" },
            { bg: TIMESHEET_STRIP_COLORS.rejected.bg, label: "Loại" },
          ].map(({ bg, label }) => (
            <div key={label} className="flex items-center gap-1">
              <div className={cn("w-1.5 h-1.5 rounded-full shrink-0", bg)} />
              <span>{label}</span>
            </div>
          ))}
        </div>
      )}

      <TimesheetGroupedTable
        timesheets={state.timesheets}
        userRole={state.userRole}
        expandedGroups={expandedGroups}
        onToggleGroup={toggleGroup}
        onRowClick={handleRowClick}
      />

      {state.pagination && actions.pageChange && (
        <TimesheetTablePagination
          page={state.pagination.page}
          pageSize={state.pagination.pageSize}
          totalPages={state.pagination.totalPages}
          totalRecords={state.pagination.totalRecords}
          onPageChange={actions.pageChange}
        />
      )}

      {selectedTimesheet && (
        <TimesheetEntryModal
          isOpen={isEntryModalOpen}
          onClose={handleEntryModalClose}
          projectId={selectedTimesheet.project_id}
          projectName={selectedTimesheet.projectName}
          employeeId={selectedTimesheet.employee_id}
          employeeName={selectedTimesheet.employeeName}
          date={new Date(selectedTimesheet.date)}
          existingEntry={selectedTimesheet}
          onSuccess={handleEntryModalClose}
          onDelete={actions.delete}
          canDelete={actions.canDelete}
          onRequestEdit={handleRequestEditWithClose}
          onRequestEditSuccess={handleEntryModalClose}
          requestingTimesheetId={meta.requestingTimesheetId}
        />
      )}
    </div>
  );
}
