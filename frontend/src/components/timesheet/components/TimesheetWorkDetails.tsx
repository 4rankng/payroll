import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Timesheet } from '@/types/api/timesheet.types';
import { getPaytypeText, getStatusBadge, formatDate } from '../utils/timesheetHelpers.tsx';

interface TimesheetWorkDetailsProps {
  timesheet: Timesheet;
  isEditMode: boolean;
  editedData: Partial<Timesheet>;
  onFieldChange: (field: string, value: string | number) => void;
}

export function TimesheetWorkDetails({
  timesheet,
  isEditMode,
  editedData,
  onFieldChange
}: TimesheetWorkDetailsProps) {
  return (
    <div className="bg-blue-50 rounded-xl p-4">
      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <Label className="text-blue-700">Ngày làm việc</Label>
          {isEditMode ? (
            <Input
              type="date"
              value={editedData.date || timesheet.date}
              onChange={(e) => onFieldChange('date', e.target.value)}
              className="mt-1"
            />
          ) : (
            <p className="typography-title-large mt-1">
              {formatDate(timesheet.date)}
            </p>
          )}
        </div>

        <div>
          <Label className="text-blue-700">Số giờ</Label>
          {isEditMode ? (
            <Input
              type="number"
              step="0.5"
              min="0"
              max="24"
              value={editedData.hours_worked || timesheet.hours_worked}
              onChange={(e) => onFieldChange('hours_worked', parseFloat(e.target.value))}
              className="mt-1"
            />
          ) : (
            <p className="typography-title-large text-green-600 mt-1">{timesheet.hours_worked} giờ</p>
          )}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <span className="typography-body-small text-blue-600">Loại công</span>
          <div className="mt-1">
            <Badge variant="outline">{getPaytypeText(timesheet.paytype)}</Badge>
          </div>
        </div>

        <div>
          <span className="typography-body-small text-blue-600">Trạng thái</span>
          <div className="mt-1">{getStatusBadge(timesheet.status)}</div>
        </div>
      </div>

      {isEditMode && (
        <div className="mt-4">
          <Label className="text-blue-700">Ghi chú</Label>
          <Textarea
            placeholder="Nhập ghi chú..."
            value={editedData.notes || ''}
            onChange={(e) => onFieldChange('notes', e.target.value)}
            rows={2}
            className="mt-1"
          />
        </div>
      )}
    </div>
  );
}
