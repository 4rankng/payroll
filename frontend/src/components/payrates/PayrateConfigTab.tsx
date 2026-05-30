import { format } from "date-fns";
import { useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Plus,
  Edit3,
  Calendar,
} from 'lucide-react';
import { PayrateRateGrid } from './components/PayrateRateGrid';
import {
  STATUS_LABELS
} from './types';
import { dateToString } from '@/utils/dateHelpers';
import { getPayrateStatus } from '@/types/api/payrate.types';
import { PayRate } from '@/types/api/payrate.types';
import {
  useProjectPayRates,
} from '@/hooks/api/usePayRates';
import { Project } from '@/types/api/project.types';
import { authManager } from '@/lib/auth';
import { Badge } from '@/components/ui/badge';

interface PayrateConfigTabProps {
  project: Project;
}

export function PayrateConfigTab({ project }: PayrateConfigTabProps) {

  // API hooks
  const { data: allPayRatesData, isLoading: isLoadingAll } = useProjectPayRates(project.id);

  const navigate = useNavigate();
  const payrateBasePath = authManager.getUserRole() === 'partner' ? '/partner' : '/admin';

  // Determine active and upcoming payrates
  const { activePayrate, upcomingPayrate } = useMemo(() => {
    const today = dateToString(new Date());
    const payrates = allPayRatesData?.data || [];

    let active: PayRate | null = null;
    let upcoming: PayRate | null = null;

    for (const payrate of payrates) {
      const fromDate = payrate.fromDate;
      const toDate = payrate.toDate;

      // Current active: fromDate <= today AND (toDate is null OR toDate >= today)
      if (fromDate <= today && (!toDate || toDate >= today)) {
        active = payrate;
      }

      // Upcoming: fromDate > today
      if (fromDate > today) {
        if (!upcoming || fromDate < upcoming.fromDate) {
          upcoming = payrate;
        }
      }
    }

    return { activePayrate: active, upcomingPayrate: upcoming };
  }, [allPayRatesData]);

  // View state (overview only now — editing navigates to dedicated page)
  const canModifyPayrates = project.status !== 'cancelled' && project.status !== 'completed';

  const handleStartCreateNew = useCallback(() => {
    navigate(`${payrateBasePath}/projects/${project.id}/payrates/new/edit`);
  }, [navigate, payrateBasePath, project.id]);

  const handleStartEditCurrent = useCallback(() => {
    if (activePayrate) {
      navigate(`${payrateBasePath}/projects/${project.id}/payrates/${activePayrate.id}/edit`);
    }
  }, [navigate, payrateBasePath, project.id, activePayrate]);

  const handleStartEditUpcoming = useCallback(() => {
    if (upcomingPayrate) {
      navigate(`${payrateBasePath}/projects/${project.id}/payrates/${upcomingPayrate.id}/edit`);
    }
  }, [navigate, payrateBasePath, project.id, upcomingPayrate]);

  if (isLoadingAll) {
    return (
      <div className="space-y-6 p-1">
        <Skeleton className="h-48" />
        <Skeleton className="h-64" />
        <Skeleton className="h-32" />
      </div>
    );
  }

  return (
    <div className="space-y-3 w-full">
      {/* Section header: label + actions */}
      <div className="flex items-center justify-between">
        <p className="text-[10px] font-semibold text-muted-foreground uppercase tracking-widest">
          Cấu hình lương
        </p>
        {canModifyPayrates && (
          <div className="flex items-center gap-1">
            {activePayrate && (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
                onClick={handleStartEditCurrent}
              >
                <Edit3 className="h-3 w-3" />
                Sửa
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
              onClick={handleStartCreateNew}
            >
              <Plus className="h-3 w-3" />
              Tạo mới
            </Button>
          </div>
        )}
      </div>

      {/* Active payrate */}
      {activePayrate ? (
        <div className="rounded-xl border border-border bg-card">
          {/* Meta row */}
          <div className="flex items-center gap-2 px-3 py-2 border-b border-border/50">
            <Badge variant={getPayrateStatus(activePayrate) === 'active' ? 'default' : 'secondary'} className="text-[10px] px-1.5 py-0">
              {STATUS_LABELS[getPayrateStatus(activePayrate)]}
            </Badge>
            <span className="text-xs text-muted-foreground">
              từ {format(new Date(activePayrate.fromDate), 'dd/MM/yyyy')}
              {activePayrate.toDate && ` → ${format(new Date(activePayrate.toDate), 'dd/MM/yyyy')}`}
            </span>
          </div>
          {/* Rate grid */}
          <div className="p-3">
            <PayrateRateGrid rates={activePayrate.rates} />
          </div>
        </div>
      ) : (
        <div className="rounded-xl border border-dashed border-border px-4 py-6 text-center">
          <p className="text-xs text-muted-foreground">Chưa có cấu hình lương</p>
          {canModifyPayrates && (
            <Button variant="outline" size="sm" className="mt-3 text-xs h-7" onClick={handleStartCreateNew}>
              <Plus className="h-3 w-3 mr-1" />
              Tạo cấu hình đầu tiên
            </Button>
          )}
        </div>
      )}

      {/* Upcoming payrate */}
      {upcomingPayrate && (
        <div className="rounded-xl border border-border bg-card">
          <div className="flex items-center justify-between px-3 py-2 border-b border-border/50">
            <div className="flex items-center gap-2">
              <Calendar className="h-3 w-3 text-muted-foreground" />
              <span className="text-xs font-medium text-muted-foreground">Sắp tới</span>
              <span className="text-xs text-muted-foreground">
                từ {format(new Date(upcomingPayrate.fromDate), 'dd/MM/yyyy')}
                {upcomingPayrate.toDate && ` → ${format(new Date(upcomingPayrate.toDate), 'dd/MM/yyyy')}`}
              </span>
            </div>
            {canModifyPayrates && (
              <Button
                variant="ghost"
                size="sm"
                className="h-6 px-2 text-[10px] text-muted-foreground hover:text-foreground gap-1"
                onClick={handleStartEditUpcoming}
              >
                <Edit3 className="h-2.5 w-2.5" />
                Sửa
              </Button>
            )}
          </div>
          <div className="p-3">
            <PayrateRateGrid rates={upcomingPayrate.rates} />
          </div>
        </div>
      )}
    </div>
  );
}
