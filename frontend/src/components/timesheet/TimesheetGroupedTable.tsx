import { useState, useMemo, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Card, CardContent } from '@/components/ui/card';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import {
  ChevronRight,
  ChevronDown,
  Calendar,
  CheckSquare,
  XCircle,
  Users,
  Clock,
  Star,
  Banknote,
  Trash2
} from 'lucide-react';
import { Timesheet } from '@/types/api/timesheet.types';
import { showErrorNotification } from '@/utils/error-handler';
import { TimesheetEntryModal } from './components/TimesheetEntryModal';
import { GroupBulkRejectDialog } from './components/GroupBulkRejectDialog';
import { TimesheetEntryCard } from './components/TimesheetEntryCard';
import {
  groupTimesheetsByEmployeeDate,
  sortGroupedTimesheets,
  getGroupedStatusBadgeVariant,
  formatStatusBreakdown,
  getGroupTimesheetIds,
  getPendingTimesheetIds,
  canBulkApprove,
  canBulkReject,
  type GroupedTimesheet
} from './utils/timesheetGrouping';
import {
  getStatusBadge,
  getPaytypeText,
  formatCurrency,
  formatDate,
  formatDateWithWeekday,
  getVietnameseWeekdayInfo
} from './utils/timesheetHelpers';
import { createGroupBulkApprovalContent } from '@/utils/timesheetBulkHelpers';
import { cn } from '@/lib/utils';

interface TimesheetGroupedTableProps {
  timesheets: Timesheet[];
  isLoading?: boolean;
  onView: (timesheet: Timesheet) => void;
  onEdit: (timesheet: Timesheet) => void;
  onApprove: (timesheet: Timesheet) => void;
  onReject: (timesheet: Timesheet, reason: string) => void;
  onDelete: (timesheet: Timesheet) => void;
  onBulkApprove?: (ids: number[]) => Promise<void>;
  onBulkReject?: (ids: number[], reason: string) => Promise<void>;
  onRequestEdit?: (timesheet: Timesheet, onSuccess?: () => Promise<void> | void) => Promise<void> | void;
  requestingTimesheetId?: number | null;
}

