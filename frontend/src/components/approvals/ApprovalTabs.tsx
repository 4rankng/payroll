import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { ApprovalStatus } from '@/types/approval';

interface ApprovalTabsProps {
  selectedTab: ApprovalStatus | 'all';
  onTabChange: (value: ApprovalStatus | 'all') => void;
  stats: {
    total: number;
    pending: number;
    approved: number;
    rejected: number;
    reviewing: number;
  };
}

export const ApprovalTabs = ({ selectedTab, onTabChange, stats }: ApprovalTabsProps) => {
  const tabs = [
    { value: 'all' as const, label: 'Tất cả', count: stats.total },
    { value: 'pending' as const, label: 'Chờ phê duyệt', count: stats.pending },
    { value: 'reviewing' as const, label: 'Đang xem xét', count: stats.reviewing },
    { value: 'approved' as const, label: 'Đã phê duyệt', count: stats.approved },
    { value: 'rejected' as const, label: 'Đã loại', count: stats.rejected },
  ];

  return (
    <Tabs value={selectedTab} onValueChange={onTabChange} className="mb-6">
      <TabsList className="grid w-full grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 h-auto p-1">
        {tabs.map((tab) => (
          <TabsTrigger
            key={tab.value}
            value={tab.value}
            className="flex items-center gap-2 px-3 py-2 typography-body-medium"
          >
            <span>{tab.label}</span>
            <Badge variant="secondary" className="typography-body-small">
              {tab.count}
            </Badge>
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
};
