import { useState, useMemo, useEffect, useCallback } from "react";
import { showErrorNotification } from "@/utils/error-handler";
import { Timesheet } from "@/types/api/timesheet.types";
import { SortingState, OnChangeFn } from "@tanstack/react-table";
import { TimesheetEntryModal } from "./components/TimesheetEntryModal";
import { userService } from "@/services/api/user.service";
import type { User } from "@/types/user";
import {
  TIMESHEET_STRIP_COLORS,
  PAYMENT_STRIP_COLORS,
} from "./utils/timesheetStatusColors";
import { cn } from "@/lib/utils";
import { TimesheetGroupedTable } from "./components/TimesheetGroupedTable";
import { TimesheetTablePagination } from "./components/TimesheetTablePagination";

interface TimesheetListTableProps {
  timesheets: Timesheet[];
  isLoading?: boolean;
  error?: unknown;
  onView: (timesheet: Timesheet) => void;
  onEdit: (timesheet: Timesheet) => void;
  onApprove: (timesheet: Timesheet) => void;
  onReject: (timesheet: Timesheet, reason: string) => void;
  onDelete: (timesheet: Timesheet) => void;
  canDelete?: (timesheet: Timesheet) => boolean;
  onExportExcel: () => void;
  isExportLoading?: boolean;
  bulkTransferPercentage?: number;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  } | null;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  onRefetch?: () => void;
  onRequestEdit?: (
    timesheet: Timesheet,
    onSuccess?: () => Promise<void> | void,
  ) => Promise<void> | void;
  requestingTimesheetId?: number | null;
  userRole?: "admin" | "partner";
  sorting?: SortingState;
  onSortingChange?: OnChangeFn<SortingState>;
}

export function TimesheetListTable({
  timesheets,
  isLoading = false,
  error,
  onView,
  onEdit,
  onApprove,
  onReject,
  onDelete,
  canDelete,
  onExportExcel,
  isExportLoading = false,
  bulkTransferPercentage = 0,
  pagination,
  onPageChange,
  onPageSizeChange,
  onRefetch,
  onRequestEdit,
  requestingTimesheetId = null,
  userRole = "admin",
  sorting,
  onSortingChange,
}: TimesheetListTableProps) {
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
      if (!onRequestEdit) return;
      void Promise.resolve(onRequestEdit(timesheet, handleEntryModalClose)).catch((err) => {
        showErrorNotification(err);
      });
    },
    [onRequestEdit, handleEntryModalClose],
  );

  return (
    <div className="space-y-5">
      {/* Status legend (admin only) */}
      {userRole === "admin" && (
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
        timesheets={timesheets}
        userRole={userRole}
        expandedGroups={expandedGroups}
        onToggleGroup={toggleGroup}
        onRowClick={handleRowClick}
      />

      {pagination && onPageChange && (
        <TimesheetTablePagination
          page={pagination.page}
          pageSize={pagination.pageSize}
          totalPages={pagination.totalPages}
          totalRecords={pagination.totalRecords}
          onPageChange={onPageChange}
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
          onDelete={onDelete}
          canDelete={canDelete}
          onRequestEdit={handleRequestEditWithClose}
          onRequestEditSuccess={handleEntryModalClose}
          requestingTimesheetId={requestingTimesheetId}
        />
      )}
    </div>
  );
}
