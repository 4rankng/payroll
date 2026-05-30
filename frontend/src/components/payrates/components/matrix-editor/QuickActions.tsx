import { useState } from 'react';
import { Input } from '@/components/ui/input';
import { Plus, X } from 'lucide-react';
import type { PayrateStructure } from '../../types';
import { addPositionToRates, getPositionsFromRates, ALL_DAY_TYPES } from '../../types';
import { getSuggestedPositions, getSuggestedHourTypes } from '../../utils/suggestions';
import { FlexibleShiftManager } from './FlexibleShiftManager';

interface QuickActionsProps {
  rates: PayrateStructure;
  onChange: (rates: PayrateStructure) => void;
  readOnly?: boolean;
  isFlexible?: boolean;
  hourTypes?: string[];
  newPosition: string;
  showAddPosition: boolean;
  setNewPosition: (value: string) => void;
  setShowAddPosition: (show: boolean) => void;
  handleAddPosition: () => void;
  newHourType: string;
  showAddHourType: boolean;
  setNewHourType: (value: string) => void;
  setShowAddHourType: (show: boolean) => void;
  handleAddHourType: () => void;
}

export function QuickActions({
  rates, onChange, readOnly = false, isFlexible = false, hourTypes = [],
  newPosition, showAddPosition, setNewPosition, setShowAddPosition, handleAddPosition,
  newHourType, showAddHourType, setNewHourType, setShowAddHourType, handleAddHourType,
}: QuickActionsProps) {
  const positions = getPositionsFromRates(rates);
  const suggestedPositions = getSuggestedPositions(rates);
  const suggestedHourTypes = getSuggestedHourTypes(rates);

  const handleSuggestedPosition = (pos: string) => onChange(addPositionToRates(rates, pos));

  const handleSuggestedHourType = (hour: string) => {
    const newRates = { ...rates };
    positions.forEach(position => {
      ALL_DAY_TYPES.forEach(dayType => {
        if (!newRates[position][dayType]) newRates[position][dayType] = {};
        newRates[position][dayType][hour] = 0;
      });
    });
    onChange(newRates);
  };

  if (readOnly) return null;

  return (
    <div className={`grid gap-3 pt-1 ${isFlexible ? 'grid-cols-1 sm:grid-cols-[1fr_1.4fr]' : 'grid-cols-1 sm:grid-cols-2'}`}>
      {/* Add position */}
      <div className="rounded-xl border border-border/60 bg-muted/20 p-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Vị trí</span>
          {!showAddPosition && (
            <button
              onClick={() => setShowAddPosition(true)}
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <Plus className="h-3 w-3" />
              Thêm
            </button>
          )}
        </div>

        {showAddPosition && (
          <div className="flex gap-1.5">
            <Input
              type="text"
              placeholder="Tên vị trí..."
              value={newPosition}
              onChange={e => setNewPosition(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') { e.stopPropagation(); handleAddPosition(); } }}
              className="h-7 text-xs flex-1"
              autoFocus
            />
            <button
              onClick={handleAddPosition}
              disabled={!newPosition.trim()}
              className="h-7 px-2.5 text-xs rounded-xl bg-primary text-primary-foreground disabled:opacity-40 hover:bg-primary/90 transition-colors"
            >
              Thêm
            </button>
            <button
              onClick={() => { setShowAddPosition(false); setNewPosition(''); }}
              className="h-7 w-7 flex items-center justify-center rounded-xl border border-border text-muted-foreground hover:text-foreground transition-colors"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        {suggestedPositions.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {suggestedPositions.map(pos => (
              <button
                key={pos}
                onClick={() => handleSuggestedPosition(pos)}
                className="text-[10px] px-2 py-0.5 rounded-full border border-border bg-background hover:border-primary hover:text-primary transition-colors capitalize"
              >
                {pos}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Add hour type — flexible projects get a time-picker UI, others get text input */}
      {isFlexible ? (
        <FlexibleShiftManager
          rates={rates}
          hourTypes={hourTypes}
          onChange={onChange}
          readOnly={readOnly}
        />
      ) : (
        <div className="rounded-xl border border-border/60 bg-muted/20 p-3 space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Khung giờ</span>
            {!showAddHourType && (
              <button
                onClick={() => setShowAddHourType(true)}
                className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                <Plus className="h-3 w-3" />
                Thêm
              </button>
            )}
          </div>

          {showAddHourType && (
            <div className="flex gap-1.5">
              <Input
                type="text"
                placeholder="VD: ca chiều"
                value={newHourType}
                onChange={e => setNewHourType(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') { e.stopPropagation(); handleAddHourType(); } }}
                className="h-7 text-xs flex-1"
                autoFocus
              />
              <button
                onClick={handleAddHourType}
                disabled={!newHourType.trim()}
                className="h-7 px-2.5 text-xs rounded-xl bg-primary text-primary-foreground disabled:opacity-40 hover:bg-primary/90 transition-colors"
              >
                Thêm
              </button>
              <button
                onClick={() => { setShowAddHourType(false); setNewHourType(''); }}
                className="h-7 w-7 flex items-center justify-center rounded-xl border border-border text-muted-foreground hover:text-foreground transition-colors"
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          )}

          {suggestedHourTypes.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {suggestedHourTypes.slice(0, 8).map(hour => (
                <button
                  key={hour}
                  onClick={() => handleSuggestedHourType(hour)}
                  className="text-[10px] px-2 py-0.5 rounded-full border border-border bg-background hover:border-primary hover:text-primary transition-colors"
                >
                  {hour}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
