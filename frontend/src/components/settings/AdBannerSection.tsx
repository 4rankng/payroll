import { useMemo, useState } from 'react';
import { format } from 'date-fns';
import { differenceInCalendarDays } from 'date-fns';
import {
  ArrowLeft,
  Loader2,
  Megaphone,
  Pause,
  Play,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import {
  useAdBanners,
  useDeleteAdBanner,
  useUpdateAdBanner,
} from '@/hooks/api/useAdBanners';
import { useAllProjects } from '@/hooks/api/useProjects';
import { cn } from '@/lib/utils';
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

const STATUS_DOT: Record<Status, string> = {
  running: 'bg-emerald-500',
  scheduled: 'bg-blue-500',
  expired: 'bg-muted-foreground/40',
  paused: 'bg-amber-500',
};

const STATUS_TEXT: Record<Status, string> = {
  running: 'text-emerald-700',
  scheduled: 'text-blue-700',
  expired: 'text-muted-foreground',
  paused: 'text-amber-700',
};

/** Tiny uppercase caption used for section eyebrows and metric labels. */
const microLabel = 'text-xs font-medium uppercase tracking-[0.08em] text-muted-foreground';

/** One cell of the metric rail. Cells share a single bordered strip. */
const Metric = ({ label, value }: { label: string; value: number }) => (
  <div className="px-4 py-3">
    <p className={microLabel}>{label}</p>
    <p className="mt-1 text-xl font-semibold tabular-nums leading-none tracking-tight text-foreground">
      {value}
    </p>
  </div>
);

/**
 * "Quảng cáo" Settings tab: campaign list with time-aware status and per-CTA
 * click counts, plus the composer with its live 375px preview.
 *
 * Layout is deliberately flat — the tab body sits directly on the page canvas
 * and uses hairlines to separate zones, so nothing renders as a card inside a
 * card.
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
    projects.forEach((p) => map.set(p.id, p.name));
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

  const tally = useMemo(() => {
    const counts = { running: 0, scheduled: 0, paused: 0, expired: 0 };
    banners.forEach((b) => { counts[statusOf(b, now).key] += 1; });
    return counts;
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
      <div className="space-y-6">
        <div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="-ml-2 h-8 gap-1.5 px-2 text-muted-foreground hover:text-foreground"
            onClick={() => setEditing(null)}
          >
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
            Chiến dịch quảng cáo
          </Button>

          <h2 className="mt-2 text-lg font-semibold tracking-tight text-foreground">
            {editing.clone
              ? 'Gia hạn chiến dịch'
              : editing.banner
                ? 'Chỉnh sửa chiến dịch'
                : 'Tạo chiến dịch quảng cáo'}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {editing.clone
              ? 'Nội dung được sao chép; thời gian bắt đầu từ hôm nay. Chiến dịch cũ giữ nguyên số liệu.'
              : 'Quảng cáo hiển thị trên trang chủ của nhân viên thuộc các dự án được chọn.'}
          </p>
        </div>

        {/* One solid surface; the composer's own field groups are borderless
            so nothing renders as a card inside a card. */}
        <div className="rounded-xl border bg-card p-4 shadow-sm sm:p-6">
          <AdBannerComposer
            key={editing.banner ? `${editing.banner.id}-${editing.clone}` : 'new'}
            initial={editing.banner}
            forceFreshWindow={editing.clone}
            onDone={() => setEditing(null)}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-lg font-semibold tracking-tight text-foreground">
            Chiến dịch quảng cáo
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Mỗi nhân viên chỉ thấy một quảng cáo tại một thời điểm.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="icon"
            aria-label="Tải lại danh sách"
            onClick={() => { void refetch(); }}
            disabled={isFetching}
          >
            <RefreshCw className={cn('h-4 w-4', isFetching && 'animate-spin')} />
          </Button>
          <Button type="button" className="gap-2" onClick={() => setEditing({ banner: null, clone: false })}>
            <Plus className="h-4 w-4" /> Tạo chiến dịch
          </Button>
        </div>
      </div>

      {!isLoading && !isError && banners.length > 0 && (
        <div className="grid grid-cols-2 divide-x divide-y overflow-hidden rounded-xl border bg-card shadow-sm sm:grid-cols-4 sm:divide-y-0">
          <Metric label="Đang chạy" value={tally.running} />
          <Metric label="Hẹn lịch" value={tally.scheduled} />
          <Metric label="Tạm dừng" value={tally.paused} />
          <Metric label="Hết hạn" value={tally.expired} />
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-16 text-muted-foreground">
          <Loader2 className="h-5 w-5 animate-spin" />
        </div>
      ) : isError ? (
        <p role="alert" className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
          {getErrorMessage(error)}
        </p>
      ) : ordered.length === 0 ? (
        <div className="rounded-xl border bg-card px-6 py-14 text-center shadow-sm">
          <Megaphone className="mx-auto h-6 w-6 text-muted-foreground" aria-hidden="true" />
          <p className="mt-3 text-sm font-medium text-foreground">Chưa có chiến dịch nào</p>
          <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
            Tạo chiến dịch đầu tiên để bắt đầu hiển thị quảng cáo cho nhân viên.
          </p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-4 gap-1.5"
            onClick={() => setEditing({ banner: null, clone: false })}
          >
            <Plus className="h-3.5 w-3.5" /> Tạo chiến dịch
          </Button>
        </div>
      ) : (
        /* One bordered surface, hairline-divided rows — a console list, not a
           stack of cards. */
        <ul className="divide-y overflow-hidden rounded-xl border bg-card shadow-sm">
          {ordered.map((banner) => {
            const status = statusOf(banner, now);
            const expired = status.key === 'expired';
            return (
              <li
                key={banner.id}
                className="group flex flex-wrap items-start justify-between gap-x-4 gap-y-3 px-4 py-3.5 transition-colors hover:bg-muted/40"
              >
                <div className={cn('min-w-0 flex-1', expired && 'opacity-60')}>
                  <div className="flex items-center gap-2">
                    <span
                      aria-hidden="true"
                      className={cn('h-1.5 w-1.5 shrink-0 rounded-full', STATUS_DOT[status.key])}
                    />
                    <h3 className="min-w-0 flex-1 break-words text-sm font-medium text-foreground sm:truncate">
                      {banner.title}
                    </h3>
                    <span className={cn('shrink-0 text-xs font-medium', STATUS_TEXT[status.key])}>
                      {status.label}
                    </span>
                  </div>

                  <p className="mt-1.5 break-words pl-3.5 text-xs text-muted-foreground sm:truncate">
                    <span className="tabular-nums">
                      {format(new Date(banner.startsAt), DATE_DISPLAY)} –{' '}
                      {format(new Date(banner.endsAt), DATE_DISPLAY)}
                    </span>
                    <span className="mx-1.5 text-border">·</span>
                    {targetingLabel(banner)}
                    <span className="mx-1.5 text-border">·</span>
                    Ưu tiên <span className="tabular-nums">{banner.priority}</span>
                  </p>

                  {banner.ctas.length > 0 && (
                    <p className="mt-1 pl-3.5 text-xs text-muted-foreground">
                      {banner.ctas.map((cta, index) => (
                        <span key={`${cta.label}-${index}`}>
                          {index > 0 && <span className="mx-1.5 text-border">·</span>}
                          {cta.label}{' '}
                          <span className="font-semibold tabular-nums text-foreground">
                            {banner.clickCounts?.[String(index)] ?? 0}
                          </span>
                        </span>
                      ))}
                      <span className="ml-1.5">lượt bấm</span>
                    </p>
                  )}
                </div>

                <div className="flex w-full basis-full shrink-0 flex-wrap items-center justify-end gap-1 sm:w-auto sm:basis-auto sm:flex-nowrap">
                  {expired ? (
                    <Button type="button" variant="outline" size="sm" className="min-h-11 sm:min-h-0" onClick={() => setEditing({ banner, clone: true })}>
                      Gia hạn
                    </Button>
                  ) : (
                    <>
                      <Button type="button" variant="outline" size="sm" className="min-h-11 sm:min-h-0" onClick={() => setEditing({ banner, clone: false })}>
                        Sửa
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="min-h-11 gap-1.5 text-muted-foreground hover:text-foreground sm:min-h-0"
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
                    className="min-h-11 min-w-11 text-muted-foreground hover:text-destructive sm:min-h-0 sm:min-w-0"
                    onClick={() => remove(banner)}
                    disabled={deleteMutation.isPending}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};
