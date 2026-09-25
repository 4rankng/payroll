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
    <section className="space-y-3 border-t border-border/70 px-4 py-4 sm:border-l sm:border-t-0 sm:px-0 sm:pl-5">
      {/* Header */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <Clock className="h-3 w-3 text-muted-foreground" />
          <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Ca làm việc</span>
        </div>
        {!readOnly && !showAdd && (
          <button
            onClick={() => setShowAdd(true)}
            className="flex min-h-11 items-center gap-1 rounded-lg px-2 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
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
              className="group flex min-h-11 items-center gap-1 rounded-lg border border-border bg-background px-2 text-xs"
            >
              {isTimeRange(key) && <Clock className="h-3 w-3 text-primary shrink-0" />}
              <span className={`font-medium ${isTimeRange(key) ? 'font-mono' : ''} text-foreground`}>
                {key}
              </span>
              {!readOnly && hourTypes.length > 1 && (
                <button
                  onClick={() => handleRemove(key)}
                  className="ml-0.5 flex h-11 w-11 items-center justify-center rounded-md text-muted-foreground opacity-100 transition-all hover:bg-destructive/10 hover:text-destructive sm:opacity-0 sm:group-hover:opacity-100"
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
          <div className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-1.5 min-[380px]:grid-cols-[auto_minmax(0,1fr)_auto_minmax(0,1fr)]">
            <span className="text-xs text-muted-foreground w-6 shrink-0">Từ</span>
            <Input
              type="time"
              value={startTime}
              onChange={e => setStartTime(e.target.value)}
              className="h-11 text-xs font-mono"
              autoFocus
            />
            <span className="text-xs text-muted-foreground shrink-0">đến</span>
            <Input
              type="time"
              value={endTime}
              onChange={e => setEndTime(e.target.value)}
              className="h-11 text-xs font-mono"
            />
          </div>

          {/* Preview + actions */}
          <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-[minmax(0,1fr)_auto_auto] min-[380px]:items-center">
            <div className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
              <span className="shrink-0">Khung giờ:</span>
              <span className={`min-w-0 break-all font-mono font-semibold ${alreadyExists ? 'text-amber-700' : 'text-foreground'}`}>
                {previewKey}
              </span>
              {alreadyExists && (
                <span className="text-amber-700 text-xs shrink-0">(đã tồn tại)</span>
              )}
            </div>
            <button
              onClick={handleAdd}
              disabled={alreadyExists || !startTime || !endTime}
              className="h-11 shrink-0 rounded-lg bg-primary px-3 text-xs text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-40"
            >
              Thêm
            </button>
            <button
              onClick={() => setShowAdd(false)}
              className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg border border-border text-muted-foreground transition-colors hover:text-foreground"
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
              className="flex min-h-11 items-center gap-1 rounded-full border border-border bg-background px-3 text-xs transition-colors hover:border-primary hover:text-primary"
            >
              <span className="font-mono">{preset.value}</span>
              <span className="text-muted-foreground/60">({preset.label})</span>
            </button>
          ))}
        </div>
      )}
    </section>
  );
}
