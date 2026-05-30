import { useIsMobile } from '@/hooks/useIsMobile';
import { TimesheetEntrySheet } from './TimesheetEntrySheet';

interface TimesheetEntrySheetWrapperProps {
  isOpen: boolean;
  onClose: () => void;
  employeeId?: number;
  projectId?: number;
  onSuccess?: () => Promise<void>;
}

/**
 * On mobile, timesheet entry is a full-page view handled by TimesheetsPageMobile
 * (which intercepts ?modal=timesheet_entry in the URL).
 * ModalRouter still mounts this component, so we return null on mobile to avoid
 * rendering the desktop sheet on top of the page.
 */
export function TimesheetEntrySheetWrapper(props: TimesheetEntrySheetWrapperProps) {
  const isMobile = useIsMobile();

  if (isMobile) return null;

  return <TimesheetEntrySheet {...props} />;
}

export default TimesheetEntrySheetWrapper;
