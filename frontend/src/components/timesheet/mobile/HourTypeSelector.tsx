import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Clock, Moon, Sun, Zap } from 'lucide-react';
import { PayrateStructure, DayType } from '@/types/api/payrate.types';

interface HourTypeSelectorProps {
  hourType: string;
  onHourTypeChange: (hourType: string) => void;
  payrateConfig?: PayrateStructure;
  position: string;
  dayType: DayType;
}

export function HourTypeSelector({
  hourType,
  onHourTypeChange,
  payrateConfig,
  position,
  dayType
}: HourTypeSelectorProps) {
  // Get available hour types from payrate config
  const getAvailableHourTypes = () => {
    const defaultHourTypes = [
      { key: 'ca ngày', label: 'Ca ngày', description: 'Ca làm việc ban ngày', icon: Sun },
      { key: 'ca đêm', label: 'Ca đêm', description: 'Ca làm việc ban đêm', icon: Moon },
      { key: 'tăng ca', label: 'Làm thêm', description: 'Giờ tăng ca ngoài ca', icon: Zap }
    ];

    if (payrateConfig && payrateConfig[position] && payrateConfig[position][dayType]) {
      const dayTypeRates = payrateConfig[position][dayType];
      const configHourTypes = Object.keys(dayTypeRates).map(key => {
        const defaultType = defaultHourTypes.find(t => t.key === key);

        return {
          key,
          label: defaultType?.label || key,
          description: defaultType?.description || 'Loại giờ từ cấu hình',
          icon: defaultType?.icon || Clock,
          rate: dayTypeRates[key]
        };
      });
      return configHourTypes;
    }

    return defaultHourTypes.map(type => ({ ...type, rate: 0 }));
  };

  const availableHourTypes = getAvailableHourTypes();

  const getHourTypeIcon = (IconComponent: unknown) => {
    return <IconComponent className="h-4 w-4" />;
  };

  const formatRate = (rate: number) => {
    return new Intl.NumberFormat('vi-VN', {
      style: 'currency',
      currency: 'VND',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0
    }).format(rate);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Clock className="h-4 w-4 text-muted-foreground" />
        <Label className="typography-body-medium">Khung giờ</Label>
      </div>

      <Select value={hourType} onValueChange={onHourTypeChange}>
        <SelectTrigger className="h-11 border-2 border-border/50 hover:border-border transition-colors">
          <SelectValue placeholder="Chọn khung giờ làm việc">
            {hourType && (
              <div className="flex items-center gap-2">
                {getHourTypeIcon(availableHourTypes.find(t => t.key === hourType)?.icon || Clock)}
                <span>{availableHourTypes.find(t => t.key === hourType)?.label}</span>
              </div>
            )}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          {availableHourTypes.map((type) => (
            <SelectItem key={type.key} value={type.key}>
              <div className="flex items-center justify-between w-full py-1">
                <div className="flex items-center gap-3">
                  {getHourTypeIcon(type.icon)}
                  <div className="flex flex-col items-start">
                    <span className="font-medium">{type.label}</span>
                    <span className="typography-body-small text-muted-foreground">
                      {type.description}
                    </span>
                  </div>
                </div>
                {type.rate > 0 && (
                  <span className="ml-2 typography-body-small text-blue-600 font-medium">
                    {formatRate(type.rate)}/h
                  </span>
                )}
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {availableHourTypes.length === 0 && (
        <div className="bg-muted/50 rounded-xl p-3">
          <p className="typography-body-small text-muted-foreground">
            Không có khung giờ nào được cấu hình cho vị trí <strong>{position}</strong> vào <strong>{dayType}</strong>
          </p>
        </div>
      )}

      {payrateConfig && availableHourTypes.length > 0 && (
        <div className="bg-blue-50 border border-blue-200 rounded-xl p-3">
          <p className="typography-body-small text-blue-700">
            <strong>Mức lương:</strong> Được tính theo cấu hình cho {position} - {dayType}
          </p>
        </div>
      )}
    </div>
  );
}