export function TimesheetGroupedTable({
  timesheets,
  isLoading = false,
  onView,
  onEdit,
  onApprove,
  onReject,
  onDelete,
  onBulkApprove,
  onBulkReject,
  onRequestEdit,
  requestingTimesheetId = null
}: TimesheetGroupedTableProps) {
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [hiddenGroups, setHiddenGroups] = useState<Set<string>>(new Set());
  const [selectedTimesheet, setSelectedTimesheet] = useState<Timesheet | null>(null);
  const [isEntryModalOpen, setIsEntryModalOpen] = useState(false);
  const [bulkRejectGroup, setBulkRejectGroup] = useState<GroupedTimesheet | null>(null);
  const [isBulkRejectDialogOpen, setIsBulkRejectDialogOpen] = useState(false);
  const [bulkApproveGroup, setBulkApproveGroup] = useState<GroupedTimesheet | null>(null);
  const [isBulkApproveDialogOpen, setIsBulkApproveDialogOpen] = useState(false);

  const groupedTimesheets = useMemo(() => {
    const grouped = groupTimesheetsByEmployeeDate(timesheets);
    return sortGroupedTimesheets(grouped);
  }, [timesheets]);

  const toggleGroup = (groupKey: string) => {
    const newExpanded = new Set(expandedGroups);
    if (newExpanded.has(groupKey)) {
      newExpanded.delete(groupKey);
    } else {
      newExpanded.add(groupKey);
    }
    setExpandedGroups(newExpanded);
  };

  const handleIndividualEntryClick = (timesheet: Timesheet) => {
    setSelectedTimesheet(timesheet);
    setIsEntryModalOpen(true);
  };

  const handleEntryModalClose = () => {
    setIsEntryModalOpen(false);
    setSelectedTimesheet(null);
  };

  const handleRequestEditWithClose = useCallback((timesheet: Timesheet) => {
    if (!onRequestEdit) {
      return;
    }
    // Call onRequestEdit with success callback to close modal
    void Promise.resolve(onRequestEdit(timesheet, handleEntryModalClose)).catch((error) => { showErrorNotification(error); });
  }, [onRequestEdit]);

  const handleBulkGroupApproveClick = (group: GroupedTimesheet) => {
    setBulkApproveGroup(group);
    setIsBulkApproveDialogOpen(true);
  };

  const handleBulkGroupApprove = async (group: GroupedTimesheet) => {
    if (onBulkApprove && canBulkApprove(group)) {
      // Only send pending approval timesheet IDs, excluding already-approved or rejected ones
      const ids = getPendingTimesheetIds(group);
      await onBulkApprove(ids);
      setIsBulkApproveDialogOpen(false);
      setBulkApproveGroup(null);
    }
  };

  const handleBulkGroupRejectClick = (group: GroupedTimesheet) => {
    setBulkRejectGroup(group);
    setIsBulkRejectDialogOpen(true);
  };

  const handleBulkGroupReject = async (group: GroupedTimesheet, reason: string) => {
    if (onBulkReject && canBulkReject(group)) {
      const ids = getGroupTimesheetIds(group);
      await onBulkReject(ids, reason);
    }
  };

  const handleBulkRejectDialogClose = () => {
    setIsBulkRejectDialogOpen(false);
    setBulkRejectGroup(null);
  };

  const handleBulkApproveDialogClose = () => {
    setIsBulkApproveDialogOpen(false);
    setBulkApproveGroup(null);
  };

  const getGroupKey = (group: GroupedTimesheet) => `${group.employeeId}_${group.date}`;

  const handleHideGroup = useCallback((group: GroupedTimesheet) => {
    setHiddenGroups(prev => {
      const next = new Set(prev);
      next.add(getGroupKey(group));
      return next;
    });
  }, []);

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <Card key={i} className="p-4">
            <div className="flex items-center gap-4 animate-pulse">
              <div className="w-10 h-10 bg-muted rounded-full" />
              <div className="flex-1 space-y-2">
                <div className="h-4 bg-muted rounded w-48" />
                <div className="h-3 bg-muted rounded w-32" />
              </div>
              <div className="space-y-2">
                <div className="h-4 bg-muted rounded w-20" />
                <div className="h-3 bg-muted rounded w-16" />
              </div>
            </div>
          </Card>
        ))}
      </div>
    );
  }

  if (groupedTimesheets.length === 0) {
    return (
      <div className="text-center py-12">
        <Calendar className="mx-auto h-12 w-12 text-muted-foreground/50" />
        <h3 className="mt-4 typography-title-large">Không tìm thấy dữ liệu bảng công</h3>
        <p className="mt-2 typography-body-medium text-muted-foreground">
          Hãy kiểm tra lại bộ lọc hoặc thử tìm kiếm khác.
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-2">
        {groupedTimesheets
          .filter((group) => !hiddenGroups.has(getGroupKey(group)))
          .map((group) => {
          const groupKey = getGroupKey(group);
          const isExpanded = expandedGroups.has(groupKey);
          const statusConfig = getGroupedStatusBadgeVariant(group.aggregatedStatus);
          const weekdayInfo = getVietnameseWeekdayInfo(group.date);

          return (
            <Card key={groupKey} className="overflow-hidden">
              {/* Summary Row - Responsive Layout */}
              <div
                className={cn(
                  "p-4 cursor-pointer transition-colors",
                  !weekdayInfo?.isSaturday && !weekdayInfo?.isSunday && "hover:bg-muted/30",
                  weekdayInfo?.isSaturday && "bg-slate-100 hover:bg-slate-200 backdrop-blur-sm",
                  weekdayInfo?.isSunday && "bg-slate-100 hover:bg-slate-200 backdrop-blur-sm"
                )}
                onClick={() => toggleGroup(groupKey)}
              >
                {/* Desktop Layout */}
                <div className="hidden md:flex items-center gap-4">
                  {/* Expand/Collapse Icon */}
                  <div className="flex-shrink-0">
                    {isExpanded ? (
                      <ChevronDown className="w-5 h-5 text-muted-foreground" />
                    ) : (
                      <ChevronRight className="w-5 h-5 text-muted-foreground" />
                    )}
                  </div>

                  {/* Employee Info */}
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <UserAvatar
                      email={group.employeeCode}
                      name={group.employeeName}
                      size="sm"
                    />
                    <div className="min-w-0">
                      <p className="typography-body-medium truncate">{group.employeeName}</p>
                      <p className="typography-body-small text-muted-foreground truncate">
                        {group.employeeCode}
                      </p>
                    </div>
                  </div>

                  {/* Date */}
                  <div className="text-center min-w-[90px]">
                    <p className="typography-body-medium">
                      {formatDateWithWeekday(group.date)}
                    </p>
                    <p className="typography-body-small text-muted-foreground">
                      {group.entries.length} mục
                    </p>
                  </div>

                  {/* Total Hours */}
                  <div className="text-center min-w-[100px]">
                    <div className="flex items-center justify-center gap-1.5">
                      <Clock className="w-4 h-4 text-muted-foreground" />
                      <Badge variant="secondary" className="typography-data font-bold text-green-700 bg-green-50 px-3">
                        {group.totalHours}h
                      </Badge>
                    </div>
                    <p className="typography-body-small text-muted-foreground mt-1">
                      {group.hasMultipleProjects && (
                        <Users className="w-3 h-3 inline mr-1" />
                      )}
                      Ca làm việc
                    </p>
                  </div>

                  {/* Total Amount */}
                  <div className="text-right min-w-[140px]">
                    <div className="flex items-center justify-end gap-1.5">
                      <Banknote className="w-4 h-4 text-muted-foreground" />
                      <span className="typography-currency font-bold tabular-nums">{formatCurrency(group.totalAmount)}đ</span>
                    </div>
                    <p className="typography-body-small text-muted-foreground mt-1">Thành tiền</p>
                  </div>

                  {/* Status + Hide */}
                  <div className="min-w-[120px] flex items-center justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleHideGroup(group);
                      }}
                      title="Ẩn dòng này khỏi danh sách"
                      aria-label="Ẩn dòng bảng công này"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                    <Badge
                      variant={statusConfig.variant}
                      className={`
                        ${statusConfig.color === 'green' ? 'bg-green-100 text-green-800' : ''}
                        ${statusConfig.color === 'yellow' ? 'bg-yellow-100 text-yellow-800' : ''}
                        ${statusConfig.color === 'red' ? 'bg-red-100 text-red-800' : ''}
                        ${statusConfig.color === 'orange' ? 'bg-orange-100 text-orange-800' : ''}
                      `}
                      title={group.aggregatedStatus === 'mixed' ? formatStatusBreakdown(group.statusBreakdown) : undefined}
                    >
                      {statusConfig.text}
                    </Badge>
                  </div>
                </div>

                {/* Mobile Layout */}
                <div className="md:hidden">
                  <div className="flex items-start gap-2">
                    {/* Expand/Collapse Icon */}
                    <div className="flex-shrink-0 mt-1">
                      {isExpanded ? (
                        <ChevronDown className="w-4 h-4 text-muted-foreground" />
                      ) : (
                        <ChevronRight className="w-4 h-4 text-muted-foreground" />
                      )}
                    </div>

                    {/* Employee Avatar */}
                    <UserAvatar
                      email={group.employeeCode}
                      name={group.employeeName}
                      size="sm"
                      className="flex-shrink-0"
                    />

                    {/* Main Content */}
                    <div className="flex-1 min-w-0 space-y-2">
                      {/* Employee Info & Date Row */}
                      <div className="flex justify-between items-start gap-2">
                        <div className="min-w-0 flex-1">
                          <p className="typography-body-medium font-semibold truncate">{group.employeeName}</p>
                          <p className="typography-body-small text-muted-foreground truncate">
                            {group.employeeCode}
                          </p>
                        </div>
                        <Badge
                          variant={statusConfig.variant}
                          className={cn(
                            'typography-body-small flex-shrink-0',
                            statusConfig.color === 'green' && 'bg-green-100 text-green-800',
                            statusConfig.color === 'yellow' && 'bg-yellow-100 text-yellow-800',
                            statusConfig.color === 'red' && 'bg-red-100 text-red-800',
                            statusConfig.color === 'orange' && 'bg-orange-100 text-orange-800'
                          )}
                          title={group.aggregatedStatus === 'mixed' ? formatStatusBreakdown(group.statusBreakdown) : undefined}
                        >
                          {statusConfig.text}
                        </Badge>
                      </div>

                      {/* Date and entries count */}
                      <div className="flex items-center gap-2 typography-body-small text-muted-foreground">
                        <Calendar className="w-3.5 h-3.5" />
                        <span>{formatDateWithWeekday(group.date)}</span>
                        <span>•</span>
                        <span>{group.entries.length} bản ghi</span>
                        {group.hasMultipleProjects && (
                          <>
                            <span>•</span>
                            <Users className="w-3 h-3 inline" />
                            <span>Nhiều dự án</span>
                          </>
                        )}
                      </div>

                      {/* Highlighted Stats Row */}
                      <div className="flex gap-2">
                        {/* Hours Card */}
                        <div className="flex-1 bg-green-50 border border-green-200 rounded-xl p-2.5">
                          <div className="flex items-center gap-1.5 mb-0.5">
                            <Clock className="w-3.5 h-3.5 text-green-600" />
                            <span className="typography-label-small text-green-700">Ca làm việc</span>
                          </div>
                          <p className="typography-title-medium font-bold text-green-800">
                            {group.totalHours}h
                          </p>
                        </div>

                        {/* Amount Card */}
                        <div className="flex-1 bg-primary/5 border border-primary/20 rounded-xl p-2.5">
                          <div className="flex items-center gap-1.5 mb-0.5">
                            <Banknote className="w-3.5 h-3.5 text-primary" />
                            <span className="typography-label-small text-primary">Thành tiền</span>
                          </div>
                          <p className="typography-title-medium font-bold text-primary tabular-nums">
                            {formatCurrency(group.totalAmount)}đ
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Expanded Content */}
              {isExpanded && (
                <div className="border-t bg-muted/20">
                  {/* Group Actions Header */}
                  <div className="px-4 py-2 border-b bg-muted/10 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-2">
                    <p className="typography-body-medium typography-label-medium text-muted-foreground">
                      <span className="hidden sm:inline">Chi tiết các mục - {group.employeeName} ({formatDate(group.date)})</span>
                      <span className="sm:hidden">Chi tiết các mục</span>
                    </p>
                    <div className="flex gap-2">
                      {canBulkApprove(group) && (
                        <Button
                          size="sm"
                          variant="outline"
                          className="text-green-700 border-green-200 hover:bg-green-50 flex-1 sm:flex-initial"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleBulkGroupApproveClick(group);
                          }}
                        >
                          <CheckSquare className="w-3 h-3 mr-1" />
                          <span className="hidden sm:inline">Duyệt tất cả</span>
                          <span className="sm:hidden">Duyệt</span>
                        </Button>
                      )}
                      {canBulkReject(group) && (
                        <Button
                          size="sm"
                          variant="outline"
                          className="text-orange-700 border-orange-200 hover:bg-orange-50 flex-1 sm:flex-initial"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleBulkGroupRejectClick(group);
                          }}
                        >
                          <XCircle className="w-3 h-3 mr-1" />
                          <span className="hidden sm:inline">Loại tất cả</span>
                          <span className="sm:hidden">Loại</span>
                        </Button>
                      )}
                    </div>
                  </div>

                  {/* Individual Entries */}
                  <div className="p-4 space-y-2">
                    {group.entries.map((entry) => (
                      <TimesheetEntryCard
                        key={entry.id}
                        entry={entry}
                        variant="compact"
                        onClick={() => handleIndividualEntryClick(entry)}
                      />
                    ))}
                  </div>
                </div>
              )}
            </Card>
          );
        })}
      </div>

      {/* Detail Sheet */}
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
          onRequestEdit={handleRequestEditWithClose}
          onRequestEditSuccess={handleEntryModalClose}
          requestingTimesheetId={requestingTimesheetId}
        />
      )}

      {/* Group Bulk Reject Dialog */}
      <GroupBulkRejectDialog
        isOpen={isBulkRejectDialogOpen}
        onClose={handleBulkRejectDialogClose}
        group={bulkRejectGroup}
        onConfirm={handleBulkGroupReject}
        isLoading={false}
      />

      {/* Group Bulk Approve Dialog */}
      {bulkApproveGroup && (
        <ConfirmDialog
          open={isBulkApproveDialogOpen}
          onOpenChange={handleBulkApproveDialogClose}
          title="Duyệt nhóm bảng công"
          description={createGroupBulkApprovalContent(
            bulkApproveGroup.employeeName,
            bulkApproveGroup.employeeCode,
            bulkApproveGroup.date,
            bulkApproveGroup.entries
          )}
          confirmText="Duyệt nhóm"
          onConfirm={() => handleBulkGroupApprove(bulkApproveGroup)}
        />
      )}
    </>
  );
}
