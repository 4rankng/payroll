import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { TableHead, TableRow } from '@/components/ui/table';
import { X } from 'lucide-react';

interface TableHeaderProps {
  hourTypes: string[];
  readOnly?: boolean;
  isFlexible?: boolean;
  editingHourType: string | null;
  editingHourTypeValue: string;
  editingHourTypeError?: string | null;
  editingHourTypeSuggestion?: string | null;
  onEditHourType: (hourType: string) => void;
  onSaveHourType: () => void;
  onRemoveHourType: (hourType: string) => void;
  onApplySuggestion?: () => void;
  onCancelEditHourType: () => void;
  setEditingHourTypeValue: (value: string) => void;
}

export function TableHeader({
  hourTypes,
  readOnly = false,
  isFlexible = false,
  editingHourType,
  editingHourTypeValue,
  editingHourTypeError,
  editingHourTypeSuggestion,
  onEditHourType,
  onSaveHourType,
  onRemoveHourType,
  onApplySuggestion,
  onCancelEditHourType,
  setEditingHourTypeValue,
}: TableHeaderProps) {
  return (
    <TableRow className="border-b border-border/60 bg-muted/30">
      <TableHead className="sticky left-0 z-20 w-44 min-w-44 border-r border-border/60 bg-muted/30 px-4 py-3 text-center align-middle text-xs font-semibold text-muted-foreground">
        Vị trí
      </TableHead>
      {!isFlexible && (
        <TableHead className="w-28 min-w-28 px-3 py-3 text-center align-middle text-xs font-semibold text-muted-foreground">
          Loại ngày
        </TableHead>
      )}
      {hourTypes.map(hourType => (
        <TableHead key={hourType} className="relative min-w-[9.5rem] px-3 py-3 text-center align-middle text-xs font-semibold text-muted-foreground group">
          <div className="flex items-center justify-center">
            {editingHourType === hourType ? (
              <Input
                type="text"
                value={editingHourTypeValue}
                onChange={e => setEditingHourTypeValue(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') { e.stopPropagation(); onSaveHourType(); }
                  if (e.key === 'Escape') { e.stopPropagation(); onCancelEditHourType(); }
                }}
                onBlur={onSaveHourType}
                className={`h-11 text-center text-xs font-mono font-semibold focus:ring-1 ${
                  editingHourTypeError
                    ? 'border border-destructive text-destructive bg-destructive/10 focus:ring-destructive'
                    : 'border-0 bg-transparent focus:ring-primary'
                }`}
                autoFocus
              />
            ) : (
              <button
                type="button"
                className={`flex min-h-11 w-full items-center justify-center gap-1 px-10 text-center ${!readOnly ? 'cursor-pointer hover:text-foreground transition-colors' : 'cursor-default'}`}
                onClick={() => !readOnly && onEditHourType(hourType)}
                title={readOnly ? hourType : `Nhấn để đổi tên: ${hourType}`}
              >
                <span className="break-words text-xs font-semibold text-foreground">{hourType}</span>
                <span className="text-[10px] font-normal text-muted-foreground">
                  {isFlexible ? '₫/ca' : '₫/giờ'}
                </span>
              </button>
            )}
            {!readOnly && hourTypes.length > 1 && (
              <button
                onClick={e => { e.preventDefault(); e.stopPropagation(); onRemoveHourType(hourType); }}
                className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded text-muted-foreground opacity-100 transition-opacity hover:bg-destructive/10 hover:text-destructive sm:opacity-0 sm:group-hover:opacity-100"
                title={`Xóa cột ${hourType}`}
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        </TableHead>
      ))}
    </TableRow>
  );
}
