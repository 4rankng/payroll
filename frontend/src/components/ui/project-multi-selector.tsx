import { useMemo } from 'react';
import { MultiSearchableDropdown } from '@/components/ui/multi-searchable-dropdown';
import type { MultiSearchableDropdownOption } from '@/components/ui/multi-searchable-dropdown';

interface ProjectMultiSelectorProps {
  value: number[];
  onChange: (projectIds: number[]) => void;
  projects: Array<{ id: number; code: string; name: string }>;
  placeholder?: string;
  disabled?: boolean;
}

export function ProjectMultiSelector({
  value,
  onChange,
  projects,
  placeholder = 'Tất cả dự án',
  disabled = false,
}: ProjectMultiSelectorProps) {
  const options = useMemo<MultiSearchableDropdownOption[]>(
    () => projects.map((p) => ({ value: String(p.id), label: `${p.code} - ${p.name}`, searchText: `${p.code} ${p.name}` })),
    [projects],
  );

  return (
    <MultiSearchableDropdown
      value={value.map(String)}
      onChange={(ids) => onChange(ids.map(Number))}
      options={options}
      placeholder={placeholder}
      searchPlaceholder="Tìm kiếm dự án..."
      emptyMessage="Không tìm thấy dự án nào."
      allOption={{ value: 'all', label: placeholder }}
      disabled={disabled}
    />
  );
}
