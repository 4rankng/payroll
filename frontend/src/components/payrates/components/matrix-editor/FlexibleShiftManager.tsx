import { useState } from 'react';
import { Plus, X, Clock } from 'lucide-react';
import { Input } from '@/components/ui/input';
import type { PayrateStructure } from '../../types';
import { getPositionsFromRates, isTimeRange, FLEXIBLE_DAY_TYPES } from '../../types';

// Flexible projects only use 'ngày thường' — no weekend/holiday distinction (MVP spec)
const DAY_TYPES = FLEXIBLE_DAY_TYPES;

const PRESET_SHIFTS: { label: string; value: string }[] = [
  { label: 'Hành chính', value: '08:00-17:00' },
  { label: 'Ca sáng', value: '06:00-14:00' },
  { label: 'Ca chiều', value: '14:00-22:00' },
  { label: 'Ca đêm', value: '22:00-06:00' },
  { label: 'Ca 12h ngày', value: '08:00-20:00' },
  { label: 'Ca 12h đêm', value: '20:00-08:00' },
];

interface FlexibleShiftManagerProps {
  rates: PayrateStructure;
  hourTypes: string[];
  onChange: (rates: PayrateStructure) => void;
  readOnly?: boolean;
}

export function FlexibleShiftManager({
  rates,
  hourTypes,
  onChange,
  readOnly = false,
}: FlexibleShiftManagerProps) {
  const [showAdd, setShowAdd] = useState(false);
  const [startTime, setStartTime] = useState('08:00');
  const [endTime, setEndTime] = useState('17:00');

  const positions = getPositionsFromRates(rates);
  const previewKey = `${startTime}-${endTime}`;
  const alreadyExists = hourTypes.includes(previewKey);

  const addShiftKey = (key: string) => {
    const newRates = { ...rates };
    positions.forEach(position => {
      DAY_TYPES.forEach(dayType => {
        if (!newRates[position]) newRates[position] = {} as PayrateStructure[string];
        if (!newRates[position][dayType]) newRates[position][dayType] = {};
        (newRates[position][dayType] as Record<string, number>)[key] = 0;
      });
    });
    onChange(newRates);
  };

  const handleAdd = () => {
    if (!startTime || !endTime || alreadyExists) return;
    addShiftKey(previewKey);
    setShowAdd(false);
  };

  const handleRemove = (key: string) => {
    if (hourTypes.length <= 1) return;
    const newRates = { ...rates };
    positions.forEach(position => {
      DAY_TYPES.forEach(dayType => {
        const hourConfig = newRates[position]?.[dayType] as Record<string, number> | undefined;
        if (hourConfig) delete hourConfig[key];
      });
    });
    onChange(newRates);
  };

  const handlePreset = (value: string) => {
    if (hourTypes.includes(value)) return;
    addShiftKey(value);
  };

  const availablePresets = PRESET_SHIFTS.filter(p => !hourTypes.includes(p.value));

  return (
    <div className="rounded-xl border border-border/60 bg-muted/20 p-3 space-y-2.5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Clock className="h-3 w-3 text-muted-foreground" />
          <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Ca làm việc</span>
        </div>
        {!readOnly && !showAdd && (
          <button
            onClick={() => setShowAdd(true)}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            <Plus className="h-3 w-3" />
            Thêm ca
          </button>
        )}
      </div>

      {/* Current shifts */}
      {hourTypes.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {hourTypes.map(key => (
            <div
              key={key}
              className="group flex items-center gap-1 text-xs bg-background border border-border rounded-lg px-2 py-1"
            >
              {isTimeRange(key) && <Clock className="h-3 w-3 text-primary shrink-0" />}
              <span className={`font-medium ${isTimeRange(key) ? 'font-mono' : ''} text-foreground`}>
                {key}
              </span>
              {!readOnly && hourTypes.length > 1 && (
                <button
                  onClick={() => handleRemove(key)}
                  className="ml-0.5 opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive transition-all"
                  title={`Xóa ca ${key}`}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Add shift form */}
      {showAdd && !readOnly && (
        <div className="space-y-2 pt-2 border-t border-border/40">
          {/* Time pickers */}
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-muted-foreground w-6 shrink-0">Từ</span>
            <Input
              type="time"
              value={startTime}
              onChange={e => setStartTime(e.target.value)}
              className="h-7 text-xs font-mono flex-1"
              autoFocus
            />
            <span className="text-[10px] text-muted-foreground shrink-0">đến</span>
            <Input
              type="time"
              value={endTime}
              onChange={e => setEndTime(e.target.value)}
              className="h-7 text-xs font-mono flex-1"
            />
          </div>

          {/* Preview + actions */}
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground flex-1 min-w-0">
              <span className="shrink-0">Khung giờ:</span>
              <span className={`font-mono font-semibold truncate ${alreadyExists ? 'text-amber-600' : 'text-foreground'}`}>
                {previewKey}
              </span>
              {alreadyExists && (
                <span className="text-amber-600 text-[10px] shrink-0">(đã tồn tại)</span>
              )}
            </div>
            <button
              onClick={handleAdd}
              disabled={alreadyExists || !startTime || !endTime}
              className="h-7 px-2.5 text-xs rounded-lg bg-primary text-primary-foreground disabled:opacity-40 hover:bg-primary/90 transition-colors shrink-0"
            >
              Thêm
            </button>
            <button
              onClick={() => setShowAdd(false)}
              className="h-7 w-7 flex items-center justify-center rounded-lg border border-border text-muted-foreground hover:text-foreground transition-colors shrink-0"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        </div>
      )}

      {/* Common presets */}
      {!readOnly && availablePresets.length > 0 && !showAdd && (
        <div className="flex flex-wrap gap-1.5">
          {availablePresets.map(preset => (
            <button
              key={preset.value}
              onClick={() => handlePreset(preset.value)}
              className="flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-full border border-border bg-background hover:border-primary hover:text-primary transition-colors"
            >
              <span className="font-mono">{preset.value}</span>
              <span className="text-muted-foreground/60">({preset.label})</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
