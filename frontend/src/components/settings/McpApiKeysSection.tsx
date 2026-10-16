import { type FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import {
  AlertTriangle,
  Bot,
  Check,
  Copy,
  KeyRound,
  Loader2,
  Lock,
  Plus,
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
  useMcpApiKeys,
  useCreateMCPAPIKey,
  useRevokeMCPAPIKey,
  useSetMCPAPIKeyScope,
} from '@/hooks/api/useMcpApiKeys';
import type { MCPAPIKey, MCPAPIKeyScope } from '@/services/api/mcpApiKeys.service';

const DATE_TIME_DISPLAY = 'dd/MM/yyyy HH:mm';

const formatDateTime = (value: string | null): string =>
  value ? format(new Date(value), DATE_TIME_DISPLAY) : '—';

/** Absolute endpoint an agent connects to (Streamable HTTP). */
const MCP_ENDPOINT = `${API_CONFIG.baseURL || (typeof window !== 'undefined' ? window.location.origin : '')}${API_CONFIG.apiV1}/mcp`;

const SCOPE_OPTIONS: ReadonlyArray<{
  value: MCPAPIKeyScope;
  label: string;
  description: string;
}> = [
  {
    value: 'read_only',
    label: 'Chỉ đọc (READ ONLY)',
    description: 'Chỉ các GET — không thể thay đổi dữ liệu',
  },
  {
    value: 'full',
    label: 'Toàn quyền (ALL)',
    description: 'Tất cả endpoint, kể cả tạo/sửa/xoá',
  },
];

const scopeBadgeClass = (scope: MCPAPIKeyScope): string =>
  scope === 'full'
    ? 'border-destructive/30 bg-destructive/10 px-2 py-0.5 text-xs font-medium text-destructive'
    : 'border-border bg-muted/60 px-2 py-0.5 text-xs font-medium text-muted-foreground';

const scopeLabel = (scope: MCPAPIKeyScope): string =>
  scope === 'full' ? 'ALL' : 'READ ONLY';

interface McpKeyRowProps {
  apiKey: MCPAPIKey;
  onToggleScope: (apiKey: MCPAPIKey) => void;
  onRevoke: (apiKey: MCPAPIKey) => void;
  toggling: boolean;
  revoking: boolean;
}

const McpKeyRow = ({ apiKey, onToggleScope, onRevoke, toggling, revoking }: McpKeyRowProps) => {
  const revoked = apiKey.revoked_at != null;

  return (
    <li className="flex flex-col gap-3 p-4 transition-colors hover:bg-muted/30 sm:flex-row sm:items-center sm:justify-between sm:p-5">
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="break-words text-sm font-semibold text-foreground sm:text-base">
            {apiKey.name}
          </span>
          <Badge variant="outline" className={scopeBadgeClass(apiKey.scope)}>
            {scopeLabel(apiKey.scope)}
          </Badge>
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
          {apiKey.scope_updated_at ? (
            <span>Đổi phạm vi lần cuối {formatDateTime(apiKey.scope_updated_at)}</span>
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
            <span>{apiKey.scope === 'full' ? 'Về chỉ đọc' : 'Cấp ghi (ALL)'}</span>
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

/** Admin control surface for MCP agent API keys + the agent integration guide. */
export const McpApiKeysSection = () => {
  const { data, isLoading, isError, isFetching, refetch } = useMcpApiKeys();
  const createKey = useCreateMCPAPIKey();
  const revokeKey = useRevokeMCPAPIKey();
  const setScopeMutation = useSetMCPAPIKeyScope();

  const [name, setName] = useState('');
  const [scope, setScope] = useState<MCPAPIKeyScope | null>(null);
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
      toast.error('Nhập tên khoá MCP');
      return;
    }
    if (!scope) {
      toast.error('Chọn phạm vi truy cập cho khoá MCP');
      return;
    }
    try {
      const res = await createKey.mutateAsync({ name: trimmed, scope });
      if (res.data?.key) {
        setRevealed({ key: res.data.key, name: res.data.name });
      }
      setName('');
      setScope(null);
    } catch {
      // Failure toast is emitted by the hook.
    }
  };

  const handleCopyKey = async () => {
    if (!revealed) return;
    try {
      await navigator.clipboard.writeText(revealed.key);
      setCopiedKey(true);
      toast.success('Đã sao chép khoá MCP vào clipboard');
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

  const handleToggleScope = async (apiKey: MCPAPIKey) => {
    const toFull = apiKey.scope !== 'full';
    const confirmed = window.confirm(
      toFull
        ? `Chuyển khoá MCP "${apiKey.name}" sang ALL? Khoá sẽ được phép ghi dữ liệu với quyền quản trị của người tạo khoá. Yêu cầu phiên đã xác thực OTP.`
        : `Chuyển khoá MCP "${apiKey.name}" về chỉ đọc (READ ONLY)? Agent sẽ bị chặn mọi thao tác ghi ở lượt gọi tiếp theo.`,
    );
    if (!confirmed) return;
    setTogglingId(apiKey.id);
    try {
      await setScopeMutation.mutateAsync({
        id: apiKey.id,
        payload: { scope: toFull ? 'full' : 'read_only' },
      });
    } catch {
      // Failure toast is emitted by the hook (incl. the OTP 403 case).
    } finally {
      setTogglingId(null);
    }
  };

  const handleRevoke = async (apiKey: MCPAPIKey) => {
    if (!window.confirm(`Thu hồi khoá MCP "${apiKey.name}" (${scopeLabel(apiKey.scope)})? Agent đang dùng khoá này sẽ mất quyền truy cập ngay.`)) {
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

  const claudeSnippet = `claude mcp add --transport http payroll ${MCP_ENDPOINT} --header "X-API-Key: <key>"`;
  const mcpJsonSnippet = JSON.stringify(
    {
      mcpServers: {
        payroll: {
          type: 'http',
          url: MCP_ENDPOINT,
          headers: { 'X-API-Key': '<key>' },
        },
      },
    },
    null,
    2,
  );
  const payrollApiExample = '{"method": "GET", "path": "/api/v1/employees", "query": "page=1&page_size=20"}';

  return (
    <div className="space-y-6">
      {/* 1. Tạo khoá MCP */}
      <section id="mcp-keys-create" aria-labelledby="mcp-keys-create-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-primary/20 bg-primary/10 text-primary">
                <Plus className="size-4" aria-hidden="true" />
              </div>
              <div className="min-w-0">
                <CardTitle id="mcp-keys-create-title" className="text-sm sm:text-base font-semibold text-foreground">
                  Tạo khoá API MCP
                </CardTitle>
                <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                  Cấp khoá cho agent (Claude Code, MCP client…) truy cập API. Khoá chỉ hiển thị một lần khi tạo.
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="p-4 sm:p-5 space-y-4">
            <form onSubmit={handleCreate} className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-[1fr_auto] sm:items-end">
                <div className="space-y-1.5">
                  <Label htmlFor="mcp-key-name" className="text-xs sm:text-sm font-medium text-foreground">
                    Tên khoá <span className="text-destructive">*</span>
                  </Label>
                  <Input
                    id="mcp-key-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Ví dụ: Claude Code trên máy chủ báo cáo"
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
                  <span>Tạo khoá MCP</span>
                </Button>
              </div>

              <fieldset className="space-y-1.5">
                <legend className="text-xs sm:text-sm font-medium text-foreground">
                  Phạm vi truy cập <span className="text-destructive">*</span>
                </legend>
                <div className="grid gap-2 sm:grid-cols-2">
                  {SCOPE_OPTIONS.map((option) => {
                    const inputId = `mcp-key-scope-${option.value}`;
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
                          name="mcp-key-scope"
                          value={option.value}
                          checked={scope === option.value}
                          onChange={() => setScope(option.value)}
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
                      Khoá được cấp cho <span className="font-medium underline">{revealed.name}</span>. Bạn sẽ không thể xem lại khoá sau khi đóng thông báo này.
                    </p>
                  </div>
                </div>

                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                  <Input
                    readOnly
                    value={revealed.key}
                    aria-label={`Khoá MCP cho ${revealed.name}`}
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

      {/* 2. Danh sách khoá MCP */}
      <section id="mcp-keys-list" aria-labelledby="mcp-keys-list-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-muted-foreground/20 bg-muted text-foreground">
                  <KeyRound className="size-4" aria-hidden="true" />
                </div>
                <div>
                  <CardTitle id="mcp-keys-list-title" className="text-sm sm:text-base font-semibold text-foreground">
                    Danh sách khoá MCP
                  </CardTitle>
                  <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                    Theo dõi khoá, phạm vi và lần dùng gần nhất. Thu hồi để chặn một khoá ngay lập tức.
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
                aria-label="Tải lại danh sách khoá MCP"
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
                <p className="font-medium text-foreground">Không thể tải danh sách khoá MCP</p>
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
                <p className="font-medium text-foreground">Chưa có khoá MCP nào</p>
                <p className="text-xs text-muted-foreground max-w-sm">
                  Tạo khoá MCP đầu tiên ở khung phía trên để cấp quyền truy cập cho agent.
                </p>
              </div>
            ) : (
              <ul className="divide-y divide-border/60">
                {keys.map((apiKey) => (
                  <McpKeyRow
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

      {/* 3. Hướng dẫn tích hợp MCP */}
      <section id="mcp-guide" aria-labelledby="mcp-guide-title">
        <Card className="overflow-hidden border-border shadow-sm">
          <CardHeader className="border-b bg-muted/20 p-4 sm:p-5">
            <div className="flex items-center gap-3">
              <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-violet-500/20 bg-violet-500/10 text-violet-700 dark:text-violet-400">
                <Bot className="size-4" aria-hidden="true" />
              </div>
              <div>
                <CardTitle id="mcp-guide-title" className="text-sm sm:text-base font-semibold text-foreground">
                  Hướng dẫn tích hợp MCP
                </CardTitle>
                <CardDescription className="text-xs sm:text-sm text-muted-foreground mt-0.5">
                  Kết nối agent qua Model Context Protocol (Streamable HTTP) để gọi API payroll.
                </CardDescription>
              </div>
            </div>
          </CardHeader>

          <CardContent className="p-4 sm:p-5 space-y-6 text-xs sm:text-sm">
            {/* Endpoint */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">Endpoint (Streamable HTTP)</p>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <div className="flex flex-1 items-center gap-2 rounded-lg border border-border/80 bg-muted/50 px-3 py-2">
                  <Terminal className="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
                  <code className="min-w-0 flex-1 break-all font-mono text-xs leading-5 text-foreground">
                    {MCP_ENDPOINT}
                  </code>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="min-h-11 sm:min-h-9 shrink-0 gap-1.5 text-xs"
                  onClick={() => void handleCopySnippet('endpoint', MCP_ENDPOINT)}
                >
                  {copiedSnippet === 'endpoint' ? (
                    <Check className="size-3.5 text-emerald-700" aria-hidden="true" />
                  ) : (
                    <Copy className="size-3.5" aria-hidden="true" />
                  )}
                  <span>{copiedSnippet === 'endpoint' ? 'Đã sao chép' : 'Sao chép'}</span>
                </Button>
              </div>
            </div>

            {/* Scope semantics */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">Phạm vi khoá</p>
              <div className="overflow-hidden rounded-lg border border-border/80">
                <table className="w-full text-left text-xs">
                  <thead className="bg-muted/40 text-muted-foreground">
                    <tr>
                      <th className="px-3 py-2 font-medium">Phạm vi</th>
                      <th className="px-3 py-2 font-medium">Quyền</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border/60">
                    <tr>
                      <td className="px-3 py-2 font-medium text-foreground">READ ONLY</td>
                      <td className="px-3 py-2 text-muted-foreground">Chỉ các GET — không thể thay đổi dữ liệu</td>
                    </tr>
                    <tr>
                      <td className="px-3 py-2 font-medium text-foreground">ALL</td>
                      <td className="px-3 py-2 text-muted-foreground">Tất cả endpoint, kể cả tạo/sửa/xoá</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <p className="text-[11px] text-muted-foreground">
                Lưu ý: khoá hoạt động với quyền quản trị của người tạo khoá.
              </p>
            </div>

            {/* Tools */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">Tools</p>
              <ul className="space-y-1.5 text-muted-foreground">
                <li>
                  <code className="font-mono text-xs bg-muted px-1 py-0.5 rounded text-foreground">payroll_api</code>
                  {' — gọi một endpoint REST bất kỳ. Params: '}
                  <code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">method</code>,{' '}
                  <code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">path</code>,{' '}
                  <code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">query</code>,{' '}
                  <code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">body</code>.
                </li>
                <li>
                  <code className="font-mono text-xs bg-muted px-1 py-0.5 rounded text-foreground">payroll_endpoints</code>
                  {' — liệt kê mọi endpoint mà khoá có thể truy cập.'}
                </li>
              </ul>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <pre className="flex-1 overflow-x-auto rounded-lg border border-border/80 bg-muted/50 p-3 font-mono text-[11px] leading-relaxed text-foreground">
                  {payrollApiExample}
                </pre>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="min-h-11 sm:min-h-9 shrink-0 gap-1.5 text-xs"
                  onClick={() => void handleCopySnippet('payroll-api', payrollApiExample)}
                >
                  {copiedSnippet === 'payroll-api' ? (
                    <Check className="size-3.5 text-emerald-700" aria-hidden="true" />
                  ) : (
                    <Copy className="size-3.5" aria-hidden="true" />
                  )}
                  <span>{copiedSnippet === 'payroll-api' ? 'Đã sao chép' : 'Sao chép'}</span>
                </Button>
              </div>
            </div>

            {/* Claude Code snippet */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">Claude Code</p>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                <pre className="flex-1 overflow-x-auto rounded-lg border border-border/80 bg-muted/50 p-3 font-mono text-[11px] leading-relaxed text-foreground">
                  {claudeSnippet}
                </pre>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="min-h-11 sm:min-h-9 shrink-0 gap-1.5 text-xs"
                  onClick={() => void handleCopySnippet('claude', claudeSnippet)}
                >
                  {copiedSnippet === 'claude' ? (
                    <Check className="size-3.5 text-emerald-700" aria-hidden="true" />
                  ) : (
                    <Copy className="size-3.5" aria-hidden="true" />
                  )}
                  <span>{copiedSnippet === 'claude' ? 'Đã sao chép' : 'Sao chép'}</span>
                </Button>
              </div>
            </div>

            {/* Generic mcp.json */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">mcp.json (MCP client bất kỳ)</p>
              <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
                <pre className="flex-1 overflow-x-auto rounded-lg border border-border/80 bg-muted/50 p-3 font-mono text-[11px] leading-relaxed text-foreground">
                  {mcpJsonSnippet}
                </pre>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="min-h-11 sm:min-h-9 shrink-0 gap-1.5 text-xs"
                  onClick={() => void handleCopySnippet('mcp-json', mcpJsonSnippet)}
                >
                  {copiedSnippet === 'mcp-json' ? (
                    <Check className="size-3.5 text-emerald-700" aria-hidden="true" />
                  ) : (
                    <Copy className="size-3.5" aria-hidden="true" />
                  )}
                  <span>{copiedSnippet === 'mcp-json' ? 'Đã sao chép' : 'Sao chép'}</span>
                </Button>
              </div>
            </div>

            {/* Recipes */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">Các truy vấn thường dùng</p>
              <ul className="space-y-1 text-muted-foreground">
                <li><code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">GET /api/v1/employees</code> — danh sách nhân viên</li>
                <li><code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">GET /api/v1/employees/{'{id}'}</code> — chi tiết nhân viên</li>
                <li><code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">GET /api/v1/projects</code> — danh sách dự án</li>
                <li><code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">GET /api/v1/projects/summary</code> — tổng quan dự án</li>
                <li><code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">GET /api/v1/timesheets</code> — bảng chấm công</li>
                <li><code className="font-mono text-[11px] bg-muted px-1 py-0.5 rounded text-foreground">GET /api/v1/projects/{'{id}'}/timesheets</code> — chấm công theo dự án</li>
              </ul>
              <p className="text-[11px] text-muted-foreground">
                Các tham số lọc/phân trang theo đúng REST API (dùng trường <code className="font-mono">query</code>).
              </p>
            </div>

            {/* Security notes */}
            <div className="space-y-2">
              <p className="font-medium text-foreground">Lưu ý bảo mật</p>
              <ul className="list-disc space-y-1 pl-5 text-muted-foreground">
                <li>Khoá chỉ hiển thị một lần khi tạo — lưu ở nơi an toàn.</li>
                <li>Ưu tiên dùng READ ONLY; chỉ dùng ALL khi agent thực sự cần ghi dữ liệu.</li>
                <li>
                  Có thể đổi phạm vi (READ ONLY ↔ ALL) hoặc thu hồi bất cứ lúc nào — thay đổi
                  áp dụng ngay ở lượt gọi tiếp theo; hệ thống ghi nhận lần đổi phạm vi và lần dùng
                  gần nhất của khoá.
                </li>
                <li>
                  Các khu vực không được phép truy cập qua MCP (đăng nhập, quản trị khoá API, webhook,
                  {' '}/api/v1/vendor) sẽ bị từ chối theo thiết kế.
                </li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
};
