import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { 
  Calendar,
  Clock,
  Copy,
  DollarSign,
  ChevronDown,
  ChevronUp
} from 'lucide-react';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  RateCategory,
  DayConfiguration,
  PayrateStructureType,
  parseCurrency
} from '../types';
import { formatCurrency } from '@/utils/formatters';
import { CurrencyInput } from '../CurrencyInput';

interface PositionRateFormProps {
  position: string;
  rates: DayConfiguration;
  rateType: PayrateStructureType;
  onChange: (position: string, rates: DayConfiguration) => void;
  onCopyFrom?: (fromPosition: string, toPosition: string) => void;
  availablePositions?: string[];
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
}


const DAY_TYPES = [
  { key: 'ngày thường', label: 'Ngày thường', icon: Calendar },
  { key: 'cuối tuần', label: 'Cuối tuần', icon: Calendar },
  { key: 'ngày lễ', label: 'Ngày lễ', icon: Calendar }
];

export function PositionRateForm({
  position,
  rates,
  rateType,
  onChange,
  onCopyFrom,
  availablePositions = [],
  isCollapsed = false,
  onToggleCollapse
}: PositionRateFormProps) {
  const [copyFromPosition, setCopyFromPosition] = useState<string>('');

  const handleRateChange = (dayType: string, timeSlot: string, value: number) => {
    const updatedRates = {
      ...rates,
      [dayType]: {
        ...rates[dayType as keyof typeof rates],
        [timeSlot]: value
      }
    };
    onChange(position, updatedRates);
  };

  const handleCopyRates = () => {
    if (copyFromPosition && onCopyFrom) {
      onCopyFrom(copyFromPosition, position);
      setCopyFromPosition('');
    }
  };

  const renderTimeSlotInput = (dayType: string, timeSlot: string, rate: number) => {
    return (
      <div key={timeSlot} className="grid grid-cols-2 gap-4 items-center p-3 bg-muted/30 rounded-xl">
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-muted-foreground" />
          <Label className="typography-body-medium">{timeSlot}</Label>
        </div>
        <CurrencyInput
          value={rate}
          onChange={(value) => handleRateChange(dayType, timeSlot, value)}
          placeholder="0"
        />
      </div>
    );
  };

  const renderDayTypeContent = (dayType: string) => {
    const dayRates = rates[dayType as keyof typeof rates];
    if (!dayRates || typeof dayRates !== 'object') return null;

    return (
      <div className="space-y-3">
        {Object.entries(dayRates).map(([timeSlot, rate]) => 
          renderTimeSlotInput(dayType, timeSlot, rate as number)
        )}
      </div>
    );
  };

  const getTotalRate = () => {
    let total = 0;
    Object.values(rates).forEach(dayType => {
      if (typeof dayType === 'object' && dayType !== null) {
        Object.values(dayType).forEach(rate => {
          if (typeof rate === 'number') {
            total += rate;
          }
        });
      }
    });
    return total;
  };

  const copyFromOptions = availablePositions.filter(pos => pos !== position);

  return (
    <Card>
      <Collapsible open={!isCollapsed} onOpenChange={() => onToggleCollapse?.()}>
        <CollapsibleTrigger asChild>
          <CardHeader className="cursor-pointer hover:bg-muted/50 transition-colors">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-3">
                <Badge variant="secondary" className="typography-body-medium">
                  {position}
                </Badge>
                <div className="typography-body-medium text-muted-foreground">
                  Tổng: {formatCurrency(getTotalRate())}
                </div>
              </CardTitle>
              <div className="flex items-center gap-2">
                {copyFromOptions.length > 0 && (
                  <div className="flex items-center gap-2 mr-4">
                    <select
                      value={copyFromPosition}
                      onChange={(e) => setCopyFromPosition(e.target.value)}
                      className="typography-body-small bg-background border rounded px-2 py-1"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <option value="">Sao chép từ...</option>
                      {copyFromOptions.map(pos => (
                        <option key={pos} value={pos}>{pos}</option>
                      ))}
                    </select>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCopyRates();
                      }}
                      disabled={!copyFromPosition}
                    >
                      <Copy className="w-3 h-3" />
                    </Button>
                  </div>
                )}
                {isCollapsed ? (
                  <ChevronDown className="w-4 h-4" />
                ) : (
                  <ChevronUp className="w-4 h-4" />
                )}
              </div>
            </div>
          </CardHeader>
        </CollapsibleTrigger>
        
        <CollapsibleContent>
          <CardContent className="pt-0">
            <Tabs defaultValue={DAY_TYPES[0].key} className="w-full">
              <TabsList className="grid w-full grid-cols-3">
                {DAY_TYPES.map(({ key, label, icon: Icon }) => (
                  <TabsTrigger key={key} value={key} className="flex items-center gap-2">
                    <Icon className="w-4 h-4" />
                    {label}
                  </TabsTrigger>
                ))}
              </TabsList>
              
              {DAY_TYPES.map(({ key }) => (
                <TabsContent key={key} value={key} className="mt-4">
                  {renderDayTypeContent(key)}
                </TabsContent>
              ))}
            </Tabs>

            <Separator className="my-4" />

            {/* Summary */}
            <div className="flex items-center justify-between typography-body-medium">
              <span className="text-muted-foreground">
                Loại: {rateType === 'hourly' ? 'Theo khung giờ' : 'Theo danh mục'}
              </span>
              <div className="flex items-center gap-2">
                <DollarSign className="w-4 h-4 text-muted-foreground" />
                <span className="font-medium">{formatCurrency(getTotalRate())}</span>
              </div>
            </div>
          </CardContent>
        </CollapsibleContent>
      </Collapsible>
    </Card>
  );
}