import { Checkbox } from '@/components/ui/checkbox';

interface SelectAllCheckboxProps {
  isAllSelected: boolean;
  onSelectAll: () => void;
  totalItems: number;
}

export const SelectAllCheckbox = ({ isAllSelected, onSelectAll, totalItems }: SelectAllCheckboxProps) => {
  return (
    <div className="flex items-center gap-2 py-2 px-4 bg-muted/50 rounded-xl">
      <Checkbox
        checked={isAllSelected}
        onCheckedChange={onSelectAll}
      />
      <span className="typography-body-medium">
        Chọn tất cả ({totalItems} yêu cầu)
      </span>
    </div>
  );
};