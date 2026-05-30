import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Trash2, Plus, Minus } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatCurrency } from '@/components/payrates/types';

interface MobileHourEntryProps {
  hourType: string;
  hours: number;
  rate: number;
  availableHourTypes: string[];
  onHourTypeChange: (oldHourType: string, newHourType: string) => void;
  onHoursChange: (hourType: string, hours: number) => void;
  onRemove: (hourType: string) => void;
  isRemovable?: boolean;
  className?: string;
}

export function MobileHourEntry({
  hourType,
  hours,
  rate,
  availableHourTypes,
  onHourTypeChange,
  onHoursChange,
  onRemove,
  isRemovable = true,
  className
}: MobileHourEntryProps) {
  const [localHours, setLocalHours] = useState(hours.toString());

  const handleHoursChange = (value: string) => {
    setLocalHours(value);
    const numericValue = parseFloat(value) || 0;
    if (numericValue >= 0 && numericValue <= 24) {
      onHoursChange(hourType, numericValue);
    }
  };

  const handleIncrement = () => {
    const newHours = Math.min(hours + 0.5, 24);
    setLocalHours(newHours.toString());
    onHoursChange(hourType, newHours);
  };

  const handleDecrement = () => {
    const newHours = Math.max(hours - 0.5, 0);
    setLocalHours(newHours.toString());
    onHoursChange(hourType, newHours);
  };

  const handleHourTypeChange = (newHourType: string) => {
    if (newHourType !== hourType) {
      onHourTypeChange(hourType, newHourType);
    }
  };

  const totalAmount = hours * rate;

  return (
    <div className={cn("border border-border rounded-xl p-3 bg-card", className)}>
      <div className="flex items-center gap-2 mb-3">
        <div className="flex-1 min-w-0">
          <Select value={hourType} onValueChange={handleHourTypeChange}>
            <SelectTrigger className="w-full h-10 typography-body-medium">
              <SelectValue placeholder="Chọn loại giờ" />
            </SelectTrigger>
            <SelectContent>
              {availableHourTypes.map(type => (
                <SelectItem key={type} value={type} className="typography-body-medium">
                  {type}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        
        {isRemovable && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onRemove(hourType)}
            className="h-10 w-10 text-red-500 hover:text-red-700 hover:bg-red-50 flex-shrink-0"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
      </div>

      <div className="space-y-3">
        {/* Hour Input with +/- Controls */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            onClick={handleDecrement}
            disabled={hours <= 0}
            className="h-12 w-12 rounded-full"
          >
            <Minus className="h-4 w-4" />
          </Button>
          
          <div className="flex-1">
            <Input
              type="number"
              min="0"
              max="24"
              step="0.5"
              value={localHours}
              onChange={(e) => handleHoursChange(e.target.value)}
              className="text-center typography-title-large h-12"
              placeholder="0"
            />
          </div>
          
          <Button
            variant="outline"
            size="icon"
            onClick={handleIncrement}
            disabled={hours >= 24}
            className="h-12 w-12 rounded-full"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        {/* Rate and Total Display */}
        {rate > 0 && (
          <div className="flex items-center justify-between typography-body-medium text-muted-foreground pt-2 border-t">
            <div>
              <span className="font-medium">{formatCurrency(rate)}</span>
              <span className="text-gray-400 ml-1">/giờ</span>
            </div>
            {totalAmount > 0 && (
              <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                <span className="font-semibold">{formatCurrency(totalAmount)}</span>
              </Badge>
            )}
          </div>
        )}
      </div>
    </div>
  );
}