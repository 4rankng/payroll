import { Input } from '@/components/ui/input';
import { X } from 'lucide-react';
import type { DayType, PayrateStructure, ValidationResult } from '../../types';
import { ALL_DAY_TYPES } from '../../types';
import { PositionCell } from './PositionCell';
import { RateCell } from './RateCell';

const DAY_LABELS: Record<DayType, string> = {
  'ngày thường': 'Ngày thường',
  'ngày nghỉ': 'Ngày nghỉ',
  'ngày lễ': 'Ngày lễ',
  weekday: 'Ngày thường',
  weekend: 'Ngày nghỉ',
  holiday: 'Ngày lễ',
};

interface PayratePositionListProps {
  rates: PayrateStructure;
  originalRates?: PayrateStructure;
  positions: string[];
  hourTypes: string[];
  readOnly?: boolean;
  validation?: ValidationResult;
  editingHourType: string | null;
  editingHourTypeValue: string;
  editingHourTypeError?: string | null;
  onEditHourType: (hourType: string) => void;
  onSaveHourType: () => void;
  onRemoveHourType: (hourType: string) => void;
  onCancelEditHourType: () => void;
  setEditingHourTypeValue: (value: string) => void;
  editingPosition: string | null;
  editingPositionValue: string;
  onEditPosition: (position: string) => void;
  onSavePosition: () => void;
  onRemovePosition: (position: string) => void;
  onCopyRates: (rates: PayrateStructure) => void;
  setEditingPositionValue: (value: string) => void;
  setEditingPosition: (position: string | null) => void;
  onRateChange: (position: string, dayType: DayType, hourType: string, value: string) => void;
}

export function PayratePositionList({
  rates,
  originalRates,
  positions,
  hourTypes,
  readOnly = false,
  validation,
  editingHourType,
  editingHourTypeValue,
  editingHourTypeError,
  onEditHourType,
  onSaveHourType,
  onRemoveHourType,
  onCancelEditHourType,
  setEditingHourTypeValue,
  editingPosition,
  editingPositionValue,
  onEditPosition,
  onSavePosition,
  onRemovePosition,
  onCopyRates,
  setEditingPositionValue,
  setEditingPosition,
  onRateChange,
}: PayratePositionListProps) {
  return (
    <div data-slot="payrate-position-list">
      <section className="border-b border-border/70 bg-muted/20 px-4 py-3" aria-labelledby="hour-types-title">
        <div className="mb-2 flex items-center justify-between gap-3">
          <h3 id="hour-types-title" className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Khung giờ
          </h3>
          <span className="text-xs text-muted-foreground">Đơn vị: ₫/giờ</span>
        </div>
        <div className="flex flex-wrap gap-2">
          {hourTypes.map(hourType => (
            <div key={hourType} className="flex min-h-11 min-w-0 items-center rounded-lg border border-border bg-card pl-3">
              {editingHourType === hourType ? (
                <Input
                  type="text"
                  value={editingHourTypeValue}
                  aria-label={`Tên khung giờ ${hourType}`}
                  aria-invalid={!!editingHourTypeError}
                  onChange={event => setEditingHourTypeValue(event.target.value)}
                  onKeyDown={event => {
                    if (event.key === 'Enter') {
                      event.stopPropagation();
                      onSaveHourType();
                    }
                    if (event.key === 'Escape') {
                      event.stopPropagation();
                      onCancelEditHourType();
                    }
                  }}
                  onBlur={onSaveHourType}
                  className="h-9 min-w-20 border-0 bg-transparent px-0 text-sm font-semibold shadow-none focus-visible:ring-0"
                  autoFocus
                />
              ) : (
                <button
                  type="button"
                  onClick={() => !readOnly && onEditHourType(hourType)}
                  className="min-h-11 min-w-16 break-words text-left text-sm font-semibold text-foreground disabled:cursor-default"
                  disabled={readOnly}
                >
                  {hourType}
                </button>
              )}
              {!readOnly && hourTypes.length > 1 && (
                <button
                  type="button"
                  onClick={() => onRemoveHourType(hourType)}
                  className="flex h-11 w-11 shrink-0 items-center justify-center rounded-r-lg text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                  aria-label={`Xóa khung giờ ${hourType}`}
                >
                  <X className="h-4 w-4" aria-hidden="true" />
                </button>
              )}
            </div>
          ))}
        </div>
      </section>

      <div className="grid grid-cols-1 divide-y divide-border/70 md:grid-cols-2 md:divide-x md:divide-y-0">
        {positions.map((position, positionIndex) => (
          <section
            key={position}
            className="min-w-0 px-4 py-4 md:border-b md:border-border/70"
            aria-label={`Mức lương cho ${position}`}
          >
            <div className="mb-4 border-b border-border/60 pb-3">
              <PositionCell
                position={position}
                positionIndex={positionIndex}
                firstPosition={positions[0]}
                positions={positions}
                readOnly={readOnly}
                rates={rates}
                editingPosition={editingPosition}
                editingPositionValue={editingPositionValue}
                onEditPosition={onEditPosition}
                onSavePosition={onSavePosition}
                onRemovePosition={onRemovePosition}
                onCopyRates={onCopyRates}
                setEditingPositionValue={setEditingPositionValue}
                setEditingPosition={setEditingPosition}
              />
            </div>

            <div className="space-y-4">
              {ALL_DAY_TYPES.map(dayType => (
                <fieldset key={dayType} className="min-w-0">
                  <legend className="mb-2 text-xs font-medium text-muted-foreground">
                    {DAY_LABELS[dayType]}
                  </legend>
                  <div className="grid min-w-0 grid-cols-2 gap-2 max-[359px]:grid-cols-1">
                    {hourTypes.map(hourType => {
                      const rate = (rates[position]?.[dayType]?.[hourType] as number) || 0;
                      return (
                        <label key={hourType} className="min-w-0 space-y-1">
                          <span className="block truncate text-xs font-medium text-muted-foreground">{hourType}</span>
                          <RateCell
                            position={position}
                            dayType={dayType}
                            hourType={hourType}
                            rate={rate}
                            readOnly={readOnly}
                            validation={validation}
                            originalRates={originalRates}
                            onChange={onRateChange}
                          />
                        </label>
                      );
                    })}
                  </div>
                </fieldset>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}
