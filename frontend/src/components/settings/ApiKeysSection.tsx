import { type FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import {
  AlertTriangle,
  Calendar,
  Check,
  Clock,
  Copy,
  KeyRound,
  Loader2,
  Plus,
  RefreshCw,
  ShieldAlert,
  Trash2,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { useApiKeys, useCreateAPIKey, useRevokeAPIKey } from '@/hooks/api/useApiKeys';
import type { APIKey, APIKeyScope } from '@/services/api/apiKeys.service';

const DATE_TIME_DISPLAY = 'dd/MM/yyyy HH:mm';

const formatDateTime = (value: string | null): string =>
  value ? format(new Date(value), DATE_TIME_DISPLAY) : '—';

/** Human labels + styling for each key scope. */
const SCOPE_META: Record<
  APIKeyScope,
  { label: string; description: string; badgeClass: string }
> = {
  timesheet_read: {
    label: 'Vendor — bảng chấm công',
    description:
      'Chỉ đọc, dành cho đối tác bên ngoài: xem danh sách dự án, nhân viên và bảng chấm công. Không thể ghi dữ liệu.',
    badgeClass:
      'border-violet-500/30 bg-violet-500/10 px-2 py-0.5 text-xs font-medium text-violet-700 dark:text-violet-300',
  },
};

/**
 * Resolve a key's scope, defaulting a missing value to 'timesheet_read'. Any
 * unrecognized stored scope (e.g. a retired legacy value) is flagged
 * 'unsupported' so the row renders it as no longer supported.
 */
const scopeOf = (apiKey: APIKey): APIKeyScope | 'unsupported' => {
  const raw: string | null | undefined = apiKey.scope;
  return raw == null || raw === '' || raw === 'timesheet_read' ? 'timesheet_read' : 'unsupported';
};

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
  const scopeKey = scopeOf(apiKey);
  const scope = scopeKey === 'unsupported' ? null : SCOPE_META[scopeKey];

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
          {scope ? (
            <Badge variant="outline" className={scope.badgeClass}>
              {scope.label}
            </Badge>
          ) : (
            <Badge variant="outline" className="border-border bg-muted/60 px-2 py-0.5 text-xs text-muted-foreground">
              Không còn hỗ trợ
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

/** Admin control surface for machine API keys used by the vendor integration. */
export const ApiKeysSection = () => {
  const { data, isLoading, isError, isFetching, refetch } = useApiKeys();
  const createKey = useCreateAPIKey();
  const revokeKey = useRevokeAPIKey();

  const [name, setName] = useState('');
  const [revealed, setRevealed] = useState<{ key: string; name: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const [revokingId, setRevokingId] = useState<number | null>(null);
  const [copiedPrefixId, setCopiedPrefixId] = useState<number | null>(null);

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
      const res = await createKey.mutateAsync({ name: trimmed, scope: 'timesheet_read' });
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

  const handleRevoke = async (apiKey: APIKey) => {
    const scopeLabel = scopeOf(apiKey) === 'unsupported' ? 'Không còn hỗ trợ' : SCOPE_META.timesheet_read.label;
    const impact = 'Đối tác đang dùng khoá này sẽ mất quyền xem bảng chấm công ngay.';
    if (
      !window.confirm(
        `Thu hồi khoá "${apiKey.name}" (${scopeLabel})? ${impact}`,
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
                  Cấp và quản lý khoá API bảo mật cho các hệ thống máy (vendor, đối tác) tra cứu bảng chấm công.
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
        <div className="grid grid-cols-2 divide-x divide-y border-t bg-muted/15 sm:grid-cols-3 sm:divide-y-0 text-sm">
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
                  Cấp khoá cho hệ thống vendor tra cứu bảng chấm công (chỉ đọc). Khoá chỉ hiển thị một lần khi tạo.
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
                      placeholder="Ví dụ: Đối tác chấm công, Hệ thống đối soát..."
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
                  Tạo khoá API đầu tiên ở khung phía trên để cấp quyền truy cập cho hệ thống vendor tra cứu bảng chấm công.
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

    </div>
  );
};
