import { memo, useEffect, useMemo, useState } from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';

import { BankDistributionChart } from '@/components/admin-dashboard/BankDistributionChart';
import { Button } from '@/components/ui/button';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { Skeleton } from '@/components/ui/skeleton';
import { ConsolePanel } from '@/components/shared/console-panel';
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
      triggerClassName="w-[min(44vw,12rem)]"
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

interface BankTransferBreakdownCardProps {
  className?: string;
  /** Controlled project filter — when omitted the card manages its own
   *  selection via the internal ProjectSelector (desktop dashboard). */
  selectedProjectId?: number | null;
  onProjectChange?: (id: number | null) => void;
}

export const BankTransferBreakdownCard = memo(function BankTransferBreakdownCard({
  className,
  selectedProjectId: controlledProjectId,
  onProjectChange,
}: BankTransferBreakdownCardProps) {
  const [internalProjectId, setInternalProjectId] = useState<number | null>(null);
  const [slowLoad, setSlowLoad] = useState(false);
  const isControlled = controlledProjectId !== undefined;
  const selectedProjectId = isControlled ? controlledProjectId : internalProjectId;
  const setSelectedProjectId = (id: number | null) => {
    if (isControlled) {
      onProjectChange?.(id);
    } else {
      setInternalProjectId(id);
    }
  };
  const {
    data: allData,
    isLoading,
    isError,
    isFetching,
    refetch,
  } = useBankUsageAllProjects();

  useEffect(() => {
    if (!isLoading) {
      setSlowLoad(false);
      return;
    }
    const timer = setTimeout(() => setSlowLoad(true), 8000);
    return () => clearTimeout(timer);
  }, [isLoading]);

  const projects = allData?.projects ?? [];

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
    <ConsolePanel
      kicker="Chi trả"
      title="Ngân hàng nhận lương"
      className={className}
      bodyClassName="flex flex-1 flex-col justify-center"
      actions={
        !isControlled && projects.length > 0 ? (
          <ProjectSelector
            projects={projects}
            selectedId={selectedProjectId}
            onSelect={setSelectedProjectId}
          />
        ) : undefined
      }
    >
      {isLoading && slowLoad ? (
        <div className="flex flex-col items-start justify-between gap-3 rounded-lg border border-border bg-muted/20 p-3 sm:flex-row sm:items-center">
          <p className="text-[13px] leading-5 text-muted-foreground">
            Phân bổ ngân hàng đang tải chậm hơn bình thường.
          </p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-9 shrink-0"
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            <RefreshCw className="mr-2 h-3.5 w-3.5" aria-hidden="true" />
            Tải lại
          </Button>
        </div>
      ) : isLoading ? (
        <div aria-label="Đang tải phân bổ ngân hàng">
          <div className="mx-auto h-28 w-28 animate-pulse rounded-full bg-muted motion-reduce:animate-none" />
          <div className="mt-3 grid grid-cols-3 gap-2">
            {Array.from({ length: 3 }).map((_, index) => (
              <Skeleton key={index} className="h-12 rounded-lg" />
            ))}
          </div>
        </div>
      ) : isError ? (
        <div
          className="flex flex-col items-start justify-between gap-3 rounded-lg border border-destructive/20 bg-destructive/5 p-3 sm:flex-row sm:items-center"
          role="alert"
        >
          <div className="flex items-start gap-2.5">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" aria-hidden="true" />
            <p className="text-[13px] leading-5 text-foreground">
              Không thể tải phân bổ ngân hàng. Kiểm tra kết nối rồi thử lại.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-9 shrink-0"
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            <RefreshCw
              className={`mr-2 h-3.5 w-3.5 ${isFetching ? 'animate-spin motion-reduce:animate-none' : ''}`}
              aria-hidden="true"
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
                showIllustration={false}
          title="Chưa có dữ liệu ngân hàng"
          description="Dữ liệu sẽ xuất hiện khi nhân viên có thông tin nhận lương."
          size="sm"
          className="py-4"
        />
      )}
    </ConsolePanel>
  );
});
