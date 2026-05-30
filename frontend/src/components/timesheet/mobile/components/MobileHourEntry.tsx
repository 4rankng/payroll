import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Plus, Minus } from 'lucide-react';
import { formatCurrency } from '@/components/payrates/types';

interface MobileHourEntryProps {
  hourType: string;
  hours: number;
  rate: number;
  onHoursChange: (hours: number) => void;
}

export function MobileHourEntry({
  hourType,
  hours,
  rate,
  onHoursChange
}: MobileHourEntryProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="typography-body-medium text-foreground">{hourType}</span>
        {rate > 0 && (
          <span className="typography-body-small text-muted-foreground">{formatCurrency(rate)}/giờ</span>
        )}
      </div>
      
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onHoursChange(Math.max(0, hours - 0.5))}
          disabled={hours <= 0}
          className="h-10 w-10 rounded-full flex-shrink-0 touch-manipulation"
        >
          <Minus className="h-4 w-4" />
        </Button>
        
        <Input
          type="number"
          min="0"
          max="24"
          step="0.5"
          value={hours || ''}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => onHoursChange(parseFloat(e.target.value) || 0)}
          className="text-center typography-body-large h-10 flex-1 prevent-zoom"
          placeholder="0"
        />
        
        <Button
          variant="outline"
          size="sm"
          onClick={() => onHoursChange(Math.min(24, hours + 0.5))}
          disabled={hours >= 24}
          className="h-10 w-10 rounded-full flex-shrink-0 touch-manipulation"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>
      
      {hours > 0 && rate > 0 && (
        <div className="flex justify-end">
          <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200 typography-body-small">
            {formatCurrency(hours * rate)}
          </Badge>
        </div>
      )}
    </div>
  );
}