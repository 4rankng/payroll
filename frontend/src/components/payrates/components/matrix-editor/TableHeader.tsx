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
      <TableHead className="w-32 text-xs font-semibold text-muted-foreground py-2 px-3">Vị trí</TableHead>
      {!isFlexible && (
        <TableHead className="w-24 text-xs font-semibold text-muted-foreground py-2 px-3">Loại ngày</TableHead>
      )}
      {hourTypes.map(hourType => (
        <TableHead key={hourType} className="min-w-[120px] text-xs font-semibold text-muted-foreground py-2 px-2 group">
          <div className="flex items-center gap-1">
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
                className={`h-6 text-xs p-1 focus:ring-1 font-mono font-semibold ${
                  editingHourTypeError
                    ? 'border border-destructive text-destructive bg-destructive/10 focus:ring-destructive'
                    : 'border-0 bg-transparent focus:ring-primary'
                }`}
                autoFocus
              />
            ) : (
              <span
                className={`text-xs font-semibold truncate ${!readOnly ? 'cursor-pointer hover:text-foreground transition-colors' : ''}`}
                onClick={() => !readOnly && onEditHourType(hourType)}
                title={readOnly ? hourType : `Nhấn để đổi tên: ${hourType}`}
              >
                {hourType}
              </span>
            )}
            {!readOnly && hourTypes.length > 1 && (
              <button
                onClick={e => { e.preventDefault(); e.stopPropagation(); onRemoveHourType(hourType); }}
                className="opacity-0 group-hover:opacity-100 transition-opacity ml-auto h-4 w-4 flex items-center justify-center rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                title={`Xóa cột ${hourType}`}
              >
                <X className="h-2.5 w-2.5" />
              </button>
            )}
          </div>
        </TableHead>
      ))}
    </TableRow>
  );
}
