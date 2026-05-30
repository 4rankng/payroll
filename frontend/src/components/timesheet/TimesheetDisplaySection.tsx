import { TimesheetFilters } from '@/components/timesheet/TimesheetFilters';
import { TimesheetListTable } from '@/components/timesheet/TimesheetListTable';
import { TimesheetMobileList } from '@/components/timesheet/TimesheetMobileList';
import { EditRequestTable } from '@/components/timesheet/EditRequestTable';
import { TimesheetEmptyState } from '@/components/timesheet/TimesheetEmptyState';
import { useMediaQuery } from '@/hooks/use-media-query';
import { useTimesheetSummary } from '@/hooks/api/useTimesheets';
import type { Timesheet } from '@/types/api/timesheet.types';
import type { useTimesheetManagement } from '@/hooks/timesheet/useTimesheetManagement';

interface TimesheetDisplaySectionProps {
  timesheetManagement: ReturnType<typeof useTimesheetManagement>;
  onEdit: (timesheet: Timesheet) => void;
  onApprove?: (timesheet: Timesheet) => void;
  onDelete?: (timesheet: Timesheet) => void;
  onAddTimesheet?: () => void;
  onRequestEdit?: (timesheet: Timesheet, onSuccess?: () => Promise<void> | void) => Promise<void> | void;
  requestingTimesheetId?: number | null;
  onBulkApprove?: () => void;
  bulkTransferPercentage?: number;
  showEditRequestTable?: boolean;
  userRole?: 'admin' | 'partner';
  onEditRequestRowClick?: (timesheet: Timesheet) => void;
  onExportExcel?: () => void;
  onPaymentHistory?: () => void;
  isExportLoading?: boolean;
}

export function TimesheetDisplaySection({
  timesheetManagement,
  onEdit,
  onApprove,
  onDelete,
  onAddTimesheet,
  bulkTransferPercentage = 0,
  onRequestEdit,
  requestingTimesheetId = null,
  showEditRequestTable = false,
  userRole = 'admin',
  onEditRequestRowClick,
  onExportExcel,
  onPaymentHistory,
  isExportLoading = false,
}: TimesheetDisplaySectionProps) {
  const isMobile = useMediaQuery('(max-width: 767px)');
  // Partner role doesn't approve/reject - pass no-op functions
  const handleApprove = userRole === 'partner' ? () => {} : (onApprove || timesheetManagement.handleApprove);
  const handleReject = userRole === 'partner' ? () => {} : timesheetManagement.handleReject;
  const handleDelete = onDelete || timesheetManagement.handleDelete;

  // Detect "active filter" state — anything other than the default month picker
  // counts as an active filter for the empty-state copy.
  const hasActiveFilter =
    timesheetManagement.searchTerm.trim() !== '' ||
    (timesheetManagement.selectedProject !== 'all' && timesheetManagement.selectedProject !== '') ||
    (timesheetManagement.selectedEmployee !== 'all' && timesheetManagement.selectedEmployee !== '') ||
    timesheetManagement.statusFilter !== 'all';

  // Pull the unfiltered summary so we can decide between "no data this period"
  // (no filter, totalEntries === 0) and "no filter results" (filter set, list empty).
  const { data: globalSummary } = useTimesheetSummary({});

  const isListEmpty =
    !timesheetManagement.isLoading && timesheetManagement.timesheets.length === 0;

  const handleClearFilters = () => {
    timesheetManagement.setSearchTerm('');
    timesheetManagement.setSelectedProject('all');
    timesheetManagement.setSelectedEmployee('all');
    timesheetManagement.setStatusFilter('all');
  };

  return (
    <>
      {showEditRequestTable && (
        <EditRequestTable userRole={userRole} onRowClick={onEditRequestRowClick} />
      )}

      <TimesheetFilters
        selectedMonth={timesheetManagement.selectedMonth}
        onMonthChange={timesheetManagement.setSelectedMonth}
        selectedProject={timesheetManagement.selectedProject}
        onProjectChange={timesheetManagement.setSelectedProject}
        selectedEmployee={timesheetManagement.selectedEmployee}
        onEmployeeChange={timesheetManagement.setSelectedEmployee}
        statusFilter={timesheetManagement.statusFilter}
        onStatusChange={timesheetManagement.setStatusFilter}
        projects={timesheetManagement.projects}
        projectEmployees={timesheetManagement.projectEmployees}
        searchTerm={timesheetManagement.searchTerm}
        onSearchChange={timesheetManagement.setSearchTerm}
        onAddTimesheet={onAddTimesheet}
        onExportExcel={onExportExcel}
        onPaymentHistory={onPaymentHistory}
        isExportLoading={isExportLoading}
        userRole={userRole}
      />

      {/* Empty / "all approved" states — only relevant for admin desktop view */}
      {userRole === 'admin' && !isMobile && (
        <>
          {isListEmpty && hasActiveFilter && (
            <TimesheetEmptyState variant="no-filter-results" onClearFilters={handleClearFilters} />
          )}
          {isListEmpty && !hasActiveFilter && (globalSummary?.totalEntries ?? 0) === 0 && (
            <TimesheetEmptyState variant="no-data" onAddTimesheet={onAddTimesheet} />
          )}

        </>
      )}

      <div
        className={isMobile ? '' : 'pb-6'}
        // Hide the table when an action-oriented empty state is already shown above
        // (avoids stacking two empty placeholders).
        hidden={!isMobile && userRole === 'admin' && isListEmpty}
      >
        {isMobile ? (
          <TimesheetMobileList
            timesheets={timesheetManagement.timesheets}
            isLoading={timesheetManagement.isLoading}
            error={timesheetManagement.error}
            onView={timesheetManagement.handleView}
            onEdit={onEdit}
            onApprove={handleApprove}
            onReject={handleReject}
            onDelete={handleDelete}
            canDelete={timesheetManagement.canDeleteTimesheet}
            onExportExcel={timesheetManagement.handleExportExcel}
            isExportLoading={timesheetManagement.isLoading}
            bulkTransferPercentage={bulkTransferPercentage}
            pagination={timesheetManagement.paginationInfo}
            onPageChange={timesheetManagement.handlePageChange}
            onPageSizeChange={timesheetManagement.handlePageSizeChange}
            onRequestEdit={onRequestEdit}
            requestingTimesheetId={requestingTimesheetId}
          />
        ) : (
          <TimesheetListTable
            timesheets={timesheetManagement.timesheets}
            isLoading={timesheetManagement.isLoading}
            error={timesheetManagement.error}
            onView={timesheetManagement.handleView}
            onEdit={onEdit}
            onApprove={handleApprove}
            onReject={handleReject}
            onDelete={handleDelete}
            canDelete={timesheetManagement.canDeleteTimesheet}
            onExportExcel={timesheetManagement.handleExportExcel}
            isExportLoading={timesheetManagement.isLoading}
            bulkTransferPercentage={bulkTransferPercentage}
            pagination={timesheetManagement.paginationInfo}
            onPageChange={timesheetManagement.handlePageChange}
            onPageSizeChange={timesheetManagement.handlePageSizeChange}
            sorting={timesheetManagement.sorting}
            onSortingChange={timesheetManagement.setSorting}
            onRequestEdit={onRequestEdit}
            requestingTimesheetId={requestingTimesheetId}
            userRole={userRole}
          />
        )}
      </div>
    </>
  );
}
