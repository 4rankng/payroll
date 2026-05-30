import { format } from "date-fns";
import { useState, useEffect, useCallback } from 'react';
import { toast } from '@/components/ui/sonner';
import { ApprovalItem } from '@/types/approval';
import { timesheetService } from '@/services/api/timesheet.service';
import { Timesheet } from '@/types/api/timesheet.types';

export const useApprovalData = () => {
  const [approvalItems, setApprovalItems] = useState<ApprovalItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  // Fetch pending approvals from timesheet service
  const fetchApprovals = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await timesheetService.getTimesheets({
        status: 'pending_approval',
        pageSize: 100 // Get a reasonable number of pending approvals
      });

      // Extract timesheets from response
      const timesheets = Array.isArray(response.data) ? response.data : [];

      // Transform timesheet data to approval items
      const transformedApprovals: ApprovalItem[] = timesheets.map(transformTimesheetToApprovalItem);
      setApprovalItems(transformedApprovals);
    } catch (error) {
      console.error('Failed to fetch approvals:', error);
      toast({
        title: 'Lỗi',
        description: 'Không thể tải danh sách phê duyệt.',
        variant: 'destructive'
      });
      setApprovalItems([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Transform timesheet to approval item format
  const transformTimesheetToApprovalItem = (timesheet: Timesheet): ApprovalItem => {
    return {
      id: timesheet.id.toString(),
      type: 'timesheet',
      title: `Timesheet #${timesheet.id}`,
      description: `${timesheet.employeeName} - ${timesheet.hours_worked} giờ - ${format(new Date(timesheet.date), 'dd/MM/yyyy')}`,
      submittedBy: {
        name: timesheet.employeeName,
        role: 'Partner',
        avatar: undefined
      },
      submittedAt: timesheet.created_at,
      priority: 'medium' as const,
      status: timesheet.status as 'pending' | 'approved' | 'rejected',
      details: {
        hours_worked: timesheet.hours_worked,
        paytype: timesheet.paytype,
        amount: timesheet.amount,
        date: timesheet.date,
        project_name: timesheet.projectName
      }
    };
  };

  // Load approvals on hook initialization
  useEffect(() => {
    fetchApprovals();
  }, [fetchApprovals]);

  const handleApprove = async (itemIds: string[], comment?: string) => {
    setIsLoading(true);
    const approvedIds: string[] = [];
    const failedIds: string[] = [];

    try {
      // Process each item individually
      for (const itemId of itemIds) {
        try {
          const timesheetId = parseInt(itemId, 10);
          await timesheetService.approveTimesheet(timesheetId);
          approvedIds.push(itemId);
        } catch (error) {
          console.error(`Failed to approve timesheet ${itemId}:`, error);
          failedIds.push(itemId);
        }
      }

      // Update local state for approved items
      setApprovalItems(prev =>
        prev.map(item =>
          approvedIds.includes(item.id)
            ? { ...item, status: 'approved' as const }
            : item
        )
      );

      // Show appropriate toast message
      if (approvedIds.length > 0) {
        toast({
          title: 'Thành công',
          description: `Đã phê duyệt ${approvedIds.length} yêu cầu${failedIds.length > 0 ? `, ${failedIds.length} yêu cầu thất bại` : ''}`,
          variant: failedIds.length > 0 ? 'destructive' : 'default'
        });
      } else {
        toast({
          title: 'Lỗi',
          description: 'Không thể phê duyệt yêu cầu nào',
          variant: 'destructive'
        });
      }
    } catch (error) {
      toast({
        title: 'Lỗi',
        description: 'Không thể phê duyệt yêu cầu',
        variant: 'destructive'
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleReject = async (itemIds: string[], reason: string) => {
    setIsLoading(true);
    const rejectedIds: string[] = [];
    const failedIds: string[] = [];

    try {
      // Process each item individually
      for (const itemId of itemIds) {
        try {
          const timesheetId = parseInt(itemId, 10);
          await timesheetService.rejectTimesheet(timesheetId, {
            rejection_reason: reason
          });
          rejectedIds.push(itemId);
        } catch (error) {
          console.error(`Failed to reject timesheet ${itemId}:`, error);
          failedIds.push(itemId);
        }
      }

      // Update local state for rejected items
      setApprovalItems(prev =>
        prev.map(item =>
          rejectedIds.includes(item.id)
            ? { ...item, status: 'rejected' as const }
            : item
        )
      );

      // Show appropriate toast message
      if (rejectedIds.length > 0) {
        toast({
          title: 'Thành công',
          description: `Đã loại ${rejectedIds.length} yêu cầu${failedIds.length > 0 ? `, ${failedIds.length} yêu cầu thất bại` : ''}`,
          variant: failedIds.length > 0 ? 'destructive' : 'default'
        });
      } else {
        toast({
          title: 'Lỗi',
          description: 'Không thể loại yêu cầu nào',
          variant: 'destructive'
        });
      }
    } catch (error) {
      toast({
        title: 'Lỗi',
        description: 'Không thể loại yêu cầu',
        variant: 'destructive'
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleSetReviewing = async (itemIds: string[]) => {
    // For timesheets, there's no separate "reviewing" API call
    // This would typically just update the local state to show reviewing status
    setApprovalItems(prev =>
      prev.map(item =>
        itemIds.includes(item.id)
          ? { ...item, status: 'reviewing' as const }
          : item
      )
    );

    toast({
      title: 'Thành công',
      description: `Đã đánh dấu ${itemIds.length} yêu cầu đang xem xét`,
    });
  };

  return {
    approvalItems,
    isLoading,
    handleApprove,
    handleReject,
    handleSetReviewing,
    refetch: fetchApprovals, // Add refetch capability
  };
};
