import { useMemo, useState } from 'react';
import { format } from 'date-fns';
import { differenceInCalendarDays } from 'date-fns';
import { Loader2, Megaphone, Pause, Play, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  useAdBanners,
  useDeleteAdBanner,
  useUpdateAdBanner,
} from '@/hooks/api/useAdBanners';
import { useAllProjects } from '@/hooks/api/useProjects';
import { getErrorMessage } from '@/utils/error-handler';
import type { AdBanner } from '@/types/api/ad-banner.types';

import { AdBannerComposer } from './AdBannerComposer';

const DATE_DISPLAY = 'dd/MM/yyyy';

type Status = 'running' | 'scheduled' | 'expired' | 'paused';

const statusOf = (banner: AdBanner, now: Date): { key: Status; label: string } => {
  const start = new Date(banner.startsAt);
  const end = new Date(banner.endsAt);
  if (!banner.isActive) return { key: 'paused', label: 'Tạm dừng' };
  if (now >= end) return { key: 'expired', label: 'Hết hạn' };
  if (now < start) return { key: 'scheduled', label: 'Hẹn lịch' };
  const daysLeft = differenceInCalendarDays(end, now);
  return { key: 'running', label: daysLeft > 0 ? `Còn ${daysLeft} ngày` : 'Hết hôm nay' };
};

const STATUS_STYLES: Record<Status, string> = {
  running: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  scheduled: 'bg-blue-50 text-blue-700 border-blue-200',
  expired: 'bg-muted text-muted-foreground border-border',
  paused: 'bg-amber-50 text-amber-700 border-amber-200',
};

/**
 * "Quảng cáo" Settings tab: campaign list with time-aware status chips and
 * per-CTA click counts, plus the composer with its live 375px preview.
 */
