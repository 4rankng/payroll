import { useState, useMemo, useCallback } from 'react';
import type { PayrateStructure, DayType } from '../types';
import { getAllHourTypes, getPositionsFromRates, isTimeRange, TIME_RANGE_RE, suggestTimeRange } from '../types';

export function useHourTypeManager(
  rates: PayrateStructure,
  onChange: (rates: PayrateStructure) => void,
  readOnly: boolean = false
) {
  const [newHourType, setNewHourType] = useState('');
  const [showAddHourType, setShowAddHourType] = useState(false);
  const [editingHourType, setEditingHourType] = useState<string | null>(null);
  const [editingHourTypeValue, setEditingHourTypeValue] = useState('');
  const [editingHourTypeError, setEditingHourTypeError] = useState<string | null>(null);
  const [editingHourTypeSuggestion, setEditingHourTypeSuggestion] = useState<string | null>(null);

  // Memoized derived state — avoids full tree traversal per keystroke
  const hourTypes = useMemo(() => getAllHourTypes(rates), [rates]);
  const positions = useMemo(() => getPositionsFromRates(rates), [rates]);

  // Derive active day types from the actual rates structure.
  // Prevents creating ghost 'ngày nghỉ'/'ngày lễ' entries for flexible projects.
  const activeDayTypes = useMemo((): DayType[] => {
    for (const pos of Object.values(rates)) {
      if (pos && typeof pos === 'object') {
        return Object.keys(pos) as DayType[];
      }
    }
    return ['ngày thường'] as DayType[];
  }, [rates]);

  // Consolidated editing-state reset
  const clearEditingState = useCallback(() => {
    setEditingHourType(null);
    setEditingHourTypeValue('');
    setEditingHourTypeError(null);
    setEditingHourTypeSuggestion(null);
  }, []);

  // Clears error when user modifies the value
  const handleSetEditingHourTypeValue = (value: string) => {
    setEditingHourTypeValue(value);
    if (editingHourTypeError) {
      setEditingHourTypeError(null);
      setEditingHourTypeSuggestion(null);
    }
  };

  const handleAddHourType = () => {
    if (!newHourType.trim() || readOnly) return;

    const hourType = newHourType.trim();
    const newRates = { ...rates };

    positions.forEach(position => {
      activeDayTypes.forEach(dayType => {
        if (!newRates[position][dayType]) {
          newRates[position][dayType] = {};
        }
        newRates[position][dayType][hourType] = 0;
      });
    });

    onChange(newRates);
    setNewHourType('');
    setShowAddHourType(false);
  };

  const handleRemoveHourType = (hourTypeToRemove: string) => {
    if (hourTypes.length <= 1 || readOnly) return;

    const newRates = { ...rates };

    positions.forEach(position => {
      activeDayTypes.forEach(dayType => {
        if (newRates[position][dayType]) {
          delete newRates[position][dayType][hourTypeToRemove];
        }
      });
    });

    onChange(newRates);
  };

  const handleEditHourType = (oldHourType: string) => {
    if (readOnly) return;
    setEditingHourType(oldHourType);
    setEditingHourTypeValue(oldHourType);
    setEditingHourTypeError(null);
    setEditingHourTypeSuggestion(null);
  };

  const handleSaveHourType = () => {
    if (!editingHourType || readOnly) {
      clearEditingState();
      return;
    }

    const trimmed = editingHourTypeValue.trim();

    // Unchanged or empty → just close
    if (!trimmed || trimmed === editingHourType) {
      clearEditingState();
      return;
    }

    // If the original key was a time range, validate the new value strictly
    if (isTimeRange(editingHourType) && !TIME_RANGE_RE.test(trimmed)) {
      const suggestion = suggestTimeRange(trimmed);
      setEditingHourTypeError('Sai định dạng — cần HH:MM-HH:MM (24 giờ)');
      setEditingHourTypeSuggestion(suggestion);
      return; // Keep edit mode open so user can fix it
    }

    const newRates = { ...rates };

    positions.forEach(position => {
      activeDayTypes.forEach(dayType => {
        if (newRates[position][dayType] && editingHourType in newRates[position][dayType]) {
          const rate = newRates[position][dayType][editingHourType];
          delete newRates[position][dayType][editingHourType];
          newRates[position][dayType][trimmed] = rate;
        }
      });
    });

    onChange(newRates);
    clearEditingState();
  };

  const handleApplySuggestion = () => {
    if (!editingHourTypeSuggestion) return;
    setEditingHourTypeValue(editingHourTypeSuggestion);
    setEditingHourTypeError(null);
    setEditingHourTypeSuggestion(null);
  };

  const handleCancelEditHourType = () => {
    clearEditingState();
  };

  return {
    // State
    newHourType,
    showAddHourType,
    editingHourType,
    editingHourTypeValue,
    editingHourTypeError,
    editingHourTypeSuggestion,
    hourTypes,

    // Setters
    setNewHourType,
    setShowAddHourType,
    setEditingHourTypeValue: handleSetEditingHourTypeValue,
    setEditingHourType,

    // Actions
    handleAddHourType,
    handleRemoveHourType,
    handleEditHourType,
    handleSaveHourType,
    handleApplySuggestion,
    handleCancelEditHourType,
  };
}
