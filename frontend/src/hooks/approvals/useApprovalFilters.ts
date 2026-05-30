import { useState, useMemo } from 'react';
import { ApprovalItem, ApprovalStatus, ApprovalPriority, ApprovalType } from '@/types/approval';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

interface UseApprovalFiltersProps {
  approvalItems: ApprovalItem[];
}

export const useApprovalFilters = ({ approvalItems }: UseApprovalFiltersProps) => {
  const [selectedTab, setSelectedTab] = useState<ApprovalStatus | 'all'>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [priorityFilter, setPriorityFilter] = useState<ApprovalPriority | 'all'>('all');
  const [typeFilter, setTypeFilter] = useState<ApprovalType | 'all'>('all');

  const filteredItems = useMemo(() => {
    let filtered = [...approvalItems];

    // Filter by status (tab)
    if (selectedTab !== 'all') {
      filtered = filtered.filter(item => item.status === selectedTab);
    }

    // Filter by search term
    if (searchTerm.trim()) {
      filtered = filtered.filter(item =>
        (item.title && vietnameseIncludes(item.title, searchTerm)) ||
        (item.description && vietnameseIncludes(item.description, searchTerm)) ||
        (item.submittedBy?.name && vietnameseIncludes(item.submittedBy.name, searchTerm))
      );
    }

    // Filter by priority
    if (priorityFilter !== 'all') {
      filtered = filtered.filter(item => item.priority === priorityFilter);
    }

    // Filter by type
    if (typeFilter !== 'all') {
      filtered = filtered.filter(item => item.type === typeFilter);
    }

    // Sort by priority and date
    filtered.sort((a, b) => {
      const priorityOrder = { urgent: 4, high: 3, medium: 2, low: 1 };
      const aPriority = priorityOrder[a.priority];
      const bPriority = priorityOrder[b.priority];
      
      if (aPriority !== bPriority) {
        return bPriority - aPriority; // Higher priority first
      }
      
      // If same priority, sort by submission time (newest first)
      return a.submittedAt.localeCompare(b.submittedAt);
    });

    return filtered;
  }, [approvalItems, selectedTab, searchTerm, priorityFilter, typeFilter]);

  const stats = useMemo(() => {
    const total = approvalItems.length;
    const pending = approvalItems.filter(item => item.status === 'pending').length;
    const approved = approvalItems.filter(item => item.status === 'approved').length;
    const rejected = approvalItems.filter(item => item.status === 'rejected').length;
    const reviewing = approvalItems.filter(item => item.status === 'reviewing').length;
    const urgent = approvalItems.filter(item => item.priority === 'urgent').length;

    return {
      total,
      pending,
      approved,
      rejected,
      reviewing,
      urgent,
    };
  }, [approvalItems]);

  const clearFilters = () => {
    setSearchTerm('');
    setPriorityFilter('all');
    setTypeFilter('all');
  };

  return {
    selectedTab,
    setSelectedTab,
    searchTerm,
    setSearchTerm,
    priorityFilter,
    setPriorityFilter,
    typeFilter,
    setTypeFilter,
    filteredItems,
    stats,
    clearFilters,
  };
};