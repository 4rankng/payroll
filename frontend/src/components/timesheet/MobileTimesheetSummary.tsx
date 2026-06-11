import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Save, X, AlertTriangle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatCurrency } from '@/utils/formatters';

interface TimesheetSummary {
  totalEmployees: number;
  totalHours: number;
  entriesCompleted: number;
  totalCost: number;
  averageHours: number;
  validationErrors: Array<{
    employeeId: number;
    totalHours: number;
    message: string;
  }>;
}

interface MobileTimesheetSummaryProps {
  summary: TimesheetSummary;
  hasChanges: boolean;
  isSaving: boolean;
  onSave: () => void;
  onCancel: () => void;
  className?: string;
}

export function MobileTimesheetSummary({
  summary,
  hasChanges,
  isSaving,
  onSave,
  onCancel,
  className
}: MobileTimesheetSummaryProps) {
  const hasValidationErrors = summary.validationErrors.length > 0;
  const maxHours = 24;
  const progressPercentage = Math.min((summary.totalHours / maxHours) * 100, 100);

  return (
    <div className={cn(
      "bg-card border border-border rounded-xl shadow-sm mx-4 mb-4",
      className
    )}>
      {/* Progress Bar */}
      <div className="p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="typography-body-medium text-foreground">
            Tổng giờ: {summary.totalHours.toFixed(1)}h
          </span>
          <span className="typography-body-medium text-muted-foreground">
            {summary.entriesCompleted}/{summary.totalEmployees} nhân viên
          </span>
        </div>

        <div className="w-full bg-gray-200 rounded-full h-2 mb-1">
          <div
            className={cn(
              "h-2 rounded-full transition-all duration-500",
              progressPercentage >= 100
                ? "bg-gradient-to-r from-yellow-400 to-yellow-500"
                : "bg-gradient-to-r from-blue-400 to-blue-500"
            )}
            style={{ width: `${Math.min(progressPercentage, 100)}%` }}
          />
        </div>

        <div className="flex items-center justify-between typography-body-small text-muted-foreground">
          <span>0h</span>
          <span>{maxHours} giờ</span>
        </div>
      </div>

      {/* Summary Stats */}
      {summary.totalCost > 0 && (
        <div className="p-4 bg-muted/50 border-t border-border">          <div className="flex items-center justify-between">
            <span className="typography-body-medium text-foreground">Tổng chi phí:</span>
            <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 font-semibold">
              {formatCurrency(summary.totalCost)}
            </Badge>
          </div>
        </div>
      )}

      {/* Error Alert */}
      {hasValidationErrors && (
        <div className="p-4 bg-red-50 border-t border-red-100">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-red-500 flex-shrink-0" />
            <div className="typography-body-medium text-red-700">
              <span className="font-medium">
                {summary.validationErrors.length} nhân viên vượt quá 24h/ngày
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex gap-3 p-4">
        <Button
          variant="outline"
          onClick={onCancel}
          disabled={!hasChanges || isSaving}
          className="flex-1 h-12 typography-body-large border-2 border-border bg-card hover:bg-muted/50 text-foreground rounded-xl flex items-center justify-center gap-2"
        >
          <X className="w-5 h-5" />
          Đóng
        </Button>

        <Button
          onClick={onSave}
          disabled={!hasChanges || isSaving || hasValidationErrors}
          className={cn(
            "flex-1 h-12 typography-body-large rounded-xl flex items-center justify-center gap-2 transition-all duration-200",
            hasValidationErrors
              ? "bg-gray-400 hover:bg-gray-400 text-white cursor-not-allowed"
              : "bg-yellow-400 hover:bg-yellow-500 text-black font-semibold"
          )}
        >
          <Save className="w-5 h-5" />
          {isSaving
            ? 'Đang lưu...'
            : hasValidationErrors
              ? 'Có lỗi cần sửa'
              : 'Lưu Bảng Công'
          }
        </Button>
      </div>
    </div>
  );
}
