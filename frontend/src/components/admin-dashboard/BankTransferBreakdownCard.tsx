import { memo, useMemo } from 'react';
import { AlertCircle, Building2, RefreshCw } from 'lucide-react';

import { BankDistributionChart } from '@/components/admin-dashboard/BankDistributionChart';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { Skeleton } from '@/components/ui/skeleton';
import { EmptyState } from '@/components/shared/EmptyState';
import { useBankUsageAllProjects } from '@/hooks/api/useDashboard';
import type { ProjectBankUsageItem } from '@/types/api/dashboard.types';

import { buildBankDistribution } from './bank-distribution';

interface ProjectSelectorProps {
  projects: ProjectBankUsageItem[];
  selectedId: number | null;
  onSelect: (id: number | null) => void;
}

export const ProjectSelector = memo(function ProjectSelector({
  projects,
  selectedId,
  onSelect,
}: ProjectSelectorProps) {
  return (
    <SearchableSelect
      value={selectedId === null ? 'all' : String(selectedId)}
      onChange={(value) => onSelect(value === 'all' ? null : Number(value))}
      placeholder="Tất cả dự án"
      searchPlaceholder="Tìm dự án..."
      triggerAriaLabel="Lọc phân bổ ngân hàng theo dự án"
      contentAlign="end"
      triggerClassName="w-[min(58vw,14rem)]"
      options={[
        { value: 'all', label: 'Tất cả dự án' },
        ...projects.map((project) => ({
          value: String(project.project_id),
          label: project.project_name,
          searchText: `${project.total_employees.toLocaleString('vi-VN')} nhân viên`,
        })),
      ]}
    />
  );
});

export const BankTransferBreakdownCard = memo(function BankTransferBreakdownCard({
  selectedProjectId = null,
}: {
  selectedProjectId?: number | null;
}) {
  const {
    data: allData,
    isLoading,
    isError,
    isFetching,
    refetch,
  } = useBankUsageAllProjects();

  const displayData = useMemo(() => {
    if (selectedProjectId !== null && allData) {
      const project = allData.projects.find((item) => item.project_id === selectedProjectId);
      if (project) {
        return {
          items: buildBankDistribution(project.banks, project.total_employees),
          totalEmployees: project.total_employees,
        };
      }
    }

    const overall = allData?.overall;
    if (!overall) {
      return null;
    }

    return {
      items: buildBankDistribution(overall.banks, overall.total_employees),
      totalEmployees: overall.total_employees,
    };
  }, [selectedProjectId, allData]);

  return (
    <Card className="overflow-hidden shadow-none">
      <CardContent className="p-3 sm:p-4">
        {isLoading ? (
          <div aria-label="Đang tải phân bổ ngân hàng">
            <div className="flex items-end justify-between gap-4">
              <div className="space-y-2">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-8 w-36" />
              </div>
              <Skeleton className="h-14 w-48 rounded-lg" />
            </div>
            <Skeleton className="mt-4 h-3 w-full rounded-full" />
            <div className="mt-4 grid gap-2 sm:grid-cols-2">
              {Array.from({ length: 6 }).map((_, index) => (
                <Skeleton key={index} className="h-[74px] rounded-xl" />
              ))}
            </div>
          </div>
        ) : isError ? (
          <div
            className="flex flex-col items-start justify-between gap-4 rounded-xl border border-destructive/20 bg-destructive/5 p-4 sm:flex-row sm:items-center"
            role="alert"
          >
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-destructive" />
              <div>
                <p className="text-sm font-medium text-foreground">
                  Không thể tải phân bổ ngân hàng
                </p>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Kiểm tra kết nối rồi thử tải lại dữ liệu.
                </p>
              </div>
            </div>
            <Button
              type="button"
              variant="outline"
              className="h-11 shrink-0 sm:h-9"
              disabled={isFetching}
              onClick={() => void refetch()}
            >
              <RefreshCw
                className={`mr-2 h-4 w-4 ${isFetching ? 'animate-spin motion-reduce:animate-none' : ''}`}
              />
              Thử lại
            </Button>
          </div>
        ) : displayData && displayData.items.length > 0 ? (
          <BankDistributionChart
            items={displayData.items}
            totalEmployees={displayData.totalEmployees}
          />
        ) : (
          <EmptyState
            title="Chưa có dữ liệu ngân hàng"
            description="Dữ liệu sẽ xuất hiện khi nhân viên có thông tin nhận lương."
            size="sm"
            className="py-4"
          />
        )}
      </CardContent>
    </Card>
  );
});
