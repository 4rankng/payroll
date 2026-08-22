import { Card, CardContent } from '@/components/ui/card';
import { EmptyState } from '@/components/shared/EmptyState';
import { ApprovalItem } from '@/types/approval';
import { ApprovalItemCard } from './ApprovalItemCard';
import { SelectAllCheckbox } from './SelectAllCheckbox';

interface ApprovalItemsListProps {
  items: ApprovalItem[];
  selectedItems: string[];
  onToggleItem: (id: string) => void;
  onSelectAll: () => void;
  onViewDetails: (item: ApprovalItem) => void;
  isAllSelected: boolean;
  hasFilters: boolean;
}

export const ApprovalItemsList = ({
  items,
  selectedItems,
  onToggleItem,
  onSelectAll,
  onViewDetails,
  isAllSelected,
  hasFilters
}: ApprovalItemsListProps) => {
  if (items.length === 0) {
    return (
      <Card>
        <CardContent className="px-4">
          <EmptyState
            title="Không có yêu cầu nào"
            description={hasFilters
              ? 'Không tìm thấy yêu cầu phù hợp với bộ lọc'
              : 'Hiện tại không có yêu cầu phê duyệt nào'}
            size="sm"
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <SelectAllCheckbox
        isAllSelected={isAllSelected}
        onSelectAll={onSelectAll}
        totalItems={items.length}
      />

      {items.map((item) => (
        <ApprovalItemCard
          key={item.id}
          item={item}
          isSelected={selectedItems.includes(item.id)}
          onToggleSelection={() => onToggleItem(item.id)}
          onViewDetails={() => onViewDetails(item)}
        />
      ))}
    </div>
  );
};
