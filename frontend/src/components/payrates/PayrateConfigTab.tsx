import { format } from "date-fns";
import { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Plus,
  Edit3,
  Calendar,
  ChevronDown,
  Trash2,
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
  useDeletePayRate,
} from '@/hooks/api/usePayRates';
import { Project } from '@/types/api/project.types';
import { authManager } from '@/lib/auth';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface PayrateConfigTabProps {
  project: Project;
}

export function PayrateConfigTab({ project }: PayrateConfigTabProps) {

  // API hooks
  const { data: allPayRatesData, isLoading: isLoadingAll } = useProjectPayRates(project.id);
  const deletePayRate = useDeletePayRate();

  const navigate = useNavigate();
  const payrateBasePath = authManager.getUserRole() === 'partner' ? '/partner' : '/admin';

  // Two-step inline delete confirm + surfaced failure (server still hard-blocks
  // configs with approved/paid timesheets).
  const [confirmDeleteId, setConfirmDeleteId] = useState<number | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [showOtherConfigs, setShowOtherConfigs] = useState(false);

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

  // Every other config (superseded/ended). From-date-only model: these rows
  // stay visible and deletable — invisible fragments are how the timeline
  // used to rot.
  const otherPayrates = useMemo(() => {
    const payrates = allPayRatesData?.data || [];
    return payrates.filter(p => p.id !== activePayrate?.id && p.id !== upcomingPayrate?.id);
  }, [allPayRatesData, activePayrate, upcomingPayrate]);

  const handleDeletePayrate = useCallback(async (payrate: PayRate) => {
    if (confirmDeleteId !== payrate.id) {
      setConfirmDeleteId(payrate.id);
      setDeleteError(null);
      return;
    }
    setConfirmDeleteId(null);
    try {
      await deletePayRate.mutateAsync({ payRateId: payrate.id, projectId: project.id });
    } catch (err: unknown) {
      const apiErr = err as { response?: { data?: { message?: string } }; message?: string };
      setDeleteError(apiErr?.response?.data?.message || apiErr?.message || 'Không thể xóa cấu hình mức lương.');
    }
  }, [confirmDeleteId, deletePayRate, project.id]);

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
        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-widest">
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
            <Badge variant={getPayrateStatus(activePayrate) === 'active' ? 'default' : 'secondary'} className="text-xs px-1.5 py-0">
              {STATUS_LABELS[getPayrateStatus(activePayrate)]}
            </Badge>
            <span className="text-xs text-muted-foreground">
              từ {format(new Date(activePayrate.fromDate), 'dd/MM/yyyy')}
              {activePayrate.toDate && ` → ${format(new Date(activePayrate.toDate), 'dd/MM/yyyy')}`}
            </span>
          </div>
          {/* Rate grid */}
          <div className="p-3">
            <PayrateRateGrid rates={activePayrate.rates} isFlexible={project.is_flexible} />
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
                className="h-6 px-2 text-xs text-muted-foreground hover:text-foreground gap-1"
                onClick={handleStartEditUpcoming}
              >
                <Edit3 className="h-2.5 w-2.5" />
                Sửa
              </Button>
            )}
          </div>
          <div className="p-3">
            <PayrateRateGrid rates={upcomingPayrate.rates} isFlexible={project.is_flexible} />
          </div>
        </div>
      )}

      {/* Other configs (ended/superseded) — visible and deletable */}
      {(otherPayrates.length > 0 || deleteError) && (
        <OtherConfigsSection
          payrates={otherPayrates}
          expanded={showOtherConfigs}
          onToggle={() => setShowOtherConfigs(v => !v)}
          canModify={canModifyPayrates}
          confirmDeleteId={confirmDeleteId}
          deletePending={deletePayRate.isPending}
          deleteError={deleteError}
          onDelete={handleDeletePayrate}
        />
      )}
    </div>
  );
}

function OtherConfigsSection({ payrates, expanded, onToggle, canModify, confirmDeleteId, deletePending, deleteError, onDelete }: {
  payrates: PayRate[];
  expanded: boolean;
  onToggle: () => void;
  canModify: boolean;
  confirmDeleteId: number | null;
  deletePending: boolean;
  deleteError: string | null;
  onDelete: (payrate: PayRate) => void;
}) {
  return (
    <div className="rounded-xl border border-border bg-card">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center justify-between px-3 py-2 text-left"
        aria-expanded={expanded}
      >
        <span className="text-xs font-medium text-muted-foreground">
          Cấu hình khác ({payrates.length})
        </span>
        <ChevronDown className={cn('h-3.5 w-3.5 text-muted-foreground transition-transform', expanded && 'rotate-180')} />
      </button>
      {expanded && (
        <div className="border-t border-border/50 px-3 py-2 space-y-2">
          {deleteError && (
            <p className="rounded-lg border border-utility-error-200 bg-utility-error-50 px-2.5 py-1.5 text-xs text-fg-error-primary">
              {deleteError}
            </p>
          )}
          {payrates.map(p => (
            <div key={p.id} className="flex items-center justify-between gap-2">
              <div className="min-w-0">
                <div className="flex items-center gap-1.5">
                  <Badge variant="secondary" className="text-xs px-1.5 py-0">Đã kết thúc</Badge>
                  <span className="text-xs text-muted-foreground truncate">
                    {format(new Date(p.fromDate), 'dd/MM/yyyy')}
                    {p.toDate && ` → ${format(new Date(p.toDate), 'dd/MM/yyyy')}`}
                  </span>
                </div>
              </div>
              {canModify && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 shrink-0 px-2 text-xs text-muted-foreground hover:text-fg-error-primary gap-1 min-h-7"
                  disabled={deletePending}
                  onClick={() => onDelete(p)}
                >
                  <Trash2 className="h-3 w-3" />
                  {confirmDeleteId === p.id ? 'Xác nhận xóa?' : 'Xóa'}
                </Button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
