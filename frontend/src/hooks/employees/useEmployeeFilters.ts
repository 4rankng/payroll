import { useState, useEffect } from 'react';
import { Employee, EmployeeFilters } from '@/types/api/employee.types';

interface UseEmployeeFiltersProps {
  employees: Employee[];
  searchEmployees: (searchTerm: string) => void;
  updateFilters: (filters: Partial<EmployeeFilters>) => void;
  clearFilters: () => void;
  currentFilters: EmployeeFilters;
}

export const useEmployeeFilters = ({
  employees,
  searchEmployees,
  updateFilters,
  clearFilters,
  currentFilters,
}: UseEmployeeFiltersProps) => {
  const [searchTerm, setSearchTerm] = useState('');

  // Trigger search when searchTerm changes
  useEffect(() => {
    searchEmployees(searchTerm);
  }, [searchTerm, searchEmployees]);

  // Employees come pre-sorted from the server now
  const filteredEmployees = employees;

  // Filter handlers
  const handleStatusFilterChange = (status: 'working' | 'unassigned' | undefined) => {
    updateFilters({ status });
  };

  const handleMonthChange = (month: string | undefined) => {
    updateFilters({ month });
  };

  const handleProjectChange = (projectId: number | null) => {
    updateFilters({ projectId: projectId || undefined });
  };

  return {
    searchTerm,
    setSearchTerm,
    sortBy: currentFilters.sortBy || 'created_at',
    sortOrder: currentFilters.sortOrder || 'desc',
    filteredEmployees,
    // New filter controls
    statusFilter: currentFilters.status,
    month: currentFilters.month,
    projectId: currentFilters.projectId || null,
    handleStatusFilterChange,
    handleMonthChange,
    handleProjectChange,
    clearAllFilters: clearFilters,
  };
};
