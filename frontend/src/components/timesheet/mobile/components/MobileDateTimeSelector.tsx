import { Card, CardContent } from '@/components/ui/card';
import { Calendar } from 'lucide-react';
import { DayTypeIndicator } from '../DayTypeIndicator';
import { PositionSelector } from '../PositionSelector';
import { HourTypeSelector } from '../HourTypeSelector';
import { HoursWorkedInput } from '../HoursWorkedInput';
import { DayType } from '@/types/api/payrate.types';
import { PayrateStructure } from '@/types/api/payrate.types';

interface MobileDateTimeSelectorProps {
  selectedDate: Date;
  onDateChange: (date: Date) => void;
  dayType: DayType | null;
  onDayTypeChange: (dayType: DayType | null) => void;
  position: string;
  onPositionChange: (position: string) => void;
  hourType: string;
  onHourTypeChange: (hourType: string) => void;
  hoursWorked: number;
  onHoursChange: (hours: number) => void;
  payrateConfig?: PayrateStructure;
}

export function MobileDateTimeSelector({
  selectedDate,
  onDateChange,
  dayType,
  onDayTypeChange,
  position,
  onPositionChange,
  hourType,
  onHourTypeChange,
  hoursWorked,
  onHoursChange,
  payrateConfig
}: MobileDateTimeSelectorProps) {
  return (
    <>
      {/* Date and Day Type */}
      <Card className="border-none shadow-sm">
        <CardContent className="pt-4 sm:pt-6 pb-4 sm:pb-6">
          <div className="space-y-4">
            <div className="flex items-center gap-2 mb-2 sm:mb-3">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <label className="typography-body-medium font-medium">Ngày làm việc</label>
            </div>
            
            
            <DayTypeIndicator
              date={selectedDate}
              dayType={dayType}
              onDayTypeChange={onDayTypeChange}
            />
          </div>
        </CardContent>
      </Card>

      {/* Position */}
      <Card className="border-none shadow-sm">
        <CardContent className="pt-4 sm:pt-6 pb-4 sm:pb-6">
          <PositionSelector
            position={position}
            onPositionChange={onPositionChange}
            payrateConfig={payrateConfig}
          />
        </CardContent>
      </Card>

      {/* Hour Type and Hours */}
      {dayType && (
        <Card className="border-none shadow-sm">
          <CardContent className="pt-4 sm:pt-6 pb-4 sm:pb-6 space-y-4">
            <HourTypeSelector
              hourType={hourType}
              onHourTypeChange={onHourTypeChange}
              payrateConfig={payrateConfig}
              position={position}
              dayType={dayType}
            />
            
            {hourType && (
              <HoursWorkedInput
                hours={hoursWorked}
                onHoursChange={onHoursChange}
              />
            )}
          </CardContent>
        </Card>
      )}
    </>
  );
}