import { memo, useCallback, useMemo, useEffect, useState } from 'react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { TableCell, TableRow } from '@/components/ui/table';
import { AlertTriangle, Trash2, Undo2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import type { TimesheetEntry } from '../types/multi-timesheet.types';
import { findPreferredDayType } from '@/utils/vietnamese';
import { handleNumericInputKeyDown } from '@/utils/numericInput';
import { hasEntryChangedFromOriginal } from '../hooks/utils/entryChange';
import { formatCurrency as formatVND } from '@/utils/formatters';

interface EntryRowProps {
  entry: TimesheetEntry;
  entryIndex: number;
  entryValidation?: { valid: boolean; errors: string[] };
  getDayTypesForPosition: (position: string) => string[];
  getHourTypesForEntry: (position: string, dayType: string) => string[];
  hasPayRateForEntry: (position: string, dayType: string, hourType: string) => boolean;
  getPayRateForEntry?: (position: string, dayType: string, hourType: string) => number;
  onEntryChange: (entryIndex: number, field: keyof TimesheetEntry, value: unknown) => void;
  onRemoveEntry: (entryIndex: number) => void;
  onPreviewRequest?: () => void;
  isLoading?: boolean;
  hideEmployeeSelector?: boolean;
  rowErrors?: string[] | null;
}

export const EntryRow = memo(({
  entry,
  entryIndex,
  entryValidation,
  getDayTypesForPosition,
  getHourTypesForEntry,
  hasPayRateForEntry,
  getPayRateForEntry,
  onEntryChange,
  onRemoveEntry,
  onPreviewRequest,
  isLoading = false,
  rowErrors,
}: EntryRowProps) => {
  const [rawInputValues, setRawInputValues] = useState<Record<string, string>>({});
  const externallyInitializedRef = useState<Set<string>>(() => new Set())[0];

  const hasErrors = useMemo(() =>
    (entryValidation ? !entryValidation.valid && entryValidation.errors.length > 0 : false) ||
    !!(rowErrors && rowErrors.length > 0),
    [entryValidation, rowErrors]
  );

  const isMarkedForDeletion = useMemo(() =>
    !!entry.originalValues && (!entry.hours || Object.keys(entry.hours).length === 0 || Object.values(entry.hours).every(v => v === 0)),
    [entry.originalValues, entry.hours]
  );

  // Existing entry that has been modified (but not fully zeroed)
  const isEdited = useMemo(() =>
    !isMarkedForDeletion && !!entry.originalValues && hasEntryChangedFromOriginal(entry),
    [isMarkedForDeletion, entry]
  );

  const totalHours = useMemo(() => {
    if (!entry.hours) return 0;
    return Object.values(entry.hours).reduce((sum, h) => sum + h, 0);
  }, [entry.hours]);

  const hourStatus = useMemo(() => {
    if (totalHours > 16) return 'excessive';
    if (totalHours > 12) return 'exceeded';
    return 'normal';
  }, [totalHours]);

  const handleFieldChange = useCallback((field: keyof TimesheetEntry, value: unknown) => {
    onEntryChange(entryIndex, field, value);
  }, [onEntryChange, entryIndex]);

  const handleHourTypeHoursChange = useCallback((hourType: string, value: number | null) => {
    const updatedHours = { ...entry.hours };
    if (value === null || value < 0) {
      delete updatedHours[hourType];
    } else if (value === 0) {
      // Keep the key with value 0 for existing entries (signals deletion to API)
      // For new entries, remove the key (no-op)
      if (entry.originalValues) {
        updatedHours[hourType] = 0;
      } else {
        delete updatedHours[hourType];
      }
    } else {
      updatedHours[hourType] = value;
    }
    handleFieldChange('hours', updatedHours);
  }, [handleFieldChange, entry.hours, entry.originalValues]);

  const handleHourTypeHoursBlur = useCallback(() => {
    setTimeout(() => { onPreviewRequest?.(); }, 0);
  }, [onPreviewRequest]);

  const handleHourTypeHoursFocus = useCallback((hourType: string) => {
    const currentValue = entry.hours?.[hourType] ?? 0;
    if (currentValue === 0) {
      setRawInputValues(prev => ({ ...prev, [hourType]: '' }));
    }
  }, [entry.hours]);

  const handleDayTypeChange = useCallback((value: string) => {
    handleFieldChange('dayType', value);
    setTimeout(() => { onPreviewRequest?.(); }, 0);
  }, [handleFieldChange, onPreviewRequest]);

  const handleRemove = useCallback(() => {
    onRemoveEntry(entryIndex);
    setTimeout(() => { onPreviewRequest?.(); }, 0);
  }, [onRemoveEntry, entryIndex, onPreviewRequest]);

  // Restore original hours when undoing deletion of an existing entry
  const handleUndoDelete = useCallback(() => {
    if (entry.originalValues) {
      const restored = { ...entry.originalValues.hours };
      onEntryChange(entryIndex, 'hours', restored);
      // Force raw inputs to reflect restored values immediately
      const newRaw: Record<string, string> = {};
      Object.entries(restored).forEach(([ht, v]) => {
        newRaw[ht] = v > 0 ? String(v) : '';
        externallyInitializedRef.delete(ht); // allow re-sync
      });
      setRawInputValues(prev => ({ ...prev, ...newRaw }));
      setTimeout(() => { onPreviewRequest?.(); }, 0);
    }
  }, [entry.originalValues, onEntryChange, entryIndex, onPreviewRequest, externallyInitializedRef]);

  // Restore a single hour type to its original value
  const handleUndoHourType = useCallback((hourType: string) => {
    if (!entry.originalValues) return;
    const originalValue = entry.originalValues.hours?.[hourType] ?? 0;
    const updatedHours = { ...entry.hours, [hourType]: originalValue };
    if (originalValue === 0) delete updatedHours[hourType];
    onEntryChange(entryIndex, 'hours', updatedHours);
    const rawVal = originalValue > 0 ? String(originalValue) : '';
    externallyInitializedRef.delete(hourType);
    setRawInputValues(prev => ({ ...prev, [hourType]: rawVal }));
    setTimeout(() => { onPreviewRequest?.(); }, 0);
  }, [entry.originalValues, entry.hours, onEntryChange, entryIndex, onPreviewRequest, externallyInitializedRef]);

  const weekdayInfo = useMemo(() => {
    if (!entry.date) return null;
    const date = new Date(entry.date);
    if (Number.isNaN(date.getTime())) return null;
    const dayIndex = date.getDay();
    const labels = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'] as const;
    return { label: labels[dayIndex], isSunday: dayIndex === 0, isSaturday: dayIndex === 6 };
  }, [entry.date]);

  const formattedDate = useMemo(() => {
    if (!entry.date) return '';
    const date = new Date(entry.date);
    if (Number.isNaN(date.getTime())) return entry.date;
    return `${String(date.getDate()).padStart(2, '0')}/${String(date.getMonth() + 1).padStart(2, '0')}`;
  }, [entry.date]);

  const dayTypes = useMemo(() => {
    if (!entry.position) return [];
    return getDayTypesForPosition(entry.position);
  }, [entry.position, getDayTypesForPosition]);

  const hourTypes = useMemo(() => {
    if (!entry.position || !entry.dayType) return [];
    return getHourTypesForEntry(entry.position, entry.dayType);
  }, [entry.position, entry.dayType, getHourTypesForEntry]);

  const createHourTypeHandler = useCallback((hourType: string) => ({
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => {
      const rawValue = e.target.value;
      setRawInputValues(prev => ({ ...prev, [hourType]: rawValue }));
      const numericValue = rawValue === '' ? 0 : parseFloat(rawValue) || 0;
      handleHourTypeHoursChange(hourType, numericValue);
    },
    onKeyDown: handleNumericInputKeyDown,
  }), [handleHourTypeHoursChange]);

  // Weekend-aware default day type: T7/CN → "ngày nghỉ", weekdays → "ngày thường"
  const WEEKEND_DAY_TYPES = useMemo(() => ['ngay nghi', 'ngày nghỉ'], []);

  useEffect(() => {
    if (!entry.position || entry.dayType || dayTypes.length === 0) return;
    const isWeekendDay = weekdayInfo?.isSaturday || weekdayInfo?.isSunday;
    const preferred = isWeekendDay ? WEEKEND_DAY_TYPES : undefined;
    handleFieldChange('dayType', findPreferredDayType(dayTypes, preferred ?? ['ngay thuong', 'ngày thường']));
  }, [entry.position, entry.dayType, dayTypes, handleFieldChange, weekdayInfo, WEEKEND_DAY_TYPES]);

  useEffect(() => {
    // When marked for deletion, zero out all raw inputs and reset init tracking
    if (isMarkedForDeletion) {
      const zeroed: Record<string, string> = {};
      hourTypes.forEach(ht => {
        zeroed[ht] = '';
        externallyInitializedRef.delete(ht);
      });
      setRawInputValues(prev => ({ ...prev, ...zeroed }));
      return;
    }

    if (!entry.hours) return;
    const newRawValues: Record<string, string> = {};
    Object.entries(entry.hours).forEach(([hourType, value]) => {
      if (!externallyInitializedRef.has(hourType)) {
        newRawValues[hourType] = value > 0 ? value.toString() : '';
        externallyInitializedRef.add(hourType);
      }
    });
    if (Object.keys(newRawValues).length > 0) {
      setRawInputValues(prev => ({ ...prev, ...newRawValues }));
    }
  // externallyInitializedRef is a stable Set ref — intentionally excluded
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [entry.hours, isMarkedForDeletion, hourTypes]);

  const isWeekend = weekdayInfo?.isSaturday || weekdayInfo?.isSunday;

  // New entry (no originalValues) that has hours entered
  const isNewWithData = !entry.originalValues && totalHours > 0;

  const rowBg = hasErrors
    ? 'bg-red-50/60'
    : hourStatus === 'excessive'
      ? 'bg-red-50/40'
      : hourStatus === 'exceeded'
        ? 'bg-amber-50/40'
        : weekdayInfo?.isSunday
          ? 'bg-red-50/30'
          : weekdayInfo?.isSaturday
            ? 'bg-amber-50/40'
            : 'bg-background';

  return (
    <>
      {hasErrors && (entryValidation?.errors.length || (rowErrors && rowErrors.length > 0)) && (
        <TableRow className="border-0 border-b border-red-200">
          <TableCell colSpan={3} className="py-2 px-3 bg-red-50">
            <div className="flex flex-col gap-0.5">
              {entryValidation && !entryValidation.valid && entryValidation.errors.map((err, i) => (
                <p key={`ev-${i}`} className="text-xs font-medium text-red-700 flex items-center gap-1.5">
                  <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                  {typeof err === 'string' ? err : 'Có lỗi xảy ra'}
                </p>
              ))}
              {rowErrors && rowErrors.map((err, i) => (
                <p key={`re-${i}`} className="text-xs font-medium text-red-700 flex items-center gap-1.5">
                  <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
                  {err}
                </p>
              ))}
            </div>
          </TableCell>
        </TableRow>
      )}

      <TableRow className={cn(rowBg, 'group border-border/50 align-top')}>
        {/* Date cell */}
        <TableCell className="py-3 px-3 w-[76px]">
          <div className="flex items-center gap-1.5 pt-0.5">
            <span className={cn(
              'text-xs font-medium tabular-nums',
              isMarkedForDeletion ? 'text-muted-foreground/50 line-through' : isWeekend ? 'text-orange-700' : 'text-foreground'
            )}>
              {formattedDate}
            </span>
            {weekdayInfo && (
              <span className={cn(
                'text-[11px] font-medium px-1.5 py-0.5 rounded-full leading-none',
                weekdayInfo.isSunday ? 'bg-red-100 text-red-700' :
                weekdayInfo.isSaturday ? 'bg-orange-100 text-orange-700' :
                'bg-muted text-muted-foreground'
              )}>
                {weekdayInfo.label}
              </span>
            )}
          </div>
        </TableCell>

        {/* Day type — compact select */}
        <TableCell className="py-3 px-2 w-px whitespace-nowrap align-top">
          {dayTypes.length === 0 ? (
            <span className="text-xs text-muted-foreground">—</span>
          ) : (
            <Select value={entry.dayType || ''} onValueChange={handleDayTypeChange} disabled={isLoading}>
              <SelectTrigger aria-label={`Loại ngày cho ${entry.employee?.fullname || "nhân viên"}, ${entry.date}`} className="h-6 text-xs border border-transparent bg-transparent px-2 shadow-none focus:ring-0 hover:border-input hover:bg-background rounded transition-colors w-auto max-w-[105px]">
                <SelectValue placeholder="Loại ngày" />
              </SelectTrigger>
              <SelectContent>
                {/* Warn user if existing data — changing day type requires deleting hours first */}
                {entry.originalValues && totalHours > 0 && (
                  <div className="px-2 py-1.5 mb-1 border-b border-amber-100 bg-amber-50 flex items-start gap-1.5">
                    <AlertTriangle className="h-3 w-3 text-amber-500 shrink-0 mt-0.5" />
                    <p className="text-xs text-amber-700 leading-tight">
                      Xóa giờ công trước khi đổi loại ngày
                    </p>
                  </div>
                )}
                {dayTypes.map((dt) => (
                  <SelectItem key={dt} value={dt} className="text-xs">{dt}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </TableCell>

        {/* Hour inputs */}
        <TableCell className="py-2.5 px-2">
          {hourTypes.length === 0 ? (
            <span className="text-xs text-muted-foreground/40">—</span>
          ) : (
            <div className="flex items-end gap-1.5">
              <div className="flex-1 flex flex-wrap items-start gap-x-2 gap-y-1">
                {hourTypes.map((hourType) => {
                  const hourValue = entry.hours?.[hourType] || 0;
                  const isDisabled = isLoading || !hasPayRateForEntry(entry.position, entry.dayType || '', hourType);

                  const originalValue = entry.originalValues?.hours?.[hourType] ?? 0;
                  const thisValueChanged = !!entry.originalValues && hourValue !== originalValue;
                  const thisFieldDeleted = !!entry.originalValues && originalValue > 0 && hourValue === 0;
                  const thisFieldEdited = thisValueChanged && !thisFieldDeleted;
                  const thisFieldNew = !entry.originalValues && hourValue > 0;

                  const boxStyle = isDisabled
                    ? 'border-border/40 bg-muted/30 text-muted-foreground/50'
                    : thisFieldDeleted
                      ? 'border-red-300 bg-red-50/70 text-red-400'
                      : thisFieldEdited
                        ? 'border-amber-300 bg-amber-50/70 text-amber-700 font-semibold'
                        : thisFieldNew
                          ? 'border-emerald-300 bg-emerald-50/70 text-emerald-700 font-semibold'
                          : hourValue > 16
                            ? 'border-red-300 bg-red-50/60 text-red-700 font-bold'
                            : hourValue > 12
                              ? 'border-amber-300 bg-amber-50/60 text-amber-600 font-semibold'
                              : hourValue > 0
                                ? 'border-emerald-300 bg-emerald-50/60 text-emerald-700 font-semibold'
                                : 'border-border/60 text-muted-foreground';

                  const slotEarnings = !isDisabled && hourValue > 0 && getPayRateForEntry
                    ? hourValue * getPayRateForEntry(entry.position, entry.dayType || '', hourType)
                    : 0;

                  return (
                    <div key={hourType} className={cn('flex flex-col gap-0.5', isDisabled && 'opacity-40')}>
                      <div className="flex items-center gap-1">
                        <span className="text-xs text-muted-foreground whitespace-nowrap select-none">{hourType}</span>
                        <input
                          type="text"
                          inputMode="decimal"
                          min="0"
                          max="24"
                          step="0.5"
                          value={rawInputValues[hourType] ?? (hourValue > 0 ? hourValue.toString() : '')}
                          {...createHourTypeHandler(hourType)}
                          onFocus={() => handleHourTypeHoursFocus(hourType)}
                          onBlur={handleHourTypeHoursBlur}
                          disabled={isDisabled}
                          placeholder="0"
                          className={cn(
                            'w-9 h-6 px-1 text-center text-xs tabular-nums rounded border',
                            'hover:border-input hover:bg-background',
                            'focus:border-primary focus:bg-background focus:outline-none focus:ring-1 focus:ring-primary/30',
                            'transition-all duration-150 placeholder:text-muted-foreground/30',
                            boxStyle,
                            isDisabled && 'cursor-not-allowed'
                          )}
                        />
                        {thisValueChanged && !isDisabled && (
                          <button
                            onClick={() => handleUndoHourType(hourType)}
                            disabled={isLoading}
                            className="flex items-center gap-0.5 text-[11px] font-medium text-muted-foreground hover:text-foreground transition-colors leading-none"
                            title={`Hoàn tác ${hourType}`}
                          >
                            <Undo2 className="h-2.5 w-2.5" />
                            {originalValue > 0 ? originalValue : '—'}
                          </button>
                        )}
                      </div>
                      {slotEarnings > 0 && (
                        <span className="text-[11px] text-emerald-700 tabular-nums leading-none pl-0.5">
                          {formatVND(slotEarnings)}
                        </span>
                      )}
                    </div>
                  );
                })}
                {hourStatus !== 'normal' && (
                  <span className={cn(
                    'inline-flex items-center gap-0.5 text-[11px] font-bold px-1 py-0.5 rounded-full self-start',
                    hourStatus === 'excessive' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-600'
                  )}>
                    <AlertTriangle className="h-2 w-2" />
                    {totalHours}h
                  </span>
                )}
              </div>
              <div className="shrink-0 self-end pb-0.5">
                {isMarkedForDeletion ? (
                  <button
                    onClick={handleUndoDelete}
                    disabled={isLoading}
                    className="h-5 w-5 rounded flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted transition-colors disabled:pointer-events-none"
                    title="Hoàn tác xóa"
                    aria-label="Hoàn tác xóa dòng chấm công"
                  >
                    <Undo2 className="h-3 w-3" />
                  </button>
                ) : (entry.originalValues || totalHours > 0) && (
                  <button
                    onClick={handleRemove}
                    disabled={isLoading}
                    className="h-5 w-5 rounded flex items-center justify-center opacity-0 group-hover:opacity-100 text-muted-foreground/40 group-hover:text-destructive/60 group-hover:bg-destructive/10 transition-all disabled:pointer-events-none"
                    title="Xóa dòng"
                    aria-label="Xóa dòng chấm công"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                )}
              </div>
            </div>
          )}
        </TableCell>
      </TableRow>
    </>
  );
});

EntryRow.displayName = 'EntryRow';
