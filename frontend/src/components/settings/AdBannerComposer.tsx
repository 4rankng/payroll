import { useMemo, useState } from 'react';
import { format, addDays } from 'date-fns';
import { Loader2, Plus, Smartphone, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { ProjectMultiSelector } from '@/components/ui/project-multi-selector';
import { EmployeeAdContent } from '@/components/employees/EmployeeAdSheet';
import {
  useCreateAdBanner,
  useUpdateAdBanner,
} from '@/hooks/api/useAdBanners';
import { useAllProjects } from '@/hooks/api/useProjects';
import { cn } from '@/lib/utils';
import { getErrorMessage } from '@/utils/error-handler';
import type { AdBanner, AdBannerCTA, AdBannerPayload } from '@/types/api/ad-banner.types';

const MAX_BULLETS = 6;
const MAX_CTAS = 3;
const MAX_LIFETIME_DAYS = 180;
const DATE_DISPLAY = 'dd/MM/yyyy';

type DurationPreset = '7' | '14' | '30' | '60' | 'custom';

const DURATION_PRESETS: Array<{ value: DurationPreset; label: string }> = [
  { value: '7', label: '7 ngày' },
  { value: '14', label: '14 ngày' },
  { value: '30', label: '30 ngày' },
  { value: '60', label: '60 ngày' },
  { value: 'custom', label: 'Tùy chọn' },
];

const toLocalDateInput = (iso: string) => iso.slice(0, 10);

const emptyCTA = (): AdBannerCTA => ({ label: '', type: 'url', value: '' });

interface AdBannerComposerProps {
  /** Campaign being edited; null = create fresh (or a Gia hạn clone). */
  initial: AdBanner | null;
  /** For Gia hạn: prefilled content but a fresh window starting today. */
  forceFreshWindow?: boolean;
  onDone: () => void;
}

const isValidPhone = (value: string) => {
  const digits = value.replace(/^\+/, '');
  return (/^\+?[0-9]{6,15}$/.test(value)) && (digits.length === value.length || value.startsWith('+'));
};

/**
 * Titled field block. Groups are separated by a hairline and an eyebrow rather
 * than by their own borders, so the form never nests a card inside a card.
 */
const FieldGroup = ({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) => (
  <section className="border-t pt-5 first:border-t-0 first:pt-0">
    <h3 className="text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
      {title}
    </h3>
    <p className="mt-1 text-sm text-muted-foreground">{description}</p>
    <div className="mt-4 space-y-4">{children}</div>
  </section>
);

const isValidCTAValue = (cta: AdBannerCTA) =>
  cta.type === 'phone' ? isValidPhone(cta.value.trim()) : cta.value.trim().startsWith('https://');

/**
 * Campaign composer with duration presets and a 375px live preview rendering
 * the SAME campaign markup the employee sees (EmployeeAdContent).
 */
export const AdBannerComposer = ({ initial, forceFreshWindow = false, onDone }: AdBannerComposerProps) => {
  const { data: projects = [] } = useAllProjects();
  const createMutation = useCreateAdBanner();
  const updateMutation = useUpdateAdBanner();

  const todayStr = format(new Date(), 'yyyy-MM-dd');
  const [title, setTitle] = useState(initial?.title ?? '');
  const [body, setBody] = useState(initial?.body ?? '');
  const [bullets, setBullets] = useState<string[]>(
    initial?.bullets?.length ? initial.bullets : [''],
  );
  const [ctas, setCtas] = useState<AdBannerCTA[]>(
    initial?.ctas?.length ? initial.ctas : [emptyCTA()],
  );
  const [footer, setFooter] = useState(initial?.footer ?? '');
  const [targetProjectIds, setTargetProjectIds] = useState<number[]>(
    initial?.targetProjectIds ?? [],
  );
  const [priority, setPriority] = useState<number>(initial?.priority ?? 0);
  const [isActive, setIsActive] = useState(initial?.isActive ?? true);

  const initialStartsAt = initial && !forceFreshWindow ? toLocalDateInput(initial.startsAt) : todayStr;
  const [startsAt, setStartsAt] = useState(initialStartsAt);

  const initialPreset: DurationPreset = '30';
  const [preset, setPreset] = useState<DurationPreset>(initialPreset);
  const [customEndsAt, setCustomEndsAt] = useState(
    initial && !forceFreshWindow ? toLocalDateInput(initial.endsAt) : '',
  );

  const endsAtDate = useMemo(() => {
    const start = new Date(`${startsAt}T00:00:00`);
    if (Number.isNaN(start.getTime())) return null;
    if (preset !== 'custom') return addDays(start, Number(preset));
    if (!customEndsAt) return null;
    const end = new Date(`${customEndsAt}T00:00:00`);
    return Number.isNaN(end.getTime()) ? null : end;
  }, [startsAt, preset, customEndsAt]);

  const lifetimeDays = useMemo(() => {
    if (!endsAtDate) return null;
    const start = new Date(`${startsAt}T00:00:00`);
    return Math.round((endsAtDate.getTime() - start.getTime()) / 86_400_000);
  }, [startsAt, endsAtDate]);

  const previewBanner: AdBanner = {
    id: initial?.id ?? 0,
    title: title || 'Tiêu đề quảng cáo',
    body,
    bullets: bullets.filter((b) => b.trim() !== ''),
    ctas: ctas
      .filter((c) => c.label.trim() !== '')
      .map((c) => ({ ...c, value: c.value.trim() })),
    footer,
    targetProjectIds,
    priority,
    startsAt: `${startsAt}T00:00:00`,
    endsAt: endsAtDate ? endsAtDate.toISOString() : '',
    isActive: true,
    createdAt: initial?.createdAt ?? '',
    updatedAt: initial?.updatedAt ?? '',
  };

  const validationError = useMemo(() => {
    if (!title.trim()) return 'Tiêu đề là bắt buộc';
    if (title.trim().length > 255) return 'Tiêu đề tối đa 255 ký tự';
    if (bullets.length > MAX_BULLETS) return `Tối đa ${MAX_BULLETS} gạch đầu dòng`;
    if (bullets.some((b) => b.trim().length > 200)) return 'Mỗi gạch đầu dòng tối đa 200 ký tự';
    if (ctas.length === 0) return 'Cần ít nhất một nút hành động';
    if (ctas.length > MAX_CTAS) return `Tối đa ${MAX_CTAS} nút hành động`;
    if (ctas.some((c) => !c.label.trim())) return 'Nhãn nút hành động là bắt buộc';
    if (ctas.some((c) => c.label.trim().length > 40)) return 'Nhãn nút hành động tối đa 40 ký tự';
    if (ctas.some((c) => !isValidCTAValue(c))) {
      return 'Giá trị nút hành động không hợp lệ: số điện thoại là chữ số (6–15), đường dẫn bắt đầu bằng https://';
    }
    if (!endsAtDate) return 'Chọn thời gian kết thúc hợp lệ';
    if (lifetimeDays !== null && lifetimeDays <= 0) return 'Ngày kết thúc phải sau ngày bắt đầu';
    if (lifetimeDays !== null && lifetimeDays > MAX_LIFETIME_DAYS) {
      return `Thời gian hiển thị tối đa ${MAX_LIFETIME_DAYS} ngày`;
    }
    return null;
  }, [title, bullets, ctas, endsAtDate, lifetimeDays]);

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  const handleSubmit = () => {
    if (validationError) {
      toast.error(validationError);
      return;
    }
    const payload: AdBannerPayload = {
      title: title.trim(),
      body: body.trim(),
      bullets: bullets.map((b) => b.trim()).filter(Boolean),
      ctas: ctas.map((c) => ({ label: c.label.trim(), type: c.type, value: c.value.trim() })),
      footer: footer.trim(),
      targetProjectIds,
      priority,
      startsAt: new Date(`${startsAt}T00:00:00`).toISOString(),
      endsAt: endsAtDate!.toISOString(),
      isActive,
    };

    const mutation =
      initial && !forceFreshWindow
        ? updateMutation.mutateAsync({ id: initial.id, payload })
        : createMutation.mutateAsync(payload);

    mutation
      .then(() => {
        toast.success(initial && !forceFreshWindow ? 'Đã cập nhật chiến dịch' : 'Đã tạo chiến dịch');
        onDone();
      })
      .catch((error: unknown) => {
        toast.error(getErrorMessage(error));
      });
  };

  const ctaTypeOptions: Array<{ value: AdBannerCTA['type']; label: string }> = [
    { value: 'phone', label: 'Gọi điện (tel:)' },
    { value: 'url', label: 'Đường dẫn (https://)' },
    { value: 'zalo', label: 'Zalo' },
  ];

  return (
    <div className="space-y-6">
    {/* Two columns from `lg` up: the admin canvas caps at 1180px, so waiting
        for `xl` left the preview stranded under a half-empty form. */}
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px] lg:gap-8">
      <div className="min-w-0 space-y-6">
        <FieldGroup title="Nội dung" description="Những gì nhân viên đọc được trong thông báo.">
        <div className="space-y-1.5">
          <Label htmlFor="ad-title">Tiêu đề</Label>
          <Input
            id="ad-title"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="TING TING SOFTWARE SOLUTIONS xin thông báo"
            maxLength={255}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="ad-body">Nội dung</Label>
          <Textarea
            id="ad-body"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder="Từ nay, công nhân dự án có thể chấm công tự động và ứng lương ngay trên điện thoại."
            rows={3}
          />
        </div>

        <div className="space-y-2">
          <Label>Gạch đầu dòng (tối đa {MAX_BULLETS})</Label>
          {bullets.map((bullet, index) => (
            <div key={index} className="flex items-center gap-2">
              <span
                aria-hidden="true"
                className="w-4 shrink-0 text-xs tabular-nums text-muted-foreground"
              >
                {index + 1}
              </span>
              <Input
                aria-label={`Gạch đầu dòng ${index + 1}`}
                value={bullet}
                onChange={(e) => {
                  const next = [...bullets];
                  next[index] = e.target.value;
                  setBullets(next);
                }}
                maxLength={200}
                placeholder="Chấm công tự động, chính xác từng ca làm"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={`Xóa gạch đầu dòng ${index + 1}`}
                disabled={bullets.length === 1}
                onClick={() => setBullets(bullets.filter((_, i) => i !== index))}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
          {bullets.length < MAX_BULLETS && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => setBullets([...bullets, ''])}
            >
              <Plus className="h-3.5 w-3.5" /> Thêm gạch đầu dòng
            </Button>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="ad-footer">Câu chào kết</Label>
          <Input
            id="ad-footer"
            value={footer}
            onChange={(e) => setFooter(e.target.value)}
            placeholder="Ting Ting Software Solutions — Đồng hành cùng người lao động."
            maxLength={255}
          />
        </div>
        </FieldGroup>

        <FieldGroup title="Nút hành động" description="Nút đầu tiên là hành động chính, hiển thị nổi bật nhất.">
        <div className="space-y-2">
          <Label>Danh sách nút (1–{MAX_CTAS})</Label>
          {/* One bordered list with hairline rows instead of a bordered box per
              CTA — same structure, a quarter of the visual noise. */}
          <div className="divide-y overflow-hidden rounded-lg border bg-muted/30">
          {ctas.map((cta, index) => (
            <div key={index} className="grid gap-2 p-3 sm:grid-cols-[1fr_10rem_minmax(0,1.4fr)_auto] sm:items-center">
              <Input
                aria-label={`Nhãn nút ${index + 1}`}
                value={cta.label}
                onChange={(e) => {
                  const next = [...ctas];
                  next[index] = { ...cta, label: e.target.value };
                  setCtas(next);
                }}
                placeholder="Gọi hotline"
                maxLength={40}
              />
              <select
                aria-label={`Loại nút ${index + 1}`}
                value={cta.type}
                onChange={(e) => {
                  const next = [...ctas];
                  next[index] = { ...cta, type: e.target.value as AdBannerCTA['type'] };
                  setCtas(next);
                }}
                className="h-9 rounded-md border bg-background px-3 text-sm"
              >
                {ctaTypeOptions.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
              <Input
                aria-label={`Giá trị nút ${index + 1}`}
                value={cta.value}
                onChange={(e) => {
                  const next = [...ctas];
                  next[index] = { ...cta, value: e.target.value };
                  setCtas(next);
                }}
                placeholder={
                  cta.type === 'phone'
                    ? '0914827988'
                    : cta.type === 'zalo'
                      ? 'https://zalo.me/g/...'
                      : 'https://...'
                }
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={`Xóa nút ${index + 1}`}
                disabled={ctas.length === 1}
                onClick={() => setCtas(ctas.filter((_, i) => i !== index))}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
          </div>
          {ctas.length < MAX_CTAS && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => setCtas([...ctas, emptyCTA()])}
            >
              <Plus className="h-3.5 w-3.5" /> Thêm nút
            </Button>
          )}
        </div>
        </FieldGroup>

        <FieldGroup title="Đối tượng & lịch chạy" description="Ai nhìn thấy chiến dịch này, và trong khoảng thời gian nào.">
        <div className="space-y-1.5">
          <Label>Dự án hiển thị</Label>
          <ProjectMultiSelector
            value={targetProjectIds}
            onChange={setTargetProjectIds}
            projects={projects.map((p) => ({ id: p.id, code: p.code, name: p.name }))}
            placeholder="Tất cả dự án"
          />
          <p className="text-xs text-muted-foreground">
            Không chọn dự án nào = hiển thị cho tất cả nhân viên đang làm việc.
          </p>
        </div>

        <div className="space-y-2">
          <Label>Thời gian hiển thị</Label>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              aria-label="Ngày bắt đầu"
              type="date"
              value={startsAt}
              onChange={(e) => setStartsAt(e.target.value)}
              className="w-40"
            />
            {/* Segmented control: the presets are one choice, so they read as
                one control rather than five competing buttons. */}
            <div className="inline-flex rounded-lg border bg-muted/40 p-0.5">
              {DURATION_PRESETS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setPreset(option.value)}
                  className={cn(
                    'h-8 rounded-md px-3 text-sm font-medium transition-colors',
                    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                    preset === option.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {option.label}
                </button>
              ))}
            </div>
            {preset === 'custom' && (
              <Input
                aria-label="Ngày kết thúc"
                type="date"
                value={customEndsAt}
                onChange={(e) => setCustomEndsAt(e.target.value)}
                className="w-40"
              />
            )}
          </div>
          <p className="text-sm text-muted-foreground">
            {endsAtDate ? (
              <>
                Kết thúc{' '}
                <span className="font-medium tabular-nums text-foreground">
                  {format(endsAtDate, DATE_DISPLAY)}
                </span>
                {lifetimeDays !== null && (
                  <span className="tabular-nums"> · {lifetimeDays} ngày</span>
                )}
              </>
            ) : (
              'Chọn thời hạn để xem ngày kết thúc'
            )}
            {lifetimeDays !== null && lifetimeDays > MAX_LIFETIME_DAYS && (
              <span className="ml-2 font-medium text-destructive">vượt giới hạn 180 ngày</span>
            )}
          </p>
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="ad-priority">Độ ưu tiên</Label>
          <Input
            id="ad-priority"
            type="number"
            value={priority}
            onChange={(e) => setPriority(Number(e.target.value))}
            className="w-32"
          />
          <p className="text-xs text-muted-foreground">
            Khi nhiều chiến dịch cùng chạy, ưu tiên cao hơn hiển thị trước.
          </p>
        </div>
        </FieldGroup>
      </div>

      <aside className="lg:sticky lg:top-4 lg:self-start">
        <div className="mb-2 flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
          <Smartphone className="h-3.5 w-3.5" aria-hidden="true" />
          Xem trước trên điện thoại
        </div>
        <div
          data-theme="employee"
          className="w-full max-w-[375px] overflow-hidden rounded-xl border bg-[var(--employee-page)]"
        >
          <EmployeeAdContent banner={previewBanner} />
        </div>
      </aside>
    </div>

    {/* Actions span the whole composer, below both columns — the preview is
        reference material, not something to scroll past to reach Save. */}
    <div className="flex flex-wrap items-center gap-3 border-t pt-4">
      <Button type="button" onClick={handleSubmit} disabled={isSubmitting || !!validationError} className="gap-2">
        {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
        {initial && !forceFreshWindow ? 'Lưu thay đổi' : 'Đăng chiến dịch'}
      </Button>
      <Button type="button" variant="ghost" onClick={onDone}>
        Hủy
      </Button>
      <div className="ml-auto flex items-center gap-2">
        <Label htmlFor="ad-active" className="text-xs font-normal text-muted-foreground">
          Bật / Tắt
        </Label>
        <Switch id="ad-active" checked={isActive} onCheckedChange={setIsActive} />
      </div>
      {validationError && (
        <p role="alert" className="w-full text-sm text-destructive">{validationError}</p>
      )}
    </div>
    </div>
  );
};
