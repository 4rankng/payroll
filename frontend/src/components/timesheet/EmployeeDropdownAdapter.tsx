import { useMemo, useCallback } from 'react';
import { EmployeeSelector } from '@/components/ui/employee-selector';
import { useQuery } from '@tanstack/react-query';
import { employeeService } from '@/services/api/employee.service';
import type { Employee } from '@/types/api/employee.types';

interface EmployeeDropdownAdapterProps {
  value: string; // employee ID as string
  onChange: (value: string) => void; // receives employee ID as string
  selectedProject?: string; // for project filtering
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  // Props to pass through to EmployeeSelector for project filtering
  availableEmployees?: Employee[];
  includeAvailableOnly?: boolean;
  excludeEmployeeIds?: number[];
  showAllOption?: boolean; // New prop to control showing "All Employees" option
  pillStyle?: boolean;
}

export function EmployeeDropdownAdapter({
  value,
  onChange,
  selectedProject,
  placeholder = "Chọn nhân viên...",
  disabled = false,
  className,
  availableEmployees = [],
  includeAvailableOnly = false,
  excludeEmployeeIds = [],
  showAllOption = true, // Default to true for timesheet pages
  pillStyle = false,
}: EmployeeDropdownAdapterProps) {
  // Fetch all employees data for finding selected employee
  const { data: employeesData } = useQuery({
    queryKey: ['employees', 'list', {
      page: 1,
      pageSize: 1000, // Large enough to find selected employee
      status: 'working'
    }],
    queryFn: () => employeeService.getEmployees({
      page: 1,
      pageSize: 1000,
      status: 'working',
      sortBy: 'fullname',
      sortOrder: 'asc'
    }),
    enabled: !!value && value !== 'all' && !includeAvailableOnly
  });

  // Convert string ID to Employee object for EmployeeSelector
  const selectedEmployee = useMemo(() => {
    if (!value || value === 'all') return null;

    // First check available employees (project-filtered)
    if (includeAvailableOnly && availableEmployees.length > 0) {
      return availableEmployees.find(emp => emp.id.toString() === value);
    }

    // Then check all employees data
    return employeesData?.data.find(emp => emp.id.toString() === value);
  }, [value, employeesData, includeAvailableOnly, availableEmployees]);

  // Handle Employee selection and convert to string ID
  const handleEmployeeSelect = useCallback((employee: Employee | null) => {
    if (!employee) {
      onChange('all');
    } else {
      onChange(employee.id.toString());
    }
  }, [onChange]);

  // Create a fake "All Employees" option for display when value is 'all'
  const displayValue = useMemo(() => {
    if (value === 'all') {
      return null; // EmployeeSelector will show placeholder
    }
    return selectedEmployee;
  }, [value, selectedEmployee]);

  return (
    <EmployeeSelector
      value={displayValue}
      onSelect={handleEmployeeSelect}
      placeholder={value === 'all' ? "Tất cả nhân viên" : placeholder}
      disabled={disabled}
      className={className}
      availableEmployees={availableEmployees}
      includeAvailableOnly={includeAvailableOnly}
      excludeEmployeeIds={excludeEmployeeIds}
      showAllOption={showAllOption}
      pillStyle={pillStyle}
    />
  );
}