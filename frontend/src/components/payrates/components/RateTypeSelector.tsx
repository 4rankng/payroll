import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Card, CardContent } from '@/components/ui/card';
import { Clock, Tag } from 'lucide-react';
import { PayrateStructureType, POSITION_RATE_TEMPLATES } from '../types';
import { COMMON_HOUR_TYPES } from '@/types/api/payrate.types';

interface RateTypeSelectorProps {
  value: PayrateStructureType;
  onChange: (value: PayrateStructureType) => void;
  disabled?: boolean;
}

export function RateTypeSelector({ value, onChange, disabled }: RateTypeSelectorProps) {
  return (
    <div className="space-y-4">
      <div>
        <Label className="typography-body-large">Loại cấu hình lương</Label>
        <p className="typography-body-medium text-muted-foreground mt-1">
          Chọn cách tính lương cho dự án
        </p>
      </div>

      <RadioGroup
        value={value}
        onValueChange={(value) => onChange(value as PayrateStructureType)}
        disabled={disabled}
        className="grid gap-4"
      >
        {POSITION_RATE_TEMPLATES.map((template) => (
          <div key={template.type}>
            <RadioGroupItem
              value={template.type}
              id={template.type}
              className="peer sr-only"
            />
            <Label
              htmlFor={template.type}
              className="cursor-pointer"
            >
              <Card className="peer-checked:ring-2 peer-checked:ring-primary peer-checked:border-primary hover:bg-muted/50 transition-colors">
                <CardContent className="flex items-start gap-4 p-4">
                  <div className="mt-0.5">
                    {template.type === 'hourly' ? (
                      <Clock className="w-5 h-5 text-primary" />
                    ) : (
                      <Tag className="w-5 h-5 text-primary" />
                    )}
                  </div>
                  <div className="flex-1">
                    <div className="font-medium">{template.name}</div>
                    <div className="typography-body-medium text-muted-foreground mt-1">
                      {template.description}
                    </div>
                    <div className="typography-body-small text-muted-foreground mt-2">
                      {template.type === 'hourly'
                        ? `Ví dụ: ${COMMON_HOUR_TYPES.TIME_RANGES.slice(0, 3).join(', ')}`
                        : `Ví dụ: ${COMMON_HOUR_TYPES.CATEGORIES.join(', ')}`
                      }
                    </div>
                  </div>
                </CardContent>
              </Card>
            </Label>
          </div>
        ))}
      </RadioGroup>
    </div>
  );
}