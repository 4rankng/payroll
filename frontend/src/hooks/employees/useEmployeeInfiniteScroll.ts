import { useState, useEffect, useCallback, useRef } from 'react';
import { useIsMobile } from '@/hooks/use-mobile';
import type { Employee } from '@/types/api/employee.types';

interface UseEmployeeInfiniteScrollProps {
  filteredEmployees: Employee[];
  itemsPerPage?: number;
}

export const useEmployeeInfiniteScroll = ({
  filteredEmployees,
  itemsPerPage = 20,
}: UseEmployeeInfiniteScrollProps) => {
  const isMobile = useIsMobile();
  const [currentPage, setCurrentPage] = useState(1);
  const [displayedEmployees, setDisplayedEmployees] = useState<Employee[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  // BUG-008 fix: Track mounted state to prevent updates on unmounted component
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
    };
  }, []);

  // Reset pagination when filters change
  useEffect(() => {
    setCurrentPage(1);
    // Remove duplicates from initial items based on employee ID
    const uniqueEmployees = filteredEmployees.filter((emp, index, arr) => 
      arr.findIndex(e => e.id === emp.id) === index
    );
    const initialItems = uniqueEmployees.slice(0, itemsPerPage);
    setDisplayedEmployees(initialItems);
    setHasMore(uniqueEmployees.length > itemsPerPage);
  }, [filteredEmployees, itemsPerPage]);

  // Load more function for infinite scroll
  const loadMore = useCallback(async () => {
    if (isLoadingMore || !hasMore) return;

    setIsLoadingMore(true);
    
    // Simulate API delay
    await new Promise(resolve => setTimeout(resolve, 500));

    // BUG-008 fix: Check if still mounted before updating state
    if (!mountedRef.current) return;

    // Remove duplicates from filtered employees first
    const uniqueEmployees = filteredEmployees.filter((emp, index, arr) =>
      arr.findIndex(e => e.id === emp.id) === index
    );

    const nextPage = currentPage + 1;
    const startIdx = currentPage * itemsPerPage;
    const endIdx = startIdx + itemsPerPage;
    const nextItems = uniqueEmployees.slice(startIdx, endIdx);

    if (nextItems.length > 0) {
      setDisplayedEmployees(prev => {
        // Create a Set of existing IDs to prevent duplicates
        const existingIds = new Set(prev.map(emp => emp.id));
        const uniqueNewItems = nextItems.filter(emp => !existingIds.has(emp.id));
        return [...prev, ...uniqueNewItems];
      });
      setCurrentPage(nextPage);
      setHasMore(endIdx < uniqueEmployees.length);
    } else {
      setHasMore(false);
    }

    setIsLoadingMore(false);
  }, [currentPage, filteredEmployees, hasMore, isLoadingMore, itemsPerPage]);

  // Return the appropriate data based on device type
  // Ensure no duplicates in the data regardless of device type
  const deduplicatedFilteredEmployees = filteredEmployees.filter((emp, index, arr) => 
    arr.findIndex(e => e.id === emp.id) === index
  );
  const dataToDisplay = isMobile ? displayedEmployees : deduplicatedFilteredEmployees;

  return {
    displayedEmployees,
    dataToDisplay,
    hasMore,
    isLoadingMore,
    loadMore,
    isMobile,
  };
};