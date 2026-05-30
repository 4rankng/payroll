import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Copy, X, Pencil } from 'lucide-react';
import type { PayrateStructure } from '../../types';
import { copyRatesFromPosition } from '../../utils/rateOperations';

interface PositionCellProps {
  position: string;
  positionIndex: number;
  firstPosition?: string;
  positions: string[];
  readOnly?: boolean;
  rates: PayrateStructure;
  editingPosition: string | null;
  editingPositionValue: string;
  onEditPosition: (position: string) => void;
  onSavePosition: () => void;
  onRemovePosition: (position: string) => void;
  onCopyRates: (rates: PayrateStructure) => void;
  setEditingPositionValue: (value: string) => void;
  setEditingPosition: (position: string | null) => void;
}

export function PositionCell({
  position, positionIndex, firstPosition, positions, readOnly = false, rates,
  editingPosition, editingPositionValue,
  onEditPosition, onSavePosition, onRemovePosition, onCopyRates,
  setEditingPositionValue, setEditingPosition,
}: PositionCellProps) {
  const handleCopyFromFirst = () => {
    if (firstPosition) onCopyRates(copyRatesFromPosition(rates, firstPosition, position));
  };

  return (
    <div className="flex flex-col gap-1 group/pos">
      {editingPosition === position ? (
        <Input
          type="text"
          value={editingPositionValue}
          onChange={e => setEditingPositionValue(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.stopPropagation(); onSavePosition(); } }}
          onBlur={onSavePosition}
          className="h-6 text-xs border-primary bg-transparent p-1 font-semibold"
          autoFocus
        />
      ) : (
        <div className="flex items-center gap-1">
          <span className="text-xs font-semibold capitalize flex-1 truncate">{position}</span>
          {!readOnly && (
            <div className="flex items-center gap-0.5 opacity-0 group-hover/pos:opacity-100 transition-opacity">
              <button
                onClick={() => onEditPosition(position)}
                className="h-4 w-4 flex items-center justify-center rounded text-muted-foreground hover:text-foreground"
                title="Đổi tên"
              >
                <Pencil className="h-2.5 w-2.5" />
              </button>
              {positions.length > 1 && (
                <button
                  onClick={e => { e.preventDefault(); e.stopPropagation(); onRemovePosition(position); }}
                  className="h-4 w-4 flex items-center justify-center rounded text-muted-foreground hover:text-destructive"
                  title="Xóa vị trí"
                >
                  <X className="h-2.5 w-2.5" />
                </button>
              )}
            </div>
          )}
        </div>
      )}
      {!readOnly && positionIndex > 0 && firstPosition && (
        <button
          onClick={handleCopyFromFirst}
          className="flex items-center gap-1 text-[10px] text-muted-foreground hover:text-primary transition-colors"
          title={`Sao chép từ ${firstPosition}`}
        >
          <Copy className="h-2.5 w-2.5" />
          Sao chép
        </button>
      )}
    </div>
  );
}
