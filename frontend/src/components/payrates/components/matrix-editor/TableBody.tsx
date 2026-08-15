import React from 'react';
import { TableBody as ShadcnTableBody, TableCell, TableRow } from '@/components/ui/table';
import { PositionCell } from './PositionCell';
import { RateCell } from './RateCell';
import { cn } from '@/lib/utils';
import type { PayrateStructure, DayType, ValidationResult } from '../../types';
import { ALL_DAY_TYPES } from '../../types';

const DAY_LABELS: Record<string, string> = {
  'ngày thường': 'Thường',
  'ngày nghỉ': 'Nghỉ',
  'ngày lễ': 'Lễ',
};

interface TableBodyProps {
  positions: string[];
  hourTypes: string[];
  rates: PayrateStructure;
  originalRates?: PayrateStructure;
  readOnly?: boolean;
  validation?: ValidationResult;
  isFlexible?: boolean;
  editingPosition: string | null;
  editingPositionValue: string;
  onEditPosition: (position: string) => void;
  onSavePosition: () => void;
  onRemovePosition: (position: string) => void;
  onCopyRates: (rates: PayrateStructure) => void;
  onRateChange: (position: string, dayType: DayType, hourType: string, value: string) => void;
  setEditingPositionValue: (value: string) => void;
  setEditingPosition: (position: string | null) => void;
}

export function TableBody({
  positions, hourTypes, rates, originalRates, readOnly = false, validation,
  isFlexible = false,
  editingPosition, editingPositionValue,
  onEditPosition, onSavePosition, onRemovePosition, onCopyRates, onRateChange,
  setEditingPositionValue, setEditingPosition,
}: TableBodyProps) {

  // Flexible projects: one clean row per position, no day-type column
  if (isFlexible) {
    return (
      <ShadcnTableBody>
        {positions.map((position, posIndex) => (
          <TableRow
            key={position}
            className={cn(
              "border-b border-border/30 transition-colors hover:bg-muted/20",
              posIndex > 0 && "border-t border-t-border/40"
            )}
          >
            {/* Position cell */}
            <TableCell className="sticky left-0 z-10 w-44 min-w-44 border-r border-border/60 bg-card px-4 py-3 align-middle">
              <PositionCell
                position={position}
                positionIndex={posIndex}
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
            </TableCell>

            {/* One rate cell per shift — always reads from 'ngày thường' */}
            {hourTypes.map(hourType => {
              const rate = (rates[position]?.['ngày thường']?.[hourType] as number) ?? 0;
              return (
                <TableCell key={hourType} className="px-2 py-2 min-w-[140px]">
                  <RateCell
                    position={position}
                    dayType={'ngày thường'}
                    hourType={hourType}
                    rate={rate}
                    readOnly={readOnly}
                    validation={validation}
                    originalRates={originalRates}
                    onChange={onRateChange}
                  />
                </TableCell>
              );
            })}
          </TableRow>
        ))}
      </ShadcnTableBody>
    );
  }

  // Standard (non-flexible): 3 rows per position for Thường / Nghỉ / Lễ
  return (
    <ShadcnTableBody>
      {positions.map((position, posIndex) => (
        <React.Fragment key={position}>
          {ALL_DAY_TYPES.map((dayType, dayIndex) => (
            <TableRow
              key={`${position}-${dayType}`}
              className={cn(
                "border-b border-border/30 transition-colors",
                dayIndex === 0 && posIndex > 0 && "border-t-2 border-t-border/60",
                "hover:bg-muted/20"
              )}
            >
              {/* Position cell — spans all 3 day rows */}
              {dayIndex === 0 && (
                <TableCell
                  className="sticky left-0 z-10 w-44 min-w-44 border-r border-border/60 bg-card px-4 py-3 align-middle"
                  rowSpan={ALL_DAY_TYPES.length}
                >
                  <PositionCell
                    position={position}
                    positionIndex={posIndex}
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
                </TableCell>
              )}

              {/* Day type */}
              <TableCell className="w-28 min-w-28 px-3 py-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {DAY_LABELS[dayType]}
                </span>
              </TableCell>

              {/* Rate cells */}
              {hourTypes.map(hourType => {
                const rate = (rates[position]?.[dayType]?.[hourType] as number) || 0;
                return (
                  <TableCell key={hourType} className="min-w-[9.5rem] px-3 py-2">
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
                  </TableCell>
                );
              })}
            </TableRow>
          ))}
        </React.Fragment>
      ))}
    </ShadcnTableBody>
  );
}
