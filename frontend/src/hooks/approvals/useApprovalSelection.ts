import { useState } from 'react';

export const useApprovalSelection = () => {
  const [selectedItems, setSelectedItems] = useState<string[]>([]);

  const toggleItem = (itemId: string) => {
    setSelectedItems(prev =>
      prev.includes(itemId)
        ? prev.filter(id => id !== itemId)
        : [...prev, itemId]
    );
  };

  const selectAll = (itemIds: string[]) => {
    setSelectedItems(itemIds);
  };

  const clearSelection = () => {
    setSelectedItems([]);
  };

  const isSelected = (itemId: string) => {
    return selectedItems.includes(itemId);
  };

  const isAllSelected = (itemIds: string[]) => {
    return itemIds.length > 0 && itemIds.every(id => selectedItems.includes(id));
  };

  const hasSelection = selectedItems.length > 0;

  return {
    selectedItems,
    toggleItem,
    selectAll,
    clearSelection,
    isSelected,
    isAllSelected,
    hasSelection,
  };
};