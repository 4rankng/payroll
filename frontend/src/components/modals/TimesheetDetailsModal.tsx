import { useSearchParams } from "react-router-dom";
import { TimesheetEntryModal } from "@/components/timesheet/components/TimesheetEntryModal";
import { useTimesheet } from "@/hooks/api/useTimesheets";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useTabDeepLink, TAB_CONFIGS } from "@/hooks/useTabDeepLink";
import { useModalNavigation } from "@/hooks/useModalNavigation";

export const modalConfig = {
  id: 'timesheet-details',
};

/**
 * Route-based Timesheet Details Modal
 * Integrates with the secure modal system for deep-linking
 */
export function TimesheetDetailsModal() {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();
  const timesheetManagement = useTimesheetManagement();

  const modalId = searchParams.get('modal');
  const timesheetId = searchParams.get('id');
  const isOpen = modalId === 'timesheet_details' && Boolean(timesheetId);

  // Use tab deep-link hook
  const { activeTab, handleTabChange } = useTabDeepLink(TAB_CONFIGS.TIMESHEET_DETAILS);

  // Fetch specific timesheet by ID
  const { data: selectedTimesheet, isLoading } = useTimesheet(
    timesheetId ? parseInt(timesheetId) : 0,
    isOpen && Boolean(timesheetId)
  );

  // Close modal using centralized navigation
  const handleClose = () => {
    closeModal();
  };

  // Don't render if modal is not open
  if (!isOpen) return null;

  if (!selectedTimesheet) return null;

  return (
    <TimesheetEntryModal
      isOpen={isOpen}
      onClose={handleClose}
      projectId={selectedTimesheet.project_id}
      projectName={selectedTimesheet.projectName}
      employeeId={selectedTimesheet.employee_id}
      employeeName={selectedTimesheet.employeeName}
      date={new Date(selectedTimesheet.date)}
      existingEntry={selectedTimesheet}
      onSuccess={handleClose}
    />
  );
}
