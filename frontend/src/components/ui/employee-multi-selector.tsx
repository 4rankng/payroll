import { useMemo } from 'react';
import { MultiSearchableDropdown } from '@/components/ui/multi-searchable-dropdown';
import type { MultiSearchableDropdownOption } from '@/components/ui/multi-searchable-dropdown';

interface EmployeeMultiSelectorProps {
  value: number[];
  onChange: (employeeIds: number[]) => void;
  employees: Array<{
    id: number;
    employee_code: string;
    fullname: string;
    cccd?: string | null;
    date_of_birth?: string | null;
  }>;
  placeholder?: string;
  disabled?: boolean;
}

export function EmployeeMultiSelector({
  value,
  onChange,
  employees,
  placeholder = 'Tất cả nhân viên',
  disabled = false,
}: EmployeeMultiSelectorProps) {
  const options = useMemo<MultiSearchableDropdownOption[]>(
    () =>
      employees.map((e) => ({
        value: String(e.id),
        label: `${e.employee_code} - ${e.fullname}`,
        searchText: `${e.employee_code} ${e.fullname} ${e.cccd || ''} ${e.date_of_birth || ''}`,
      })),
    [employees],
  );

  return (
    <MultiSearchableDropdown
      value={value.map(String)}
      onChange={(ids) => onChange(ids.map(Number))}
      options={options}
      placeholder={placeholder}
      searchPlaceholder="Tìm kiếm nhân viên..."
      emptyMessage="Không tìm thấy nhân viên nào."
      allOption={{ value: 'all', label: placeholder }}
      disabled={disabled}
    />
  );
}
