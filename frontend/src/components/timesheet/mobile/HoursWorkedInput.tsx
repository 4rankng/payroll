import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Timer, Plus, Minus, AlertTriangle } from 'lucide-react';

interface HoursWorkedInputProps {
  hours: number;
  onHoursChange: (hours: number) => void;
}

export function HoursWorkedInput({
  hours,
  onHoursChange
}: HoursWorkedInputProps) {
  const [inputValue, setInputValue] = useState(hours.toString());

  const handleInputChange = (value: string) => {
    // Allow empty string or valid number format
    if (value === '' || /^\d*\.?\d*$/.test(value)) {
      setInputValue(value);

      // Parse and validate the input
      const numValue = value === '' ? 0 : parseFloat(value) || 0;
      if (numValue >= 0 && numValue <= 24) {
        onHoursChange(numValue);
      }
    }
  };

  const handleQuickSelect = (quickHours: number) => {
    setInputValue(quickHours.toString());
    onHoursChange(quickHours);
  };

  const handleIncrement = () => {
    const newValue = Math.min(hours + 0.5, 24);
    setInputValue(newValue.toString());
    onHoursChange(newValue);
  };

  const handleDecrement = () => {
    const newValue = Math.max(hours - 0.5, 0);
    setInputValue(newValue.toString());
    onHoursChange(newValue);
  };

  const isValid = hours >= 0 && hours <= 24;
  const isOvertime = hours > 8;

  const quickHours = [4, 6, 8, 10, 12];

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Timer className="h-4 w-4 text-muted-foreground" />
        <Label className="typography-body-medium">Số giờ làm việc</Label>
        {isOvertime && (
          <Badge variant="warning" className="typography-body-small">
            Làm thêm
          </Badge>
        )}
      </div>

      {/* Input with increment/decrement */}
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={handleDecrement}
          disabled={hours <= 0}
          className="h-10 w-10 p-0"
        >
          <Minus className="h-4 w-4" />
        </Button>

        <div className="flex-1">
          <Input
            type="text"
            min="0"
            max="24"
            step="0.5"
            value={inputValue}
            onChange={(e) => handleInputChange(e.target.value)}
            placeholder="0"
            className="h-10 text-center typography-title-large border-2 border-border/50 hover:border-border focus:border-primary"
          />
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={handleIncrement}
          disabled={hours >= 24}
          className="h-10 w-10 p-0"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {/* Quick select buttons */}
      <div className="space-y-2">
        <Label className="typography-body-small text-muted-foreground">Chọn nhanh:</Label>
        <div className="flex flex-wrap gap-2">
          {quickHours.map((quickHour) => (
            <Button
              key={quickHour}
              variant={hours === quickHour ? "default" : "outline"}
              size="sm"
              onClick={() => handleQuickSelect(quickHour)}
              className="h-8 typography-body-small"
            >
              {quickHour}h
            </Button>
          ))}
        </div>
      </div>

      {/* Validation and hints */}
      {!isValid && (
        <div className="bg-destructive/10 border border-destructive/20 rounded-xl p-3 flex items-start gap-2">
          <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 flex-shrink-0" />
          <div className="typography-body-small text-destructive">
            <strong>Lỗi:</strong> Số giờ phải từ 0 đến 24
          </div>
        </div>
      )}

      {isValid && hours > 0 && (
        <div className="bg-muted/50 rounded-xl p-3">
          <div className="flex items-center justify-between typography-body-small">
            <span className="text-muted-foreground">
              {hours <= 8 ? 'Giờ hành chính' : `Giờ hành chính: 8h, Làm thêm: ${hours - 8}h`}
            </span>
            <Badge variant={isOvertime ? "warning" : "secondary"} className="typography-body-small">
              {hours}h
            </Badge>
          </div>
        </div>
      )}

      {hours === 0 && (
        <div className="text-center py-4 text-muted-foreground">
          <Timer className="h-8 w-8 mx-auto mb-2 opacity-50" />
          <p className="typography-body-small">Nhập số giờ làm việc</p>
        </div>
      )}
    </div>
  );
}