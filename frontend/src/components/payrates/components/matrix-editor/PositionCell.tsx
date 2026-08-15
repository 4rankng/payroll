import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { MoreHorizontal } from 'lucide-react';
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
    <div className="relative block min-w-0" data-slot="payrate-position-cell">
      {editingPosition === position ? (
        <Input
          type="text"
          value={editingPositionValue}
          onChange={e => setEditingPositionValue(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.stopPropagation(); onSavePosition(); } }}
          onBlur={onSavePosition}
          className="h-11 border-primary bg-background px-3 text-sm font-semibold"
          autoFocus
        />
      ) : (
        <>
          <p className="min-w-0 break-words px-8 text-center text-sm font-semibold leading-5 text-foreground">{position}</p>
          {!readOnly && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="absolute right-0 top-1/2 h-11 w-11 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label={`Tùy chọn cho vị trí ${position}`}
                >
                  <MoreHorizontal className="h-4 w-4" aria-hidden="true" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="min-w-48">
                <DropdownMenuItem onSelect={() => onEditPosition(position)}>
                  Đổi tên
                </DropdownMenuItem>
                {positionIndex > 0 && firstPosition && (
                  <DropdownMenuItem onSelect={handleCopyFromFirst}>
                    Sao chép mức lương từ {firstPosition}
                  </DropdownMenuItem>
                )}
                {positions.length > 1 && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onSelect={() => onRemovePosition(position)}
                      className="text-destructive focus:text-destructive"
                    >
                      Xóa vị trí
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </>
      )}
    </div>
  );
}
