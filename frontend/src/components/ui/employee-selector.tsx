import { useState, useMemo, useCallback } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { AsyncSearchableDropdown } from '@/components/ui/async-searchable-dropdown';
import { useDebounce } from '@/hooks/useDebounce';
import { employeeService } from '@/services/api/employee.service';
import { cn } from '@/lib/utils';
import { Users } from 'lucide-react';
import type { Employee } from '@/types/api/employee.types';

interface EmployeeSelectorProps {
  value?: Employee | null;
  onSelect: (employee: Employee | null) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  availableEmployees?: Employee[];
  includeAvailableOnly?: boolean;
  excludeEmployeeIds?: number[];
  showAllOption?: boolean; // New prop to control showing "All Employees" option
}

export function EmployeeSelector({
  value,
  onSelect,
  placeholder = "Chọn nhân viên...",
  disabled = false,
  className,
  availableEmployees = [],
  includeAvailableOnly = false,
  excludeEmployeeIds = [],
  showAllOption = true // Default to true for timesheet pages
}: EmployeeSelectorProps) {
  const [searchValue, setSearchValue] = useState('');
  const debouncedSearchValue = useDebounce(searchValue, 300);
  const isSearching = searchValue !== debouncedSearchValue;

  const {
    data: employeesData,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage
  } = useInfiniteQuery({
    queryKey: ['employees', 'selector', { search: debouncedSearchValue }],
    queryFn: async ({ pageParam = 1 }) => {
      if (includeAvailableOnly) {
        return { data: availableEmployees, pagination: null };
      }

      const filters = {
        page: pageParam,
        pageSize: 20,
        status: 'working',
        sortBy: 'fullname',
        sortOrder: 'asc' as const,
        ...(debouncedSearchValue.length >= 3 && { search: debouncedSearchValue })
      };

      return await employeeService.getEmployees(filters);
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      if (!lastPage.pagination) return undefined;
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
    enabled: !includeAvailableOnly &&
              (debouncedSearchValue.length === 0 || debouncedSearchValue.length >= 3)
  });

  const employees = useMemo(() => {
    let employeeList: Employee[] = [];

    if (includeAvailableOnly) {
      employeeList = availableEmployees.filter(employee => !excludeEmployeeIds.includes(employee.id));
    } else {
      employeeList = employeesData?.pages.flatMap(page => page.data || []) || [];
      employeeList = employeeList.filter(employee => !excludeEmployeeIds.includes(employee.id));
    }

    // Add "All Employees" option if enabled
    if (showAllOption && !isSearching && debouncedSearchValue.length < 3) {
      const allEmployeesOption: Employee = {
        id: -1, // Special ID for "All Employees"
        fullname: 'Tất cả nhân viên',
        cccd: '',
        email: '',
        username: '',
        status: 'working',
        created_at: '',
        updated_at: ''
      };
      return [allEmployeesOption, ...employeeList];
    }

    return employeeList;
  }, [employeesData, includeAvailableOnly, availableEmployees, excludeEmployeeIds, showAllOption, isSearching, debouncedSearchValue]);

  const handleSelect = useCallback((employee: Employee) => {
    // If "All Employees" is selected, pass null
    if (employee.id === -1) {
      onSelect(null);
    } else {
      onSelect(employee);
    }
  }, [onSelect]);

  const renderTrigger = useCallback((selected: Employee | null, placeholderText: string) => {
    if (selected) {
      return (
        <div className="flex flex-col items-start gap-0.5 flex-1 min-w-0">
          <span className="font-medium whitespace-normal break-words text-left">{selected.fullname}</span>
          <span className="text-sm text-muted-foreground whitespace-normal break-words text-left">
            CCCD: {selected.cccd}
          </span>
        </div>
      );
    }

    return (
      <span className="truncate text-left">{placeholderText}</span>
    );
  }, []);

  const renderOption = useCallback((employee: Employee) => (
    <div className="flex flex-col">
      {employee.id === -1 ? (
        <>
          <div className="flex items-center gap-2">
            <Users className="w-4 h-4" />
            <span className="font-medium">{employee.fullname}</span>
          </div>
        </>
      ) : (
        <>
          <span className="font-medium">{employee.fullname}</span>
          <span className="text-sm text-muted-foreground">
            CCCD: {employee.cccd}
          </span>
        </>
      )}
    </div>
  ), []);

  const filterValue = useCallback((employee: Employee) => `${employee.fullname} ${employee.cccd}`, []);

  const errorMessage = error ? 'Có lỗi xảy ra khi tìm kiếm nhân viên' : undefined;

  return (
    <AsyncSearchableDropdown
      options={employees}
      value={value ?? null}
      onSelect={handleSelect}
      placeholder={placeholder}
      searchValue={searchValue}
      onSearchValueChange={setSearchValue}
      isLoading={isLoading}
      isSearching={isSearching}
      hasNextPage={hasNextPage}
      isFetchingNextPage={isFetchingNextPage}
      loadMore={fetchNextPage}
      errorMessage={errorMessage}
      searchPlaceholder="Tên hoặc CCCD"
      loadingMessage="Đang tìm kiếm..."
      loadingMoreMessage="Đang tải thêm..."
      emptyMessages={{
        searchTooShort: "Nhập tối thiểu 3 ký tự để tìm kiếm",
        noResults: "Không tìm thấy nhân viên nào",
        empty: "Không có nhân viên khả dụng"
      }}
      minSearchLength={3}
      renderTrigger={renderTrigger}
      renderOption={renderOption}
      getOptionKey={(employee) => employee.id}
      getOptionFilterValue={filterValue}
      disabled={disabled}
      triggerClassName={cn("min-w-0 justify-between whitespace-nowrap", className)}
      modal={false}
    />
  );
}
