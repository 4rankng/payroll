import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';
import { AlertTriangle, Calendar, Sun } from 'lucide-react';
import { DayType } from '@/types/api/payrate.types';
import { getDayTypeOrNull, isSaturdayRequiringDayType, isVietnamesePublicHoliday } from '@/utils/dateHelpers';

interface DayTypeIndicatorProps {
  date: Date;
  dayType: DayType | null;
  onDayTypeChange: (dayType: DayType) => void;
}

export function DayTypeIndicator({
  date,
  dayType,
  onDayTypeChange
}: DayTypeIndicatorProps) {
  const dateString = date.toISOString().split('T')[0];
  const isSaturday = isSaturdayRequiringDayType(dateString);
  const isHoliday = isVietnamesePublicHoliday(dateString);
  const dayOfWeek = date.getDay();
  const isSunday = dayOfWeek === 0;

  // Auto-determined day type (if not Saturday)
  const autoDayType = !isSaturday ? getDayTypeOrNull(dateString) : null;

  const getDayTypeIcon = (type: DayType) => {
    switch (type) {
      case 'ngày lễ':
        return <Calendar className="h-4 w-4" />;
      case 'ngày nghỉ':
        return <Sun className="h-4 w-4" />;
      default:
        return <Calendar className="h-4 w-4" />;
    }
  };

  const getDayTypeDescription = (type: DayType) => {
    switch (type) {
      case 'ngày thường':
        return 'Ngày làm việc bình thường (Thứ 2-6)';
      case 'ngày nghỉ':
        return 'Ngày nghỉ trong tuần (Chủ nhật, Thứ 7 nghỉ)';
      case 'ngày lễ':
        return 'Ngày lễ quốc gia';
      default:
        return '';
    }
  };

  // If not Saturday, show auto-determined day type
  if (!isSaturday && autoDayType) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Label className="typography-body-medium">Loại ngày</Label>
          <div className={`flex items-center gap-1 typography-body-medium font-medium ${
            autoDayType === 'ngày lễ' ? 'text-red-600' :
            autoDayType === 'ngày nghỉ' ? 'text-muted-foreground' : 'text-blue-600'
          }`}>
            {getDayTypeIcon(autoDayType)}
            {autoDayType === 'ngày thường' ? 'ngày thường' :
             autoDayType === 'ngày nghỉ' ? 'ngày nghỉ' : 'ngày lễ'}
          </div>
        </div>

        <div className="bg-muted/50 rounded-xl p-3">
          <p className="typography-body-small text-muted-foreground">
            {getDayTypeDescription(autoDayType)}
          </p>
        </div>
      </div>
    );
  }

  // Saturday - requires user selection
  if (isSaturday) {
    return (
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Label className="typography-body-medium">Loại ngày</Label>
          <div className="flex items-center gap-1 typography-body-medium font-medium text-amber-700">
            <AlertTriangle className="h-3 w-3" />
            cần xác định
          </div>
        </div>

        <div className="bg-warning/10 border border-warning/20 rounded-xl p-3">
          <p className="typography-body-small text-warning-foreground mb-3">
            <strong>Thứ 7:</strong> Vui lòng chọn loại ngày
          </p>

          <RadioGroup
            value={dayType || ''}
            onValueChange={(value) => onDayTypeChange(value as DayType)}
            className="space-y-2"
          >
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="ngày thường" id="weekday" />
              <Label htmlFor="weekday" className="typography-body-medium">
                Ngày thường (làm việc bình thường)
              </Label>
            </div>
            <div className="flex items-center space-x-2">
              <RadioGroupItem value="ngày nghỉ" id="weekend" />
              <Label htmlFor="weekend" className="typography-body-medium">
                Ngày nghỉ (nghỉ cuối tuần)
              </Label>
            </div>
          </RadioGroup>
        </div>

        {dayType && (
          <div className="bg-muted/50 rounded-xl p-3">
            <p className="typography-body-small text-muted-foreground">
              {getDayTypeDescription(dayType)}
            </p>
          </div>
        )}
      </div>
    );
  }

  // Fallback for edge cases
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Label className="typography-body-medium">Loại ngày</Label>
        <span className="typography-body-medium font-medium text-muted-foreground">
          không xác định
        </span>
      </div>
    </div>
  );
}