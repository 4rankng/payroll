import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Clock, Banknote, CheckCircle, XCircle, Eye, Star, Calendar as CalendarIcon, Trash2 } from 'lucide-react';
import { Timesheet } from '@/types/api/timesheet.types';
import { cn } from '@/lib/utils';
import { getPaytypeText, formatDateWithWeekday } from '../utils/timesheetHelpers';
import { formatCurrency } from '@/utils/formatters';
import { getStatusColor, getStatusIcon, getStatusLabel } from '../utils/timesheetStatusHelpers';

interface TimesheetEntryCardProps {
  entry: Timesheet;
  onClick?: () => void;
  onApprove?: () => void;
  onReject?: () => void;
  showActions?: boolean;
  isLoading?: boolean;
  variant?: 'default' | 'compact' | 'timeline';
  className?: string;
}

export function TimesheetEntryCard({
  entry,
  onClick,
  onApprove,
  onReject,
  showActions = false,
  isLoading = false,
  variant = 'default',
  className
}: TimesheetEntryCardProps) {
  const statusColor = getStatusColor(entry.status);
  const StatusIcon = getStatusIcon(entry.status);

  const handleClick = (e: React.MouseEvent) => {
    if (onClick && !isLoading) {
      onClick();
    }
  };

  const handleApprove = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onApprove && !isLoading) {
      onApprove();
    }
  };

  const handleReject = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onReject && !isLoading) {
      onReject();
    }
  };

  // Compact variant for grouped table
  if (variant === 'compact') {
    return (
      <div
        className={cn(
          'relative bg-background rounded-xl p-3 border transition-all',
          'cursor-pointer hover:shadow-sm',
          statusColor.border,
          isLoading && 'opacity-50 pointer-events-none',
          className
        )}
        onClick={handleClick}
      >
        {/* Status indicator bar */}
        <div className={cn('absolute left-0 top-0 bottom-0 w-1 rounded-l-lg', statusColor.bg)} />

        {/* Desktop Layout */}
        <div className="hidden md:grid grid-cols-12 gap-3 items-center typography-body-medium pl-2">
          {/* Project */}
          <div className="col-span-3 flex items-center gap-2 min-w-0">
            <StatusIcon className={cn('w-4 h-4 shrink-0', statusColor.text)} />
            <span className="font-medium truncate">{entry.projectName}</span>
          </div>

          {/* Paytype */}
          <div className="col-span-3 flex items-center gap-2">
            <Badge variant="outline" className="typography-body-small">
              {getPaytypeText(entry.paytype)}
            </Badge>
            {entry.force_payroll && (
              <Star className="h-4 w-4 text-yellow-700 fill-yellow-600 shrink-0" />
            )}
          </div>

          {/* Hours */}
          <div className="col-span-2 flex items-center justify-center gap-1.5">
            <Clock className="w-4 h-4 text-muted-foreground" />
            <Badge variant="secondary" className="font-semibold text-green-700 bg-green-50">
              {entry.hours_worked}h
            </Badge>
          </div>

          {/* Amount */}
          <div className="col-span-3 flex items-center justify-end gap-1.5">
            <Banknote className="w-4 h-4 text-muted-foreground" />
            <span className="font-semibold tabular-nums">{formatCurrency(entry.amount)}</span>
          </div>

          {/* Actions */}
          {showActions && (
            <div className="col-span-1 flex justify-end">
              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                <Eye className="h-4 w-4" />
              </Button>
            </div>
          )}
        </div>

        {/* Mobile Layout */}
        <div className="md:hidden space-y-2 pl-2">
          {/* Project & Status */}
          <div className="flex justify-between items-start gap-2">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <StatusIcon className={cn('w-4 h-4 shrink-0', statusColor.text)} />
              <span className="typography-body-medium font-medium truncate">{entry.projectName}</span>
            </div>
            {entry.force_payroll && (
              <Star className="h-4 w-4 text-yellow-700 fill-yellow-600 shrink-0" />
            )}
          </div>

          {/* Details */}
          <div className="flex justify-between items-center typography-body-medium">
            <div className="flex gap-3">
              {/* Paytype */}
              <Badge variant="outline" className="typography-body-small">
                {getPaytypeText(entry.paytype)}
              </Badge>

              {/* Hours */}
              <div className="flex items-center gap-1">
                <Clock className="w-3.5 h-3.5 text-muted-foreground" />
                <Badge variant="secondary" className="typography-body-small font-semibold text-green-700 bg-green-50">
                  {entry.hours_worked}h
                </Badge>
              </div>
            </div>

            {/* Amount */}
            <div className="flex items-center gap-1">
              <Banknote className="w-3.5 h-3.5 text-muted-foreground" />
              <span className="font-semibold tabular-nums typography-body-small">
                {formatCurrency(entry.amount)}
              </span>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Timeline variant for modal
  if (variant === 'timeline') {
    return (
      <div
        className={cn(
          'relative bg-background rounded-xl p-4 border transition-all',
          'cursor-pointer hover:border-primary/30',
          statusColor.border,
          isLoading && 'opacity-50 pointer-events-none',
          className
        )}
        onClick={handleClick}
      >
        {/* Status indicator bar */}
        <div className={cn('absolute left-0 top-0 bottom-0 w-1.5 rounded-l-lg', statusColor.bg)} />

        <div className="space-y-3 pl-2">
          {/* Header: Project & Status */}
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <StatusIcon className={cn('w-5 h-5 shrink-0', statusColor.text)} />
              <div className="min-w-0">
                <p className="typography-body-large font-semibold truncate">{entry.projectName}</p>
                <p className="typography-body-small text-muted-foreground">
                  {getPaytypeText(entry.paytype)}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {entry.force_payroll && (
                <Star className="h-4 w-4 text-yellow-700 fill-yellow-600" />
              )}
              <Badge variant="outline" className={cn('typography-body-small', statusColor.badge)}>
                {getStatusLabel(entry.status)}
              </Badge>
            </div>
          </div>

          {/* Metrics */}
          <div className="grid grid-cols-2 gap-3">
            <div className="flex items-center gap-2 bg-muted/30 rounded-xl p-2">
              <Clock className="w-4 h-4 text-muted-foreground" />
              <div>
                <p className="typography-body-small text-muted-foreground">Ca làm việc</p>
                <p className="typography-data font-semibold text-green-700">{entry.hours_worked} giờ</p>
              </div>
            </div>
            <div className="flex items-center gap-2 bg-muted/30 rounded-xl p-2">
              <Banknote className="w-4 h-4 text-muted-foreground" />
              <div>
                <p className="typography-body-small text-muted-foreground">Thành tiền</p>
                <p className="typography-currency font-semibold tabular-nums">
                  {formatCurrency(entry.amount)}
                </p>
              </div>
            </div>
          </div>

          {/* Paid Amount if exists */}
          {entry.paid_amount && entry.paid_amount > 0 && (
            <div className="flex items-center justify-between p-2 bg-emerald-50 border border-emerald-200 rounded-xl">
              <span className="typography-body-small text-emerald-900">Đã tạm ứng:</span>
              <span className="typography-currency font-semibold text-emerald-700 tabular-nums">
                {formatCurrency(entry.paid_amount)}
              </span>
            </div>
          )}

          {/* Quick Actions */}
          {showActions && entry.status === 'pending_approval' && (
            <div className="flex gap-2 pt-2 border-t">
              <Button
                variant="outline"
                size="sm"
                onClick={handleReject}
                disabled={isLoading}
                className="flex-1 border-red-200 text-red-700 hover:bg-red-50"
              >
                <XCircle className="w-4 h-4 mr-1" />
                Loại
              </Button>
              <Button
                variant="default"
                size="sm"
                onClick={handleApprove}
                disabled={isLoading}
                className="flex-1 bg-green-600 hover:bg-green-700"
              >
                <CheckCircle className="w-4 h-4 mr-1" />
                Duyệt
              </Button>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Default variant - full featured card
  return (
    <div
      className={cn(
        'relative bg-background rounded-xl p-4 border transition-all',
        'cursor-pointer hover:shadow-sm hover:border-primary/40',
        statusColor.border,
        isLoading && 'opacity-50 pointer-events-none',
        className
      )}
      onClick={handleClick}
    >
      {/* Status indicator bar */}
      <div className={cn('absolute left-0 top-0 bottom-0 w-1.5 rounded-l-lg', statusColor.bg)} />

      <div className="space-y-4 pl-2">
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <StatusIcon className={cn('w-6 h-6 shrink-0', statusColor.text)} />
            <div className="min-w-0">
              <p className="typography-title-medium font-semibold truncate">{entry.projectName}</p>
              <p className="typography-body-medium text-muted-foreground">{entry.employeeName}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {entry.force_payroll && (
              <Star className="h-5 w-5 text-yellow-700 fill-yellow-600" />
            )}
            <Badge variant="outline" className={cn('typography-body-medium', statusColor.badge)}>
              {getStatusLabel(entry.status)}
            </Badge>
          </div>
        </div>

        {/* Date */}
        <div className="flex items-center gap-2 text-muted-foreground typography-body-small">
          <CalendarIcon className="w-4 h-4" />
          <span>Ngày làm:</span>
          <span className="font-medium text-foreground">
            {formatDateWithWeekday(entry.date)}
          </span>
        </div>

        {/* Metrics Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="space-y-1">
            <p className="typography-label-medium text-muted-foreground">Loại ca</p>
            <Badge variant="secondary" className="typography-body-small">
              {getPaytypeText(entry.paytype)}
            </Badge>
          </div>
          <div className="space-y-1">
            <p className="typography-label-medium text-muted-foreground">Ca làm việc</p>
            <div className="flex items-center gap-1.5">
              <Clock className="w-4 h-4 text-muted-foreground" />
              <span className="typography-data font-semibold text-green-700">{entry.hours_worked}h</span>
            </div>
          </div>
          <div className="space-y-1">
            <p className="typography-label-medium text-muted-foreground">Đơn giá</p>
            <span className="typography-data font-medium tabular-nums">
              {formatCurrency(entry.payrate)}/{entry.projectIsFlexible ? "ca" : "giờ"}
            </span>
          </div>
          <div className="space-y-1">
            <p className="typography-label-medium text-muted-foreground">Thành tiền</p>
            <div className="flex items-center gap-1.5">
              <Banknote className="w-4 h-4 text-muted-foreground" />
              <span className="typography-currency font-semibold tabular-nums">
                {formatCurrency(entry.amount)}
              </span>
            </div>
          </div>
        </div>

        {/* Actions */}
        {showActions && (
          <div className="flex gap-2 pt-2 border-t">
            <Button
              variant="outline"
              size="sm"
              onClick={handleClick}
              disabled={isLoading}
              className="flex-1"
            >
              <Eye className="w-4 h-4 mr-1" />
              Xem chi tiết
            </Button>
            {entry.status === 'pending_approval' && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleReject}
                  disabled={isLoading}
                  className="border-red-200 text-red-700 hover:bg-red-50"
                >
                  <XCircle className="w-4 h-4 mr-1" />
                  Loại
                </Button>
                <Button
                  variant="default"
                  size="sm"
                  onClick={handleApprove}
                  disabled={isLoading}
                  className="bg-green-600 hover:bg-green-700"
                >
                  <CheckCircle className="w-4 h-4 mr-1" />
                  Duyệt
                </Button>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// Skeleton loading state
export function TimesheetEntryCardSkeleton({ variant = 'default' }: { variant?: 'default' | 'compact' | 'timeline' }) {
  if (variant === 'compact') {
    return (
      <div className="bg-background rounded-xl p-3 border animate-pulse">
        <div className="hidden md:grid grid-cols-12 gap-3 items-center">
          <div className="col-span-3 h-4 bg-muted rounded" />
          <div className="col-span-3 h-4 bg-muted rounded" />
          <div className="col-span-2 h-4 bg-muted rounded" />
          <div className="col-span-3 h-4 bg-muted rounded" />
          <div className="col-span-1 h-4 bg-muted rounded" />
        </div>
        <div className="md:hidden space-y-2">
          <div className="h-4 bg-muted rounded w-3/4" />
          <div className="flex gap-2">
            <div className="h-4 bg-muted rounded flex-1" />
            <div className="h-4 bg-muted rounded w-20" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-background rounded-xl p-4 border animate-pulse">
      <div className="space-y-4">
        <div className="flex justify-between">
          <div className="h-6 bg-muted rounded w-1/3" />
          <div className="h-6 bg-muted rounded w-20" />
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="space-y-2">
              <div className="h-3 bg-muted rounded w-16" />
              <div className="h-4 bg-muted rounded" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
