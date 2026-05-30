import { SearchBar } from '@/components/shared/SearchBar';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ApprovalPriority, ApprovalType } from '@/types/approval';

interface ApprovalFiltersProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  priorityFilter: ApprovalPriority | 'all';
  onPriorityChange: (value: ApprovalPriority | 'all') => void;
  typeFilter: ApprovalType | 'all';
  onTypeChange: (value: ApprovalType | 'all') => void;
}

export const ApprovalFilters = ({
  searchTerm,
  onSearchChange,
  priorityFilter,
  onPriorityChange,
  typeFilter,
  onTypeChange,
}: ApprovalFiltersProps) => {
  return (
    <div className="flex items-center gap-1.5 flex-wrap mb-4">
      <SearchBar
        searchTerm={searchTerm}
        onSearchChange={onSearchChange}
        placeholder="Tìm kiếm yêu cầu..."
        className="w-56"
      />
      <Select value={priorityFilter} onValueChange={onPriorityChange}>
        <SelectTrigger className="h-8 w-[140px] text-xs">
          <SelectValue placeholder="Độ ưu tiên" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Tất cả độ ưu tiên</SelectItem>
          <SelectItem value="urgent">Khẩn cấp</SelectItem>
          <SelectItem value="high">Cao</SelectItem>
          <SelectItem value="medium">Trung bình</SelectItem>
          <SelectItem value="low">Thấp</SelectItem>
        </SelectContent>
      </Select>
      <Select value={typeFilter} onValueChange={onTypeChange}>
        <SelectTrigger className="h-8 w-[140px] text-xs">
          <SelectValue placeholder="Loại yêu cầu" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Tất cả loại</SelectItem>
          <SelectItem value="timesheet">Bảng công</SelectItem>
          <SelectItem value="payrate">Mức lương</SelectItem>
          <SelectItem value="payroll">Bảng lương</SelectItem>
          <SelectItem value="employee">Nhân viên</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
};
