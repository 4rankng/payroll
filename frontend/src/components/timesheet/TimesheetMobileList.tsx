import { useMemo, useCallback, useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Clock, Banknote, ClipboardEdit, ChevronRight, ChevronDown } from 'lucide-react';
import { Timesheet } from '@/types/api/timesheet.types';
import { TimesheetEntryModal } from './components/TimesheetEntryModal';
import { showErrorNotification } from '@/utils/error-handler';
import { groupTimesheetsByEmployeeDate, sortGroupedTimesheets } from './utils/timesheetGrouping';
import { formatCurrency, formatDateWithWeekday, getMergedStatusBadge, getPaytypeText } from './utils/timesheetHelpers';
import { cn } from '@/lib/utils';
import { AccentStripCard, type AccentColor } from '@/components/shared/AccentStripCard';
import { EmptyState } from '@/components/shared/EmptyState';
import { MobilePagination } from '@/components/shared/MobilePagination';
import type { MouseEvent } from 'react';
import { useTimesheetContext } from './TimesheetContext';

export function TimesheetMobileList() {
  const { state, actions, meta } = useTimesheetContext();
  const [selectedTimesheet, setSelectedTimesheet] = useState<Timesheet | null>(null);
  const [isEntryModalOpen, setIsEntryModalOpen] = useState(false);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());

  const groupedTimesheets = useMemo(() => {
    const grouped = groupTimesheetsByEmployeeDate(state.timesheets);
    return sortGroupedTimesheets(grouped);
  }, [state.timesheets]);

  const handleEntryClick = (timesheet: Timesheet) => {
    setSelectedTimesheet(timesheet);
    setIsEntryModalOpen(true);
  };

  const handleEntryModalClose = useCallback(() => {
    setIsEntryModalOpen(false);
    setSelectedTimesheet(null);
  }, []);

  const handleRequestEditWithClose = useCallback((timesheet: Timesheet) => {
    if (!actions.requestEdit) return;
    void Promise.resolve(actions.requestEdit(timesheet, handleEntryModalClose)).catch((error) => { showErrorNotification(error); });
  }, [actions.requestEdit, handleEntryModalClose]);

  const getRequestEditHandler = useCallback((timesheet: Timesheet) => {
    if (!actions.requestEdit) return undefined;
    return (event: MouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      void Promise.resolve(actions.requestEdit(timesheet)).catch((error) => { showErrorNotification(error); });
    };
  }, [actions.requestEdit]);

  const toggleGroup = useCallback((key: string) => {
    setExpandedGroups(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

  if (state.isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="bg-card rounded-xl border border-border p-4 animate-pulse space-y-2.5">
            <div className="flex justify-between">
              <div className="flex items-center gap-2.5">
                <div className="h-8 w-8 bg-muted rounded-full" />
                <div className="space-y-1.5">
                  <div className="h-3.5 bg-muted rounded w-28" />
                  <div className="h-3 bg-muted/60 rounded w-16" />
                </div>
              </div>
              <div className="h-5 bg-muted rounded-full w-16" />
            </div>
            <div className="flex gap-4 pt-1">
              <div className="h-3 bg-muted/60 rounded w-12" />
              <div className="h-3 bg-muted/60 rounded w-20" />
              <div className="h-3 bg-muted/60 rounded w-16 ml-auto" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (groupedTimesheets.length === 0) {
    return (
      <EmptyState
        icon={Clock}
        title="Không có dữ liệu"
        description="Thử điều chỉnh bộ lọc để xem kết quả"
      />
    );
  }

  return (
    <>
      <div className="space-y-2.5">
        {groupedTimesheets.map((group) => {
          const groupKey = `${group.employeeId}_${group.date}`;
          const isExpanded = expandedGroups.has(groupKey);
          const projectNames = [...new Set(group.entries.map(e => e.projectName))];
          const projectDisplay = projectNames.length > 1
            ? `${projectNames[0]} +${projectNames.length - 1}`
            : projectNames[0];

          const dominantStatus = group.entries.map(e => e.status).includes('pending_approval')
            ? 'pending_approval'
            : group.entries.map(e => e.status).includes('approved')
              ? 'approved'
              : group.entries[0]?.status;
          const dominantPayment = group.entries.map(e => e.payment_status).includes('paid') ? 'paid' : group.entries[0]?.payment_status;

          const accentColor: AccentColor = dominantPayment === 'paid'
            ? 'green'
            : dominantStatus === 'pending_approval'
              ? 'amber'
              : dominantStatus === 'approved'
                ? 'blue'
                : dominantStatus === 'rejected'
                  ? 'red'
                  : 'gray';

          return (
            <AccentStripCard
              key={groupKey}
              accentColor={accentColor}
            >
              <div
                className="px-4 pt-3.5 pb-3 cursor-pointer active:bg-muted/40 transition-colors touch-manipulation"
                onClick={() => {
                  if (group.entries.length === 1) handleEntryClick(group.entries[0]);
                  else toggleGroup(groupKey);
                }}
                role="button"
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    if (group.entries.length === 1) handleEntryClick(group.entries[0]);
                    else toggleGroup(groupKey);
                  }
                }}
                aria-expanded={group.entries.length > 1 ? isExpanded : undefined}
              >
                <div className="flex items-start justify-between gap-2 mb-2.5">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-foreground leading-tight truncate">{group.employeeName}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate">{projectDisplay}</p>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0 mt-0.5">
                    {getMergedStatusBadge(dominantStatus, dominantPayment)}
                    {group.entries.length > 1 ? (
                      <ChevronDown className={cn('h-3.5 w-3.5 text-muted-foreground/60 transition-transform', isExpanded && 'rotate-180')} />
                    ) : (
                      <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/60" />
                    )}
                  </div>
                </div>

                <div className="flex items-center gap-4 text-xs">
                  <div className="flex items-center gap-1.5">
                    <Clock className="h-3 w-3 text-muted-foreground shrink-0" />
                    <span className="font-semibold text-foreground tabular-nums">{group.totalHours}h</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Banknote className="h-3 w-3 text-muted-foreground shrink-0" />
                    <span className="font-semibold text-foreground tabular-nums">{formatCurrency(group.totalAmount)}đ</span>
                  </div>
                  <span className="text-muted-foreground ml-auto tabular-nums">{formatDateWithWeekday(group.date, 'short')}</span>
                </div>
              </div>

              {group.entries.length > 1 && isExpanded && (
                <div className="border-t border-border/60 divide-y divide-border/40 bg-muted/[0.03]">
                  {group.entries.map((entry) => (
                    <div
                      key={entry.id}
                      className="px-4 py-3 flex items-center justify-between gap-3 cursor-pointer active:bg-muted/40 touch-manipulation"
                      onClick={() => handleEntryClick(entry)}
                    >
                      <div className="min-w-0 flex-1">
                        <p className="text-xs font-medium text-foreground truncate">{entry.projectName}</p>
                        <div className="flex items-center gap-2 mt-0.5">
                          <p className="text-xs text-muted-foreground">{getPaytypeText(entry.paytype)}</p>
                          {entry.request_edit_id != null && (
                            <Badge variant="warning" className="inline-flex items-center gap-0.5 text-[10px] h-4 px-1.5">
                              <ClipboardEdit className="w-2.5 h-2.5" />
                              Yêu cầu sửa
                            </Badge>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <div className="text-right">
                          <p className="text-xs font-semibold text-foreground tabular-nums">{entry.hours_worked}h</p>
                          <p className="text-xs text-muted-foreground tabular-nums">{formatCurrency(entry.amount)}đ</p>
                        </div>
                        {getMergedStatusBadge(entry.status, entry.payment_status)}
                        <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50 shrink-0" />
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {group.entries.length === 1 && (() => {
                const entry = group.entries[0];
                const showRequestEdit = actions.requestEdit &&
                  entry.status === 'approved' &&
                  entry.payment_status === 'pending' &&
                  !entry.allowed_edit &&
                  entry.request_edit_id == null;
                if (!showRequestEdit) return null;
                return (
                  <div className="border-t border-border/60 px-4 py-2.5">
                    <Button
                      onClick={getRequestEditHandler(entry)}
                      variant="outline"
                      size="sm"
                      className="w-full h-8 text-xs"
                      disabled={meta.requestingTimesheetId === entry.id}
                    >
                      {meta.requestingTimesheetId === entry.id ? 'Đang gửi...' : 'Yêu cầu chỉnh sửa'}
                    </Button>
                  </div>
                );
              })()}
            </AccentStripCard>
          );
        })}
      </div>

      {state.pagination && actions.pageChange && (
        <MobilePagination pagination={state.pagination} onPageChange={actions.pageChange} />
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
    </>
  );
}
