import { type FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import {
  AlertTriangle,
  Check,
  Copy,
  FileKey2,
  KeyRound,
  Loader2,
  Lock,
  Plus,
  Power,
  RefreshCw,
  ShieldAlert,
  Terminal,
  Trash2,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { API_CONFIG } from '@/config/api.config';
import {
  useOpsApiKeys,
  useCreateOpsAPIKey,
  useRevokeOpsAPIKey,
  useSetOpsAPIKeyScope,
} from '@/hooks/api/useOpsApiKeys';
import type { OpsAPIKey, OpsAPIKeyScope } from '@/services/api/opsApiKeys.service';

const DATE_TIME_DISPLAY = 'dd/MM/yyyy HH:mm';

const formatDateTime = (value: string | null): string =>
  value ? format(new Date(value), DATE_TIME_DISPLAY) : '—';

/** Base URL an ops agent's server.mjs calls (the backend API origin). */
const OPS_API_BASE = `${API_CONFIG.baseURL || (typeof window !== 'undefined' ? window.location.origin : '')}${API_CONFIG.apiV1}`;

const SCOPE_OPTIONS: ReadonlyArray<{
  value: OpsAPIKeyScope;
  label: string;
  description: string;
}> = [
  {
    value: 'read_only',
    label: 'Chỉ đọc (READ ONLY)',
    description: 'Chỉ các GET — mặc định an toàn khi tạo khoá',
  },
  {
    value: 'read_write',
    label: 'Đọc + ghi (READ WRITE)',
    description: 'Mọi thao tác (POST/PUT/PATCH/DELETE) — tự hồi về chỉ đọc sau cửa sổ ghi',
  },
];

const scopeBadgeClass = (scope: OpsAPIKeyScope): string =>
  scope === 'read_write'
    ? 'border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-700 dark:text-amber-400'
    : 'border-border bg-muted/60 px-2 py-0.5 text-xs font-medium text-muted-foreground';

const scopeLabel = (scope: OpsAPIKeyScope): string =>
  scope === 'read_write' ? 'READ WRITE' : 'READ ONLY';

interface OpsKeyRowProps {
  apiKey: OpsAPIKey;
  onToggleScope: (apiKey: OpsAPIKey) => void;
  onRevoke: (apiKey: OpsAPIKey) => void;
  toggling: boolean;
  revoking: boolean;
}

const OpsKeyRow = ({ apiKey, onToggleScope, onRevoke, toggling, revoking }: OpsKeyRowProps) => {
  const revoked = apiKey.revoked_at != null;
  // The stored scope says read_write but the window lapsed — show the truth.
  const windowLapsed =
    apiKey.scope === 'read_write' && apiKey.effective_scope === 'read_only';

  return (
    <li className="flex flex-col gap-3 p-4 transition-colors hover:bg-muted/30 sm:flex-row sm:items-center sm:justify-between sm:p-5">
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="break-words text-sm font-semibold text-foreground sm:text-base">
            {apiKey.name}
          </span>
          <Badge variant="outline" className={scopeBadgeClass(apiKey.effective_scope)}>
            {scopeLabel(apiKey.effective_scope)}
          </Badge>
          {windowLapsed ? (
            <Badge
              variant="outline"
              className="gap-1.5 border-border bg-muted/60 px-2 py-0.5 text-xs text-muted-foreground"
            >
              <Power className="size-3" aria-hidden="true" />
              Hết cửa sổ ghi
            </Badge>
          ) : null}
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

        <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1.5 font-mono">
            <KeyRound className="size-3 shrink-0" aria-hidden="true" />
            {apiKey.key_prefix}••••••••••••
          </span>
          <span>Tạo {formatDateTime(apiKey.created_at)}</span>
          <span>
            Dùng lần cuối {apiKey.last_used_at ? formatDateTime(apiKey.last_used_at) : 'Chưa dùng'}
          </span>
          {apiKey.write_window_expires_at ? (
            <span>Cửa sổ ghi đến {formatDateTime(apiKey.write_window_expires_at)}</span>
          ) : null}
        </div>
      </div>

      {!revoked ? (
        <div className="flex shrink-0 flex-col gap-2 self-start sm:flex-row sm:items-center sm:self-center">
          <Button
            type="button"
            variant="outline"
            className="min-h-11 gap-1.5 border-amber-500/40 text-amber-700 hover:border-amber-500 hover:bg-amber-500/10 hover:text-amber-800 focus-visible:ring-amber-500 sm:min-h-9 dark:text-amber-400"
            onClick={() => onToggleScope(apiKey)}
            disabled={toggling || revoking}
          >
            {toggling ? (
              <Loader2 className="size-4 animate-spin" aria-hidden="true" />
            ) : (
              <Lock className="size-4" aria-hidden="true" />
            )}
            <span>
              {apiKey.scope === 'read_write' ? 'Về chỉ đọc' : 'Cấp ghi (8h)'}
            </span>
          </Button>
          <Button
            type="button"
            variant="outline"
            className="min-h-11 gap-1.5 border-destructive/30 text-destructive hover:border-destructive hover:bg-destructive/10 hover:text-destructive focus-visible:ring-destructive sm:min-h-9"
            onClick={() => onRevoke(apiKey)}
            disabled={toggling || revoking}
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

/**
 * Admin console for pk_ops_* keys (payroll-ops-mcp channel) plus the
 * "Cài đặt MCP vào agent của bạn" install guide: create a key, store it in a
 * mode-600 secrets file, wire the MCP client config, mind the scope, know
 * the kill switch.
 */
export const OpsApiKeysSection = () => {
  const { data, isLoading, isError, isFetching, refetch } = useOpsApiKeys();
  const createKey = useCreateOpsAPIKey();
  const revokeKey = useRevokeOpsAPIKey();
  const setScope = useSetOpsAPIKeyScope();

  const [name, setName] = useState('');
  const [scope, setScopeValue] = useState<OpsAPIKeyScope | null>(null);
  const [writeWindowHours, setWriteWindowHours] = useState('8');
  const [revealed, setRevealed] = useState<{ key: string; name: string } | null>(null);
  const [copiedKey, setCopiedKey] = useState(false);
  const [revokingId, setRevokingId] = useState<number | null>(null);
  const [togglingId, setTogglingId] = useState<number | null>(null);
  const [copiedSnippet, setCopiedSnippet] = useState<string | null>(null);

  const keys = data?.data ?? [];

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      toast.error('Nhập tên khoá ops (tên agent)');
      return;
    }
    if (!scope) {
      toast.error('Chọn phạm vi truy cập cho khoá ops');
      return;
    }
    const hours = Number.parseInt(writeWindowHours, 10);
    try {
      const res = await createKey.mutateAsync({
        name: trimmed,
        scope,
        ...(scope === 'read_write' && Number.isFinite(hours) && hours > 0
          ? { write_window_hours: hours }
          : {}),
      });
      if (res.data?.key) {
        setRevealed({ key: res.data.key, name: res.data.name });
      }
      setName('');
      setScopeValue(null);
      setWriteWindowHours('8');
    } catch {
      // Failure toast is emitted by the hook.
    }
  };

  const handleCopyKey = async () => {
    if (!revealed) return;
    try {
      await navigator.clipboard.writeText(revealed.key);
      setCopiedKey(true);
      toast.success('Đã sao chép khoá ops vào clipboard');
      setTimeout(() => setCopiedKey(false), 2000);
    } catch {
      toast.error('Không thể sao chép khoá');
    }
  };

  const handleCopySnippet = async (id: string, text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedSnippet(id);
      toast.success('Đã sao chép vào clipboard');
      setTimeout(() => setCopiedSnippet(null), 1800);
    } catch {
      toast.error('Không thể sao chép nội dung');
    }
  };

  const handleToggleScope = async (apiKey: OpsAPIKey) => {
    // Base the action on the STORED scope: a lapsed read_write key already
    // behaves read-only, so its toggle persists that revert; a read_only
    // key grants write (opening a fresh 8h window).
    const toReadWrite = apiKey.scope !== 'read_write';
    const confirmed = window.confirm(
      toReadWrite
        ? `Cấp quyền ghi cho agent "${apiKey.name}"? Khoá sẽ được mọi thao tác (POST/PUT/PATCH/DELETE) với quyền quản trị và TỰ HỒI về chỉ đọc sau 8 giờ. Yêu cầu phiên đã xác thực OTP.`
        : `Chuyển khoá "${apiKey.name}" về chỉ đọc? Agent sẽ bị chặn mọi thao tác ghi ở lượt gọi tiếp theo.`,
    );    if (!confirmed) return;
    setTogglingId(apiKey.id);
    try {
      await setScope.mutateAsync({
        id: apiKey.id,
        payload: { scope: toReadWrite ? 'read_write' : 'read_only' },
      });
    } catch {
      // Failure toast is emitted by the hook (incl. the OTP 403 case).
    } finally {
      setTogglingId(null);
    }
  };

  const handleRevoke = async (apiKey: OpsAPIKey) => {
    if (
      !window.confirm(
        `Thu hồi khoá ops "${apiKey.name}" (${scopeLabel(apiKey.effective_scope)})? Agent đang dùng khoá này sẽ mất quyền truy cập ngay.`,
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

  // ── Install-guide snippets (step 2 & 3 of the guide) ─────────────────────
  const secretsFileSnippet = [
    'mkdir -p ~/.secrets',
    'echo -n "pk_ops_DÁN_KHOÁ_VỪA-TẠO-VÀO-ĐÂY" > ~/.secrets/payroll-ops-agent.key',
    'chmod 600 ~/.secrets/payroll-ops-agent.key',
  ].join('\n');
  const mcpConfigSnippet = JSON.stringify(
    {
      mcpServers: {
        'payroll-ops': {
          command: 'node',
          args: ['/duong/dan/toi/payroll-ops-mcp/server.mjs'],
          env: {
            PAYROLL_OPS_KEY_FILE: '~/.secrets/payroll-ops-agent.key',
            PAYROLL_BASE_URL: OPS_API_BASE,
          },
        },
      },
    },
    null,
    2,
  );
  const claudeCodeSnippet = `claude mcp add payroll-ops -e PAYROLL_OPS_KEY_FILE=~/.secrets/payroll-ops-agent.key -e PAYROLL_BASE_URL=${OPS_API_BASE} -- node /duong/dan/toi/payroll-ops-mcp/server.mjs`;
  const killSwitchSnippet = 'PAYROLL_OPS_MCP_DISABLED=1';

  const copyButton = (id: string, text: string, label?: string) => (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className="min-h-11 sm:min-h-9 shrink-0 gap-1.5 text-xs"
      onClick={() => void handleCopySnippet(id, text)}
    >
      {copiedSnippet === id ? (
        <Check className="size-3.5 text-emerald-700" aria-hidden="true" />
      ) : (
        <Copy className="size-3.5" aria-hidden="true" />
      )}
      <span>{copiedSnippet === id ? 'Đã sao chép' : (label ?? 'Sao chép')}</span>
    </Button>
  );

  const snippetBlock = (id: string, text: string, label?: string) => (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
      <pre className="flex-1 overflow-x-auto rounded-lg border border-border/80 bg-muted/50 p-3 font-mono text-[11px] leading-relaxed text-foreground">
        {text}
      </pre>
      {copyButton(id, text, label)}
    </div>
  );

  return (
    <div className="space-y-6">
      {/* 1. Tạo khoá ops (pk_ops_*) */}
      <section id="ops-keys-create" aria-labelledby="ops-keys-create-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/20 bg-primary/10 text-primary">
                <Plus className="size-4" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <CardTitle id="ops-keys-create-title" className="text-sm sm:text-base font-semibold text-foreground">
                  Tạo khoá ops cho agent (payroll-ops-mcp)
                </CardTitle>
                <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                  Khoá <span className="font-mono">pk_ops_*</span> chạy với quyền quản trị của bạn, giới hạn bởi phạm vi
                  khoá. Mỗi agent một khoá — audit ghi rõ agent nào gọi gì. Khoá chỉ hiển thị một lần khi tạo.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-4 sm:p-5 space-y-4">
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
                <div className="space-y-1.5">
                  <Label htmlFor="ops-key-name" className="text-xs sm:text-sm font-medium text-foreground">
                    Tên khoá (tên agent) <span className="text-destructive">*</span>
                  </Label>
                  <Input
                    id="ops-key-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Ví dụ: hermes, claude-code-bao-cao"
                    maxLength={100}
                    className="min-h-11 text-sm sm:min-h-9"
                    disabled={createKey.isPending}
                  />
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
                  <span>Tạo khoá ops</span>
                </Button>
              </div>

              <fieldset className="space-y-1.5">
                <legend className="text-xs sm:text-sm font-medium text-foreground">
                  Phạm vi truy cập <span className="text-destructive">*</span>
                </legend>
                <div className="grid gap-2 sm:grid-cols-2">
                  {SCOPE_OPTIONS.map((option) => {
                    const inputId = `ops-key-scope-${option.value}`;
                    return (
                      <label
                        key={option.value}
                        htmlFor={inputId}
                        className={cn(
                          'flex cursor-pointer items-start gap-2.5 rounded-lg border p-3 transition-colors',
                          'focus-within:ring-1 focus-within:ring-ring',
                          scope === option.value
                            ? 'border-emerald-500/50 bg-emerald-500/5'
                            : 'border-border hover:bg-muted/40',
                        )}
                      >
                        <input
                          id={inputId}
                          type="radio"
                          name="ops-key-scope"
                          value={option.value}
                          checked={scope === option.value}
                          onChange={() => setScopeValue(option.value)}
                          disabled={createKey.isPending}
                          className="mt-0.5 size-4 shrink-0 accent-emerald-700 dark:accent-emerald-400"
                        />
                        <span className="min-w-0 space-y-0.5">
                          <span className="block text-xs sm:text-sm font-medium text-foreground">
                            {option.label}
                          </span>
                          <span className="block text-[11px] sm:text-xs text-muted-foreground">
                            {option.description}
                          </span>
                        </span>
                      </label>
                    );
                  })}
                </div>
              </fieldset>

              {scope === 'read_write' ? (
                <div className="space-y-1.5">
                  <Label htmlFor="ops-key-window" className="text-xs sm:text-sm font-medium text-foreground">
                    Cửa sổ ghi (giờ)
                  </Label>
                  <Input
                    id="ops-key-window"
                    type="number"
                    min={1}
                    max={168}
                    value={writeWindowHours}
                    onChange={(e) => setWriteWindowHours(e.target.value)}
                    className="min-h-11 w-32 text-sm sm:min-h-9"
                    disabled={createKey.isPending}
                  />
                  <p className="text-[11px] text-muted-foreground">
                    Sau thời gian này khoá tự hồi về chỉ đọc (mặc định 8 giờ).
                  </p>
                </div>
              ) : null}
            </form>

            {revealed ? (
              <div
                role="alert"
                className="mt-4 space-y-3 rounded-xl border border-amber-300/80 bg-amber-50/80 p-4 text-sm shadow-sm dark:border-amber-700/60 dark:bg-amber-950/40"
              >
                <div className="flex items-center gap-2">
                  <AlertTriangle className="size-5 shrink-0 text-amber-700 dark:text-amber-400" aria-hidden="true" />
                  <div>
                    <p className="font-semibold text-amber-950 dark:text-amber-200">
                      Khoá chỉ hiển thị một lần — sao chép ngay
                    </p>
                    <p className="text-xs text-amber-900 dark:text-amber-300/80">
                      Khoá ops được cấp cho <span className="font-medium underline">{revealed.name}</span>. Lưu vào file
                      bí mật theo hướng dẫn bên dưới — không bao giờ dán khoá vào file config.
                    </p>
                  </div>
                </div>

                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Input
                    readOnly
                    value={revealed.key}
                    aria-label={`Khoá ops cho ${revealed.name}`}
                    className="min-h-11 flex-1 border-amber-300/80 bg-background font-mono text-xs sm:min-h-9 sm:text-sm font-semibold selection:bg-amber-200"
                    onFocus={(e) => e.currentTarget.select()}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    className="min-h-11 shrink-0 gap-1.5 border-amber-400/80 bg-background hover:bg-amber-100/50 sm:min-h-9"
                    onClick={handleCopyKey}
                  >
                    {copiedKey ? (
                      <Check className="size-4 text-emerald-700" aria-hidden="true" />
                    ) : (
                      <Copy className="size-4" aria-hidden="true" />
                    )}
                    <span>{copiedKey ? 'Đã sao chép' : 'Sao chép'}</span>
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="min-h-11 text-xs text-amber-900 hover:bg-amber-200/50 dark:text-amber-200 sm:min-h-9"
                    onClick={() => setRevealed(null)}
                  >
                    Đã lưu khoá (Đóng)
                  </Button>
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </section>

      {/* 2. Danh sách khoá ops */}
      <section id="ops-keys-list" aria-labelledby="ops-keys-list-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-muted-foreground/20 bg-muted text-foreground">
                  <FileKey2 className="size-4" aria-hidden="true" />
                </div>
                <div>
                  <CardTitle id="ops-keys-list-title" className="text-sm sm:text-base font-semibold text-foreground">
                    Danh sách khoá ops
                  </CardTitle>
                  <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                    Theo dõi phạm vi (hiệu dụng) và lần dùng gần nhất. Cấp ghi tự hồi sau cửa sổ; thu hồi để chặn ngay.
                  </CardDescription>
                </div>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => void refetch()}
                disabled={isLoading || isFetching}
                className="min-h-11 sm:min-h-9 gap-1.5 self-start text-xs text-muted-foreground hover:text-foreground sm:self-auto"
                aria-label="Tải lại danh sách khoá ops"
              >
                <RefreshCw
                  className={cn('size-3.5', (isLoading || isFetching) && 'animate-spin')}
                  aria-hidden="true"
                />
                <span>Tải lại</span>
              </Button>
            </div>
          </CardHeader>

          <CardContent className="p-0">
            {isLoading ? (
              <div className="space-y-4 p-5">
                <Skeleton className="h-12 w-full rounded-lg" />
                <Skeleton className="h-12 w-full rounded-lg" />
              </div>
            ) : isError ? (
              <div className="flex flex-col items-center justify-center gap-3 p-8 text-center text-sm text-muted-foreground">
                <ShieldAlert className="size-8 text-destructive" aria-hidden="true" />
                <p className="font-medium text-foreground">Không thể tải danh sách khoá ops</p>
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
                  <FileKey2 className="size-6" aria-hidden="true" />
                </div>
                <p className="font-medium text-foreground">Chưa có khoá ops nào</p>
                <p className="text-xs text-muted-foreground max-w-sm">
                  Tạo khoá ops đầu tiên ở khung phía trên, rồi làm theo hướng dẫn cài đặt bên dưới.
                </p>
              </div>
            ) : (
              <ul className="divide-y divide-border/60">
                {keys.map((apiKey) => (
                  <OpsKeyRow
                    key={apiKey.id}
                    apiKey={apiKey}
                    onToggleScope={handleToggleScope}
                    onRevoke={handleRevoke}
                    toggling={togglingId === apiKey.id}
                    revoking={revokingId === apiKey.id}
                  />
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </section>

      {/* 3. Hướng dẫn cài đặt MCP vào agent */}
      <section id="ops-mcp-install" aria-labelledby="ops-mcp-install-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-violet-500/20 bg-violet-500/10 text-violet-700 dark:text-violet-400">
                <Terminal className="size-4" aria-hidden="true" />
              </div>
              <div>
                <CardTitle id="ops-mcp-install-title" className="text-sm sm:text-base font-semibold text-foreground">
                  Cài đặt MCP vào agent của bạn
                </CardTitle>
                <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                  Kết nối payroll-ops-mcp (stdio) vào Hermes, Paseo, Claude Desktop hoặc Claude Code — agent sẽ gọi
                  được API payroll với đúng phạm vi khoá.
                </CardDescription>
              </div>
            </div>
          </CardHeader>

          <CardContent className="p-4 sm:p-5 space-y-6 text-xs sm:text-sm">
            <ol className="space-y-5">
              {/* Bước 1 */}
              <li className="space-y-2">
                <p className="font-medium text-foreground">
                  <span className="mr-1.5 inline-flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary">1</span>
                  Tạo khoá <span className="font-mono">pk_ops_*</span> ở khung phía trên
                </p>
                <p className="text-muted-foreground">
                  Mặc định <span className="font-medium">read_only</span> (chỉ GET). Khoá chạy với quyền quản trị của
                  bạn nhưng bị giới hạn bởi phạm vi — mọi lượt gọi đều được ghi audit theo tên agent.
                </p>
              </li>

              {/* Bước 2 */}
              <li className="space-y-2">
                <p className="font-medium text-foreground">
                  <span className="mr-1.5 inline-flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary">2</span>
                  Lưu khoá vào file bí mật (chế độ 600)
                </p>
                <p className="text-muted-foreground">
                  Trên máy chạy agent, lưu khoá vào file riêng và khoá quyền truy cập. Server MCP từ chối file đọc
                  được bởi group/other.
                </p>
                {snippetBlock('secrets-file', secretsFileSnippet)}
              </li>

              {/* Bước 3 */}
              <li className="space-y-2">
                <p className="font-medium text-foreground">
                  <span className="mr-1.5 inline-flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary">3</span>
                  Cấu hình MCP client
                </p>
                <p className="text-muted-foreground">
                  Claude Desktop, Hermes, Paseo runner dùng dạng <span className="font-mono">mcpServers</span> sau —
                  chỉ tham chiếu ĐƯỜNG DẪN file bí mật, không bao giờ dán khoá vào config:
                </p>
                {snippetBlock('mcp-config', mcpConfigSnippet)}
                <p className="text-muted-foreground">Với Claude Code dùng lệnh:</p>
                {snippetBlock('claude-code', claudeCodeSnippet)}
              </li>

              {/* Bước 4 */}
              <li className="space-y-2">
                <p className="font-medium text-foreground">
                  <span className="mr-1.5 inline-flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary">4</span>
                  Phạm vi khoá: chỉ đọc theo mặc định
                </p>
                <p className="text-muted-foreground">
                  Giữ <span className="font-medium">read_only</span> trừ khi agent thực sự cần ghi. Muốn ghi: bấm
                  «Cấp ghi (8h)» trong danh sách (yêu cầu phiên đã xác thực OTP) — khoá tự hồi về chỉ đọc sau cửa sổ
                  ghi, và việc đổi phạm vi áp dụng ngay ở lượt gọi tiếp theo.
                </p>
              </li>

              {/* Bước 5 */}
              <li className="space-y-2">
                <p className="font-medium text-foreground">
                  <span className="mr-1.5 inline-flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary">5</span>
                  Kill switch (tắt toàn bộ kênh khi cần)
                </p>
                <p className="text-muted-foreground">
                  Đặt biến môi trường sau ở backend (restart để áp dụng) và/hoặc ở server MCP — mọi lượt gọi của
                  khoá ops sẽ thất bại đóng ngay lập tức:
                </p>
                {snippetBlock('kill-switch', killSwitchSnippet)}
              </li>
            </ol>

            <div className="rounded-lg border border-border/80 bg-muted/30 p-3 text-[11px] text-muted-foreground">
              Chi tiết đầy đủ: <span className="font-mono">payroll-ops-mcp/README.md</span> và{' '}
              <span className="font-mono">docs/runbooks/payroll-ops-mcp.md</span>. Kênh này tách biệt hoàn toàn với
              khoá MCP chatbot phía trên — khác namespace, khác phạm vi, không dùng chung khoá.
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
};
