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
          className="h-11 border-primary bg-transparent p-2 text-xs font-semibold"
          autoFocus
        />
      ) : (
        <div className="flex items-center gap-1">
          <span className="flex-1 break-words text-xs font-semibold capitalize">{position}</span>
          {!readOnly && (
            <div className="flex items-center gap-1 opacity-100 transition-opacity sm:opacity-0 sm:group-hover/pos:opacity-100">
              <button
                onClick={() => onEditPosition(position)}
                className="flex h-11 w-11 items-center justify-center rounded text-muted-foreground hover:bg-muted hover:text-foreground"
                title="Đổi tên"
              >
                <Pencil className="h-2.5 w-2.5" />
              </button>
              {positions.length > 1 && (
                <button
                  onClick={e => { e.preventDefault(); e.stopPropagation(); onRemovePosition(position); }}
                  className="flex h-11 w-11 items-center justify-center rounded text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
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
          className="flex min-h-11 items-center gap-1 rounded px-2 text-[10px] text-muted-foreground transition-colors hover:bg-muted hover:text-primary"
          title={`Sao chép từ ${firstPosition}`}
        >
          <Copy className="h-2.5 w-2.5" />
          Sao chép
        </button>
      )}
    </div>
  );
}
