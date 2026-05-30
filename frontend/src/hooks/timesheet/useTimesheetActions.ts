import { useNavigate } from 'react-router-dom';
import { toast } from '@/components/ui/sonner';
import { useApproveTimesheet, useRejectTimesheet, useBulkApproveTimesheets } from '@/hooks/api/useTimesheets';
import type { Timesheet } from '@/types/api/timesheet.types';

export function useTimesheetActions() {
  const navigate = useNavigate();

  const approveTimesheetMutation = useApproveTimesheet();
  const rejectTimesheetMutation = useRejectTimesheet();
  const bulkApproveMutation = useBulkApproveTimesheets();

  const handleView = (timesheet: Timesheet) => {
    // Navigate to timesheet detail view
    navigate(`/partner/timesheets/${timesheet.id}`);
  };

  const handleEdit = (timesheet: Timesheet) => {
    // Navigate to timesheet edit page
    navigate(`/partner/timesheets/${timesheet.id}/edit`);
  };

  const handleApprove = async (timesheet: Timesheet) => {
    try {
      await approveTimesheetMutation.mutateAsync(timesheet.id);
      toast({
        title: 'Duyệt thành công',
        description: `Đã duyệt bảng công cho ${timesheet.employeeName}.`,
      });
    } catch (error) {
      toast({
        title: 'Lỗi',
        description: 'Không thể duyệt bảng công. Vui lòng thử lại.',
        variant: 'destructive',
      });
    }
  };

  const handleReject = async (timesheet: Timesheet) => {
    try {
      await rejectTimesheetMutation.mutateAsync(timesheet.id);
      toast({
        title: 'Loại thành công',
        description: `Đã loại bảng công cho ${timesheet.employeeName}.`,
      });
    } catch (error) {
      toast({
        title: 'Lỗi',
        description: 'Không thể loại bảng công. Vui lòng thử lại.',
        variant: 'destructive',
      });
    }
  };

  const handleBulkApprove = async (timesheetIds: number[]) => {
    try {
      await bulkApproveMutation.mutateAsync(timesheetIds);
      toast({
        title: 'Duyệt hàng loạt thành công',
        description: `Đã duyệt ${timesheetIds.length} bảng công.`,
      });
    } catch (error) {
      toast({
        title: 'Lỗi',
        description: 'Không thể duyệt hàng loạt. Vui lòng thử lại.',
        variant: 'destructive',
      });
    }
  };

  const handleExport = () => {
    // Create a blob for CSV export
    const csvContent = "data:text/csv;charset=utf-8,";
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement("a");
    link.setAttribute("href", encodedUri);
    link.setAttribute("download", `timesheets-${new Date().toISOString().split('T')[0]}.csv`);
    document.body.appendChild(link);
    link.click();
    // Safely remove the link
    if (link.parentNode) {
      link.parentNode.removeChild(link);
    }
  };

  const handleImport = () => {
    // Create file input for import
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.xls,.xlsx,.csv';
    input.onchange = (event) => {
      const file = (event.target as HTMLInputElement)?.files?.[0];
      if (file) {
        // Handle file upload logic here
      }
    };
    input.click();
  };

  return {
    handleView,
    handleEdit,
    handleApprove,
    handleReject,
    handleBulkApprove,
    handleExport,
    handleImport,
  };
}
