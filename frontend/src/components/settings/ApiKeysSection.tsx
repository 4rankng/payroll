import { type FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import {
  AlertTriangle,
  Bot,
  Calendar,
  Check,
  ChevronDown,
  Clock,
  Copy,
  Download,
  Eye,
  FileCode,
  KeyRound,
  Loader2,
  Plus,
  RefreshCw,
  ShieldAlert,
  Sparkles,
  Terminal,
  Trash2,
  Workflow,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useApiKeys, useCreateAPIKey, useRevokeAPIKey } from '@/hooks/api/useApiKeys';
import type { APIKey } from '@/services/api/apiKeys.service';
import skillMarkdown from '@/assets/skills/payroll-password-reset/SKILL.md?raw';

const DATE_TIME_DISPLAY = 'dd/MM/yyyy HH:mm';

// The path the downloaded skill belongs at. A zero-width space after each slash
// lets the long segment wrap at its separators instead of mid-word on narrow screens.
const SKILL_TARGET_PATH_DISPLAY = '.claude/skills/payroll-password-reset/SKILL.md'
  .split('/')
  .join('/\u200B');

const RAW_SKILL_TARGET_PATH = '.claude/skills/payroll-password-reset/SKILL.md';

const formatDateTime = (value: string | null): string =>
  value ? format(new Date(value), DATE_TIME_DISPLAY) : '—';

interface APIKeyRowProps {
  apiKey: APIKey;
  onRevoke: (apiKey: APIKey) => void;
  revoking: boolean;
  onCopyPrefix: (prefix: string, id: number) => void;
  copiedPrefixId: number | null;
}

const APIKeyRow = ({
  apiKey,
  onRevoke,
  revoking,
  onCopyPrefix,
  copiedPrefixId,
}: APIKeyRowProps) => {
  const revoked = apiKey.revoked_at != null;
  const isCopied = copiedPrefixId === apiKey.id;

  return (
    <li className="flex flex-col gap-3.5 p-4 transition-colors hover:bg-muted/30 sm:gap-4 sm:p-5 md:flex-row md:items-center md:justify-between">
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex flex-wrap items-center gap-2 sm:gap-2.5">
          <span className="break-words text-sm font-semibold text-foreground sm:text-base">
            {apiKey.name}
          </span>
          {revoked ? (
            <Badge
              variant="outline"
              className="gap-1.5 border-border bg-muted/60 px-2 py-0.5 text-xs text-muted-foreground"
            >
              <span className="size-1.5 rounded-full bg-muted-foreground/60" />
              Đã thu hồi
            </Badge>
          ) : (
            <Badge
              variant="outline"
              className="gap-1.5 border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-400"
            >
              <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Đang hoạt động
            </Badge>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex items-center gap-1.5 rounded-md border border-border/80 bg-muted/40 px-2 py-1 font-mono text-xs text-foreground">
            <KeyRound className="size-3 text-muted-foreground shrink-0" aria-hidden="true" />
            <span className="font-medium">{apiKey.key_prefix}••••••••••••</span>
            <button
              type="button"
              title="Sao chép tiền tố khoá"
              aria-label={`Sao chép tiền tố ${apiKey.key_prefix}`}
              onClick={() => onCopyPrefix(apiKey.key_prefix, apiKey.id)}
              className="ml-1 rounded p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            >
              {isCopied ? (
                <Check className="size-3 text-emerald-700 dark:text-emerald-400" aria-hidden="true" />
              ) : (
                <Copy className="size-3" aria-hidden="true" />
              )}
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <Calendar className="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
            <span>Tạo {formatDateTime(apiKey.created_at)}</span>
          </span>
          <span className="inline-flex items-center gap-1.5">
            <Clock className="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
            <span>
              Dùng lần cuối {apiKey.last_used_at ? formatDateTime(apiKey.last_used_at) : 'Chưa dùng'}
            </span>
          </span>
          {revoked && apiKey.revoked_at ? (
            <span className="inline-flex items-center gap-1.5 text-destructive">
              <ShieldAlert className="size-3.5 shrink-0" aria-hidden="true" />
              <span>Thu hồi {formatDateTime(apiKey.revoked_at)}</span>
            </span>
          ) : null}
        </div>
      </div>

      {!revoked ? (
        <div className="shrink-0 self-start md:self-center">
          <Button
            type="button"
            variant="outline"
            className="min-h-11 gap-1.5 border-destructive/30 text-destructive hover:border-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:ring-destructive sm:min-h-9"
            onClick={() => onRevoke(apiKey)}
            disabled={revoking}
          >
            {revoking ? (
              <Loader2 className="size-4 animate-spin" aria-hidden="true" />
            ) : (
              <Trash2 className="size-4" aria-hidden="true" />
            )}
            <span>Thu hồi</span>
          </Button>
        </div>
      ) : (
        <div className="shrink-0 self-start text-xs text-muted-foreground italic md:self-center">
          Vô hiệu hoá
        </div>
      )}
    </li>
  );
};

/** Admin control surface for machine API keys used by the chatbot integration. */
export const ApiKeysSection = () => {
  const { data, isLoading, isError, isFetching, refetch } = useApiKeys();
  const createKey = useCreateAPIKey();
  const revokeKey = useRevokeAPIKey();

  const [name, setName] = useState('');
  const [revealed, setRevealed] = useState<{ key: string; name: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const [revokingId, setRevokingId] = useState<number | null>(null);
  const [copiedPrefixId, setCopiedPrefixId] = useState<number | null>(null);
  const [pathCopied, setPathCopied] = useState(false);
  const [showSkillPreview, setShowSkillPreview] = useState(false);
  const [skillTextCopied, setSkillTextCopied] = useState(false);

  const keys = data?.data ?? [];
  const activeKeysCount = keys.filter((k) => k.revoked_at == null).length;
  const revokedKeysCount = keys.filter((k) => k.revoked_at != null).length;

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      toast.error('Nhập tên khoá API');
      return;
    }
    try {
      const res = await createKey.mutateAsync({ name: trimmed });
      if (res.data?.key) {
        setRevealed({ key: res.data.key, name: res.data.name });
      }
      setName('');
    } catch {
      // Failure toast is emitted by the hook.
    }
  };

  const handleCopy = async () => {
    if (!revealed) return;
    try {
      await navigator.clipboard.writeText(revealed.key);
      setCopied(true);
      toast.success('Đã sao chép khoá API vào clipboard');
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error('Không thể sao chép khoá');
    }
  };

  const handleCopyPrefix = async (prefix: string, id: number) => {
    try {
      await navigator.clipboard.writeText(prefix);
      setCopiedPrefixId(id);
      toast.success(`Đã sao chép tiền tố "${prefix}"`);
      setTimeout(() => setCopiedPrefixId(null), 1800);
    } catch {
      toast.error('Không thể sao chép tiền tố');
    }
  };

  const handleCopyPath = async () => {
    try {
      await navigator.clipboard.writeText(RAW_SKILL_TARGET_PATH);
      setPathCopied(true);
      toast.success('Đã sao chép đường dẫn SKILL.md');
      setTimeout(() => setPathCopied(false), 1800);
    } catch {
      toast.error('Không thể sao chép đường dẫn');
    }
  };

  const handleCopySkillText = async () => {
    try {
      await navigator.clipboard.writeText(skillMarkdown);
      setSkillTextCopied(true);
      toast.success('Đã sao chép toàn bộ nội dung SKILL.md');
      setTimeout(() => setSkillTextCopied(false), 1800);
    } catch {
      toast.error('Không thể sao chép nội dung');
    }
  };

  const handleRevoke = async (apiKey: APIKey) => {
    if (
      !window.confirm(
        `Thu hồi khoá "${apiKey.name}"? Chatbot đang dùng khoá này sẽ ngừng hoạt động ngay.`,
      )
    ) {
      return;
    }
    setRevokingId(apiKey.id);
    try {
      await revokeKey.mutateAsync(apiKey.id);
    } catch {
      // Failure toast is emitted by the hook.
    } finally {
      setRevokingId(null);
    }
  };

  const handleDownloadGuide = () => {
    const blob = new Blob([skillMarkdown], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'payroll-password-reset-SKILL.md';
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
    toast.success('Đã tải hướng dẫn API cho agent');
  };

  return (
    <div className="space-y-6">
      {/* 1. Header Overview & KPI Metric Rail */}
      <div className="overflow-hidden rounded-xl border bg-card shadow-sm">
        <div className="bg-gradient-to-r from-emerald-500/10 via-emerald-500/5 to-transparent p-5 sm:p-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex items-start gap-3.5">
              <div className="flex size-11 shrink-0 items-center justify-center rounded-xl border border-emerald-500/30 bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 shadow-sm">
                <KeyRound className="size-5.5" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-base font-semibold tracking-tight text-foreground sm:text-lg">
                    Khoá API & Tích hợp Machine-to-Machine
                  </h2>
                  <Badge
                    variant="outline"
                    className="border-emerald-500/30 bg-emerald-500/10 text-[11px] font-semibold text-emerald-700 dark:text-emerald-300"
                  >
                    Header: X-API-Key
                  </Badge>
                </div>
                <p className="mt-1 max-w-2xl text-xs sm:text-sm text-muted-foreground leading-relaxed">
                  Cấp và quản lý khoá API bảo mật cho các hệ thống máy (Zalo Chatbot, AI Agent, Webhook) gọi API đặt lại mật khẩu và xác thực nhân viên.
                </p>
              </div>
            </div>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void refetch()}
              disabled={isLoading || isFetching}
              className="min-h-11 sm:min-h-9 gap-1.5 self-start text-xs text-muted-foreground hover:text-foreground sm:self-auto"
              aria-label="Tải lại danh sách khoá API"
            >
              <RefreshCw
                className={cn('size-3.5', (isLoading || isFetching) && 'animate-spin')}
                aria-hidden="true"
              />
              <span>Tải lại</span>
            </Button>
          </div>
        </div>

        {/* Metric Rail */}
        <div className="grid grid-cols-2 divide-x divide-y border-t bg-muted/15 sm:grid-cols-4 sm:divide-y-0 text-sm">
          <div className="p-3.5 sm:p-4 text-center">
            <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
              Tổng số khoá
            </p>
            <p className="mt-1 text-xl font-bold tracking-tight text-foreground">{keys.length}</p>
          </div>
          <div className="p-3.5 sm:p-4 text-center">
            <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
              Đang hoạt động
            </p>
            <p className="mt-1 flex items-center justify-center gap-1.5 text-xl font-bold tracking-tight text-emerald-700 dark:text-emerald-400">
              <span className="size-2 rounded-full bg-emerald-500 inline-block animate-pulse" />
              {activeKeysCount}
            </p>
          </div>
          <div className="p-3.5 sm:p-4 text-center">
            <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
              Đã thu hồi
            </p>
            <p className="mt-1 flex items-center justify-center gap-1.5 text-xl font-bold tracking-tight text-muted-foreground">
              <span className="size-2 rounded-full bg-muted-foreground/50 inline-block" />
              {revokedKeysCount}
            </p>
          </div>
          <div className="p-3.5 sm:p-4 text-center">
            <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
              Tích hợp Claude
            </p>
            <p className="mt-1 flex items-center justify-center gap-1 text-xs font-semibold text-primary">
              <Sparkles className="size-3.5 text-amber-700" aria-hidden="true" />
              <span>SKILL.md sẵn sàng</span>
            </p>
          </div>
        </div>
      </div>

      {/* 2. Cấp khoá API mới */}
      <section id="api-keys-create" aria-labelledby="api-keys-create-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/20 bg-primary/10 text-primary">
                <Plus className="size-4" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <CardTitle id="api-keys-create-title" className="text-sm sm:text-base font-semibold text-foreground">
                  Tạo khoá API
                </CardTitle>
                <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                  Cấp một khoá cho hệ thống bên ngoài (ví dụ chatbot Zalo) gọi API đặt lại mật khẩu. Khoá chỉ hiển thị một lần khi tạo.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-4 sm:p-5 space-y-4">
            <form onSubmit={handleCreate} className="space-y-3">
              <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
                <div className="space-y-1.5">
                  <Label htmlFor="api-key-name" className="text-xs sm:text-sm font-medium text-foreground">
                    Tên khoá <span className="text-destructive">*</span>
                  </Label>
                  <div className="relative">
                    <Input
                      id="api-key-name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="Ví dụ: Zalo Chatbot, Claude Assistant..."
                      maxLength={100}
                      className="min-h-11 pr-14 text-sm sm:min-h-9"
                      disabled={createKey.isPending}
                    />
                    <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-[11px] text-muted-foreground select-none">
                      {name.length}/100
                    </span>
                  </div>
                </div>
                <Button
                  type="submit"
                  className="min-h-11 sm:min-h-9 gap-1.5 bg-emerald-700 text-white hover:bg-emerald-800 font-medium"
                  disabled={createKey.isPending || !name.trim()}
                >
                  {createKey.isPending ? (
                    <Loader2 className="size-4 animate-spin" aria-hidden="true" />
                  ) : (
                    <KeyRound className="size-4" aria-hidden="true" />
                  )}
                  <span>Tạo khoá API</span>
                </Button>
              </div>
              <p className="text-[11px] sm:text-xs text-muted-foreground">
                Mẹo: Đặt tên mô tả rõ hệ thống hoặc môi trường sử dụng để dễ quản lý và theo dõi nhật ký hoạt động.
              </p>
            </form>

            {revealed ? (
              <div
                role="alert"
                className="mt-4 space-y-3 rounded-xl border border-amber-300/80 bg-amber-50/80 p-4 text-sm shadow-sm dark:border-amber-700/60 dark:bg-amber-950/40"
              >
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex items-center gap-2">
                    <AlertTriangle className="size-5 shrink-0 text-amber-700 dark:text-amber-400" aria-hidden="true" />
                    <div>
                      <p className="font-semibold text-amber-950 dark:text-amber-200">
                        Khoá chỉ hiển thị một lần. Hãy sao chép ngay.
                      </p>
                      <p className="text-xs text-amber-900 dark:text-amber-300/80">
                        Khoá được cấp cho <span className="font-medium underline">{revealed.name}</span>. Bạn sẽ không thể xem lại khoá sau khi đóng thông báo này.
                      </p>
                    </div>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="h-8 self-end text-xs text-amber-900 hover:bg-amber-200/50 dark:text-amber-200 sm:self-center"
                    onClick={() => setRevealed(null)}
                  >
                    Đã lưu khoá (Đóng)
                  </Button>
                </div>

                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Input
                    readOnly
                    value={revealed.key}
                    aria-label={`Khoá API cho ${revealed.name}`}
                    className="min-h-11 flex-1 border-amber-300/80 bg-background font-mono text-xs sm:min-h-9 sm:text-sm font-semibold selection:bg-amber-200"
                    onFocus={(e) => e.currentTarget.select()}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    className="min-h-11 shrink-0 gap-1.5 border-amber-400/80 bg-background hover:bg-amber-100/50 sm:min-h-9"
                    onClick={handleCopy}
                  >
                    {copied ? (
                      <Check className="size-4 text-emerald-700" aria-hidden="true" />
                    ) : (
                      <Copy className="size-4" aria-hidden="true" />
                    )}
                    <span>{copied ? 'Đã sao chép' : 'Sao chép'}</span>
                  </Button>
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </section>

      {/* 3. Khoá đang có */}
      <section id="api-keys-list" aria-labelledby="api-keys-list-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-muted-foreground/20 bg-muted text-foreground">
                  <KeyRound className="size-4" aria-hidden="true" />
                </div>
                <div>
                  <CardTitle id="api-keys-list-title" className="text-sm sm:text-base font-semibold text-foreground">
                    Khoá đang có
                  </CardTitle>
                  <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                    Danh sách khoá API đã cấp. Thu hồi để chặn một khoá ngay lập tức.
                  </CardDescription>
                </div>
              </div>
              <Badge variant="outline" className="text-xs font-semibold px-2 py-0.5">
                {keys.length} khoá
              </Badge>
            </div>
          </CardHeader>

          <CardContent className="p-0">
            {isLoading ? (
              <div className="space-y-4 p-5">
                <div className="flex items-center gap-3">
                  <Skeleton className="size-8 rounded-lg" />
                  <div className="space-y-1.5 flex-1">
                    <Skeleton className="h-4 w-40" />
                    <Skeleton className="h-3 w-64" />
                  </div>
                </div>
                <div className="flex items-center gap-3 pt-3 border-t">
                  <Skeleton className="size-8 rounded-lg" />
                  <div className="space-y-1.5 flex-1">
                    <Skeleton className="h-4 w-48" />
                    <Skeleton className="h-3 w-56" />
                  </div>
                </div>
              </div>
            ) : isError ? (
              <div className="flex flex-col items-center justify-center gap-3 p-8 text-center text-sm text-muted-foreground">
                <ShieldAlert className="size-8 text-destructive" aria-hidden="true" />
                <p className="font-medium text-foreground">Không thể tải danh sách khoá API</p>
                <p className="text-xs text-muted-foreground max-w-sm">
                  Đã xảy ra lỗi khi truy vấn danh sách khoá từ máy chủ. Vui lòng thử lại.
                </p>
                <Button
                  type="button"
                  variant="outline"
                  className="min-h-11 sm:min-h-9 gap-1.5 mt-2"
                  onClick={() => void refetch()}
                >
                  <RefreshCw className="size-3.5" aria-hidden="true" />
                  <span>Thử lại</span>
                </Button>
              </div>
            ) : keys.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-2.5 p-10 text-center text-sm text-muted-foreground">
                <div className="flex size-12 items-center justify-center rounded-full bg-muted/60 text-muted-foreground">
                  <KeyRound className="size-6" aria-hidden="true" />
                </div>
                <p className="font-medium text-foreground">Chưa có khoá API nào</p>
                <p className="text-xs text-muted-foreground max-w-sm">
                  Tạo khoá API đầu tiên ở khung phía trên để cấp quyền truy cập cho chatbot Zalo hoặc agent hỗ trợ nhân viên.
                </p>
              </div>
            ) : (
              <ul className="divide-y divide-border/60">
                {keys.map((apiKey) => (
                  <APIKeyRow
                    key={apiKey.id}
                    apiKey={apiKey}
                    onRevoke={handleRevoke}
                    revoking={revokingId === apiKey.id}
                    onCopyPrefix={handleCopyPrefix}
                    copiedPrefixId={copiedPrefixId}
                  />
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </section>

      {/* 4. Hướng dẫn cho agent */}
      <section id="api-keys-guide" aria-labelledby="api-keys-guide-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-violet-500/20 bg-violet-500/10 text-violet-700 dark:text-violet-400">
                  <Bot className="size-4" aria-hidden="true" />
                </div>
                <div>
                  <CardTitle id="api-keys-guide-title" className="text-sm sm:text-base font-semibold text-foreground">
                    Hướng dẫn cho agent
                  </CardTitle>
                  <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                    Tải hướng dẫn tích hợp API dưới dạng Claude skill để agent (chatbot) biết cách gọi các API này.
                  </CardDescription>
                </div>
              </div>
              <Badge
                variant="outline"
                className="w-fit border-violet-500/30 bg-violet-500/10 text-[11px] font-medium text-violet-700 dark:text-violet-300"
              >
                Claude Skill Specification
              </Badge>
            </div>
          </CardHeader>

          <CardContent className="p-4 sm:p-5 space-y-4">
            {/* Quick 3-step feature overview */}
            <div className="grid gap-3 sm:grid-cols-3 text-xs">
              <div className="rounded-lg border border-border/80 bg-muted/20 p-3 space-y-1">
                <div className="flex items-center gap-1.5 font-semibold text-foreground">
                  <Workflow className="size-3.5 text-primary shrink-0" aria-hidden="true" />
                  <span>1. Tra cứu nhân sự</span>
                </div>
                <p className="text-muted-foreground text-[11px] leading-relaxed">
                  Gọi <code className="font-mono text-[10px] bg-muted px-1 py-0.5 rounded">/lookup</code> để xác thực số điện thoại và tên nhân viên.
                </p>
              </div>

              <div className="rounded-lg border border-border/80 bg-muted/20 p-3 space-y-1">
                <div className="flex items-center gap-1.5 font-semibold text-foreground">
                  <ShieldAlert className="size-3.5 text-primary shrink-0" aria-hidden="true" />
                  <span>2. Xác thực OTP ZNS</span>
                </div>
                <p className="text-muted-foreground text-[11px] leading-relaxed">
                  Gửi mã xác thực qua Zalo ZNS và kiểm tra mã OTP với <code className="font-mono text-[10px] bg-muted px-1 py-0.5 rounded">/otp</code> &amp; <code className="font-mono text-[10px] bg-muted px-1 py-0.5 rounded">/verify</code>.
                </p>
              </div>

              <div className="rounded-lg border border-border/80 bg-muted/20 p-3 space-y-1">
                <div className="flex items-center gap-1.5 font-semibold text-foreground">
                  <KeyRound className="size-3.5 text-primary shrink-0" aria-hidden="true" />
                  <span>3. Đặt lại mật khẩu</span>
                </div>
                <p className="text-muted-foreground text-[11px] leading-relaxed">
                  Cấp mật khẩu mới an toàn qua <code className="font-mono text-[10px] bg-muted px-1 py-0.5 rounded">/reset</code> và thông báo cho người dùng.
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-xs sm:text-sm leading-5 text-muted-foreground">
                Tệp <span className="font-mono text-xs font-semibold text-foreground">SKILL.md</span> mô tả 4 endpoint tích hợp, quy trình đặt lại
                mật khẩu 3 bước và cách xử lý lỗi. Đặt tệp vào thư mục skill của agent:
              </p>

              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <div className="flex flex-1 items-center gap-2 rounded-lg border border-border/80 bg-muted/50 px-3 py-2 text-xs">
                  <Terminal className="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
                  <code className="min-w-0 flex-1 break-all font-mono text-xs leading-5 text-foreground">
                    {SKILL_TARGET_PATH_DISPLAY}
                  </code>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="min-h-11 sm:min-h-9 shrink-0 gap-1.5 text-xs"
                  onClick={handleCopyPath}
                >
                  {pathCopied ? (
                    <Check className="size-3.5 text-emerald-700" aria-hidden="true" />
                  ) : (
                    <Copy className="size-3.5" aria-hidden="true" />
                  )}
                  <span>{pathCopied ? 'Đã sao chép' : 'Sao chép đường dẫn'}</span>
                </Button>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2.5 pt-1">
              <Button
                type="button"
                variant="outline"
                className="min-h-11 sm:min-h-9 gap-1.5 text-xs sm:text-sm font-medium"
                onClick={handleDownloadGuide}
              >
                <Download className="size-4" aria-hidden="true" />
                <span>Tải hướng dẫn API (SKILL.md)</span>
              </Button>

              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="min-h-11 sm:min-h-9 gap-1.5 text-xs text-muted-foreground hover:text-foreground"
                onClick={() => setShowSkillPreview((prev) => !prev)}
                aria-expanded={showSkillPreview}
              >
                {showSkillPreview ? (
                  <Eye className="size-3.5 text-primary" aria-hidden="true" />
                ) : (
                  <FileCode className="size-3.5" aria-hidden="true" />
                )}
                <span>{showSkillPreview ? 'Thu gọn xem trước' : 'Xem trước nội dung SKILL.md'}</span>
                <ChevronDown
                  className={cn('size-3.5 transition-transform duration-200', showSkillPreview && 'rotate-180')}
                  aria-hidden="true"
                />
              </Button>
            </div>

            {/* Collapsible preview of SKILL.md */}
            {showSkillPreview ? (
              <div className="space-y-2 rounded-xl border border-border bg-muted/40 p-4 transition-all">
                <div className="flex items-center justify-between pb-2 border-b">
                  <div className="flex items-center gap-2 text-xs font-medium text-foreground">
                    <FileCode className="size-4 text-violet-700" aria-hidden="true" />
                    <span>Nội dung tệp: SKILL.md</span>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-7 gap-1 text-[11px]"
                    onClick={handleCopySkillText}
                  >
                    {skillTextCopied ? (
                      <Check className="size-3 text-emerald-700" aria-hidden="true" />
                    ) : (
                      <Copy className="size-3" aria-hidden="true" />
                    )}
                    <span>{skillTextCopied ? 'Đã sao chép' : 'Sao chép văn bản'}</span>
                  </Button>
                </div>
                <pre className="max-h-72 overflow-y-auto whitespace-pre-wrap break-words rounded-md bg-background/80 p-3 font-mono text-[11px] leading-relaxed text-muted-foreground border">
                  {skillMarkdown}
                </pre>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </section>
    </div>
  );
};
