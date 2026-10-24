import { useState } from 'react';
import type { PayrateStructure } from '../types';
import { addPositionToRates, removePositionFromRates, getPositionsFromRates } from '../types';
import { normalizePayrateKey, isPayrateKeyAvailable } from '@/utils/payrateKey';

export function usePositionManager(
  rates: PayrateStructure,
  onChange: (rates: PayrateStructure) => void,
  readOnly: boolean = false
) {
  const [newPosition, setNewPosition] = useState('');
  const [showAddPosition, setShowAddPosition] = useState(false);
  const [editingPosition, setEditingPosition] = useState<string | null>(null);
  const [editingPositionValue, setEditingPositionValue] = useState('');

  const positions = getPositionsFromRates(rates);

  const handleAddPosition = () => {
    const candidate = normalizePayrateKey(newPosition);
    if (!candidate || readOnly) return;
    // Fold-based duplicate guard: a case/diacritic variant of an existing
    // position is the same position — skip, mirroring the exact-key no-op.
    if (!isPayrateKeyAvailable(positions, candidate)) {
      setNewPosition('');
      setShowAddPosition(false);
      return;
    }

    const newRates = addPositionToRates(rates, candidate);
    onChange(newRates);
    setNewPosition('');
    setShowAddPosition(false);
  };

  const handleRemovePosition = (position: string) => {
    if (positions.length <= 1 || readOnly) return; // Keep at least one position

    const newRates = removePositionFromRates(rates, position);
    onChange(newRates);
  };

  const handleEditPosition = (position: string) => {
    if (readOnly) return;
    setEditingPosition(position);
    setEditingPositionValue(position);
  };

  const handleSavePosition = () => {
    const candidate = normalizePayrateKey(editingPositionValue);
    if (!editingPosition || !candidate || candidate === editingPosition || readOnly) {
      setEditingPosition(null);
      return;
    }
    if (!isPayrateKeyAvailable(positions, candidate, editingPosition)) {
      setEditingPosition(null);
      return;
    }

    const newRates = { ...rates };

    // Rename the position
    if (newRates[editingPosition]) {
      newRates[candidate] = newRates[editingPosition];
      delete newRates[editingPosition];
    }

    onChange(newRates);
    setEditingPosition(null);
    setEditingPositionValue('');
  };

  const handleCancelEditPosition = () => {
    setEditingPosition(null);
    setEditingPositionValue('');
  };

  return {
    // State
    newPosition,
    showAddPosition,
    editingPosition,
    editingPositionValue,
    positions,
    
    // Setters
    setNewPosition,
    setShowAddPosition,
    setEditingPositionValue,
    setEditingPosition,
    
    // Actions
    handleAddPosition,
    handleRemovePosition,
    handleEditPosition,
    handleSavePosition,
    handleCancelEditPosition,
  };
}