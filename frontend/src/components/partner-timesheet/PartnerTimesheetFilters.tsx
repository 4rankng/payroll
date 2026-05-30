import { useMemo } from 'react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

interface PartnerTimesheetFiltersProps {
  selectedMonth: string;
  onMonthChange: (month: string) => void;
  onUpload: (file: File, projectId: number) => void;
}

export const PartnerTimesheetFilters = ({
  selectedMonth,
  onMonthChange,
}: PartnerTimesheetFiltersProps) => {
  const months = useMemo(() => {
    const now = new Date();
    const result = [];
    for (let i = 0; i < 12; i++) {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
      result.push({
        value: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`,
        label: `T${d.getMonth() + 1}/${d.getFullYear()}`,
      });
    }
    return result;
  }, []);

  return (
    <Select value={selectedMonth} onValueChange={onMonthChange}>
      <SelectTrigger className="h-8 w-[120px] text-xs">
        <SelectValue placeholder="Chọn tháng" />
      </SelectTrigger>
      <SelectContent>
        {months.map((m) => (
          <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
