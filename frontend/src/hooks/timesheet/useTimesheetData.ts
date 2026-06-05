import { 
  useTimesheets, 
  useCreateTimesheet, 
  useUpdateTimesheet, 
  useApproveTimesheet,
  useBulkApproveTimesheets,
  useDeleteTimesheet
} from "@/hooks/api/useTimesheets";
import type { 
  CreateTimesheetData, 
  UpdateTimesheetData,
  TimesheetFilters 
} from "@/types/api/timesheet.types";

export const useTimesheetData = (filters?: TimesheetFilters) => {
  const { data: response, isLoading, refetch } = useTimesheets(filters);
  const createMutation = useCreateTimesheet();
  const updateMutation = useUpdateTimesheet();
  const approveMutation = useApproveTimesheet();
  const bulkApproveMutation = useBulkApproveTimesheets();
  const deleteMutation = useDeleteTimesheet();

  const timesheets = response?.data || [];

  const handleCreateTimesheet = async (timesheetData: CreateTimesheetData): Promise<void> => {
    await createMutation.mutateAsync(timesheetData);
    refetch();
  };

  const handleUpdateTimesheet = async (timesheetId: number, timesheetData: UpdateTimesheetData): Promise<void> => {
    await updateMutation.mutateAsync({ id: timesheetId, data: timesheetData });
    refetch();
  };

  const handleApproveTimesheet = async (timesheetId: number): Promise<void> => {
    await approveMutation.mutateAsync(timesheetId);
    refetch();
  };

  const handleBulkApprove = async (timesheetIds: number[]): Promise<void> => {
    await bulkApproveMutation.mutateAsync({ timesheet_ids: timesheetIds });
    refetch();
  };

  const handleDeleteTimesheet = async (timesheetId: number): Promise<void> => {
    await deleteMutation.mutateAsync(timesheetId);
    refetch();
  };

  return {
    timesheets,
    isLoading: isLoading || createMutation.isPending || updateMutation.isPending || 
               approveMutation.isPending || bulkApproveMutation.isPending || deleteMutation.isPending,
    handleCreateTimesheet,
    handleUpdateTimesheet,
    handleApproveTimesheet,
    handleBulkApprove,
    handleDeleteTimesheet,
    refetch,
  };
};