export const AdBannerSection = () => {
  const { data: banners = [], isLoading, isError, error, refetch, isFetching } = useAdBanners();
  const { data: projects = [] } = useAllProjects();
  const updateMutation = useUpdateAdBanner();
  const deleteMutation = useDeleteAdBanner();

  // null = list mode; 'new' = composer fresh; otherwise the campaign being
  // edited ('clone' flag distinguishes Gia hạn).
  const [editing, setEditing] = useState<{ banner: AdBanner | null; clone: boolean } | null>(null);

  const projectNameById = useMemo(() => {
    const map = new Map<number, string>();
    projects.forEach((p) => map.set(p.id, `${p.code} — ${p.name}`));
    return map;
  }, [projects]);

  const now = new Date();

  const ordered = useMemo(() => {
    const rank = (b: AdBanner) => {
      const s = statusOf(b, now);
      if (s.key === 'running') return 0;
      if (s.key === 'scheduled') return 1;
      if (s.key === 'paused') return 2;
      return 3;
    };
    return [...banners].sort((a, b) => rank(a) - rank(b));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [banners]);

  const targetingLabel = (banner: AdBanner) =>
    banner.targetProjectIds.length === 0
      ? 'Tất cả dự án'
      : banner.targetProjectIds.map((id) => projectNameById.get(id) ?? `Dự án #${id}`).join(', ');

  const toggleActive = (banner: AdBanner) => {
    updateMutation.mutate(
      {
        id: banner.id,
        payload: {
          title: banner.title,
          body: banner.body,
          bullets: banner.bullets,
          ctas: banner.ctas,
          footer: banner.footer,
          targetProjectIds: banner.targetProjectIds,
          priority: banner.priority,
          startsAt: banner.startsAt,
          endsAt: banner.endsAt,
          isActive: !banner.isActive,
        },
      },
      {
        onSuccess: () => toast.success(banner.isActive ? 'Đã tạm dừng chiến dịch' : 'Đã kích hoạt chiến dịch'),
        onError: (err: unknown) => toast.error(getErrorMessage(err)),
      },
    );
  };

  const remove = (banner: AdBanner) => {
    if (!window.confirm(`Xóa chiến dịch "${banner.title}"? Số liệu lượt bấm sẽ không còn hiển thị.`)) return;
    deleteMutation.mutate(banner.id, {
      onSuccess: () => toast.success('Đã xóa chiến dịch'),
      onError: (err: unknown) => toast.error(getErrorMessage(err)),
    });
  };

  if (editing) {
    return (
      <div className="rounded-xl border bg-card p-4 sm:p-6">
        <h2 className="mb-1 text-sm font-semibold text-foreground">
          {editing.clone
            ? 'Gia hạn chiến dịch'
            : editing.banner
              ? 'Chỉnh sửa chiến dịch'
              : 'Tạo chiến dịch quảng cáo'}
        </h2>
        <p className="mb-5 text-sm text-muted-foreground">
          {editing.clone
            ? 'Nội dung được sao chép; thời gian bắt đầu từ hôm nay. Chiến dịch cũ giữ nguyên số liệu.'
            : 'Quảng cáo hiển thị trên trang chủ của nhân viên thuộc các dự án được chọn.'}
        </p>
        <AdBannerComposer
          key={editing.banner ? `${editing.banner.id}-${editing.clone}` : 'new'}
          initial={editing.banner}
          forceFreshWindow={editing.clone}
          onDone={() => setEditing(null)}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Megaphone className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <p className="text-sm text-muted-foreground">
            {banners.length} chiến dịch — mỗi nhân viên thấy một quảng cáo mỗi lúc.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Tải lại danh sách"
            onClick={() => { void refetch(); }}
            disabled={isFetching}
          >
            <RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
          </Button>
          <Button type="button" className="gap-2" onClick={() => setEditing({ banner: null, clone: false })}>
            <Plus className="h-4 w-4" /> Tạo chiến dịch
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" />
        </div>
      ) : isError ? (
        <p role="alert" className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
          {getErrorMessage(error)}
        </p>
      ) : ordered.length === 0 ? (
        <p className="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground">
          Chưa có chiến dịch nào. Tạo chiến dịch đầu tiên để bắt đầu hiển thị quảng cáo cho nhân viên.
        </p>
      ) : (
        <ul className="space-y-3">
          {ordered.map((banner) => {
            const status = statusOf(banner, now);
            const expired = status.key === 'expired';
            return (
              <li
                key={banner.id}
                className={`rounded-xl border bg-card p-4 ${expired ? 'opacity-60' : ''}`}
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="outline" className={STATUS_STYLES[status.key]}>
                        {status.label}
                      </Badge>
                      <h3 className="truncate text-sm font-semibold text-foreground">{banner.title}</h3>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {targetingLabel(banner)} · {format(new Date(banner.startsAt), DATE_DISPLAY)} —{' '}
                      {format(new Date(banner.endsAt), DATE_DISPLAY)} · Ưu tiên {banner.priority}
                    </p>
                  </div>
                  <div className="flex flex-wrap items-center gap-1.5">
                    {expired ? (
                      <Button type="button" variant="outline" size="sm" onClick={() => setEditing({ banner, clone: true })}>
                        Gia hạn
                      </Button>
                    ) : (
                      <>
                        <Button type="button" variant="outline" size="sm" onClick={() => setEditing({ banner, clone: false })}>
                          Sửa
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="gap-1.5"
                          onClick={() => toggleActive(banner)}
                          disabled={updateMutation.isPending}
                        >
                          {banner.isActive ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
                          {banner.isActive ? 'Tạm dừng' : 'Kích hoạt'}
                        </Button>
                      </>
                    )}
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      aria-label={`Xóa chiến dịch ${banner.title}`}
                      onClick={() => remove(banner)}
                      disabled={deleteMutation.isPending}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                {banner.ctas.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {banner.ctas.map((cta, index) => (
                      <span
                        key={`${cta.label}-${index}`}
                        className="inline-flex items-center gap-1.5 rounded-full bg-muted px-2.5 py-1 text-xs text-muted-foreground"
                      >
                        {cta.label}
                        <strong className="text-foreground">{banner.clickCounts?.[String(index)] ?? 0}</strong>
                      </span>
                    ))}
                  </div>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};
