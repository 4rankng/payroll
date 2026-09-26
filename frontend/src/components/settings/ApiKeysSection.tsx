import { type FormEvent, type ReactNode, useState } from 'react';
import { toast } from 'sonner';
import { format } from 'date-fns';
import { AlertTriangle, Check, Copy, Loader2, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useApiKeys, useCreateAPIKey, useRevokeAPIKey } from '@/hooks/api/useApiKeys';
import type { APIKey } from '@/services/api/apiKeys.service';

const DATE_TIME_DISPLAY = 'dd/MM/yyyy HH:mm';

type SettingsSectionProps = {
  id: string;
  title: string;
  description: string;
  children: ReactNode;
};

const SettingsSection = ({ id, title, description, children }: SettingsSectionProps) => {
  const titleId = `${id}-title`;
  return (
    <section
      id={id}
      aria-labelledby={titleId}
      className="grid gap-5 border-t px-4 py-6 sm:px-6 lg:grid-cols-[14rem_minmax(0,1fr)] lg:gap-8 lg:py-7 xl:grid-cols-[16rem_minmax(0,1fr)]"
    >
      <div className="min-w-0">
        <h3 id={titleId} className="text-sm font-semibold text-foreground">
          {title}
        </h3>
        <p className="mt-2 max-w-xs text-sm leading-5 text-muted-foreground">{description}</p>
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
};

const formatDateTime = (value: string | null): string =>
  value ? format(new Date(value), DATE_TIME_DISPLAY) : '—';

const APIKeyRow = ({
  apiKey,
  onRevoke,
  revoking,
}: {
  apiKey: APIKey;
  onRevoke: (apiKey: APIKey) => void;
  revoking: boolean;
}) => {
  const revoked = apiKey.revoked_at != null;
  return (
    <li className="flex flex-col gap-3 border-t py-4 first:border-t-0 first:pt-0 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="break-words text-sm font-medium text-foreground">{apiKey.name}</span>
          {revoked ? (
            <Badge variant="outline" className="text-muted-foreground">
              Đã thu hồi
            </Badge>
          ) : (
            <Badge variant="success">Hoạt động</Badge>
          )}
        </div>
        <p className="break-all font-mono text-xs text-muted-foreground">{apiKey.key_prefix}…</p>
        <p className="text-xs text-muted-foreground">
          Tạo {formatDateTime(apiKey.created_at)} · Dùng lần cuối {apiKey.last_used_at ? formatDateTime(apiKey.last_used_at) : 'Chưa dùng'}
        </p>
      </div>
      {!revoked ? (
        <Button
          type="button"
          variant="outline"
          className="min-h-11 shrink-0 text-destructive sm:min-h-9"
          onClick={() => onRevoke(apiKey)}
          disabled={revoking}
        >
          {revoking ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
          Thu hồi
        </Button>
      ) : null}
    </li>
  );
};

/** Admin control surface for machine API keys used by the chatbot integration. */
export const ApiKeysSection = () => {
  const { data, isLoading, isError, refetch } = useApiKeys();
  const createKey = useCreateAPIKey();
  const revokeKey = useRevokeAPIKey();

  const [name, setName] = useState('');
  const [revealed, setRevealed] = useState<{ key: string; name: string } | null>(null);
  const [copied, setCopied] = useState(false);
  const [revokingId, setRevokingId] = useState<number | null>(null);

  const keys = data?.data ?? [];

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
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error('Không thể sao chép khoá');
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

  return (
    <div className="space-y-0">
      <SettingsSection
        id="api-keys-create"
        title="Tạo khoá API"
        description="Cấp một khoá cho hệ thống bên ngoài (ví dụ chatbot Zalo) gọi API đặt lại mật khẩu. Khoá chỉ hiển thị một lần khi tạo."
      >
        <form onSubmit={handleCreate} className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="api-key-name" className="block text-sm font-medium">
              Tên khoá
            </Label>
            <Input
              id="api-key-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Zalo Chatbot"
              maxLength={100}
              className="min-h-11 sm:min-h-9"
            />
          </div>
          <Button type="submit" className="min-h-11 sm:min-h-9" disabled={createKey.isPending}>
            {createKey.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Tạo khoá API
          </Button>
        </form>

        {revealed ? (
          <div
            role="alert"
            className="mt-4 space-y-2 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm dark:border-amber-700/60 dark:bg-amber-950/40"
          >
            <p className="flex items-center gap-2 font-medium text-amber-900 dark:text-amber-200">
              <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
              Khoá chỉ hiển thị một lần. Hãy sao chép ngay.
            </p>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Input
                readOnly
                value={revealed.key}
                aria-label={`Khoá API cho ${revealed.name}`}
                className="min-h-11 flex-1 font-mono text-xs sm:min-h-9"
                onFocus={(e) => e.currentTarget.select()}
              />
              <Button
                type="button"
                variant="outline"
                className="min-h-11 shrink-0 sm:min-h-9"
                onClick={handleCopy}
              >
                {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                {copied ? 'Đã sao chép' : 'Sao chép'}
              </Button>
            </div>
          </div>
        ) : null}
      </SettingsSection>

      <SettingsSection
        id="api-keys-list"
        title="Khoá đang có"
        description="Danh sách khoá API đã cấp. Thu hồi để chặn một khoá ngay lập tức."
      >
        {isLoading ? (
          <p className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" /> Đang tải…
          </p>
        ) : isError ? (
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <span>Không thể tải danh sách khoá.</span>
            <Button type="button" variant="outline" className="min-h-11 sm:min-h-9" onClick={() => void refetch()}>
              Thử lại
            </Button>
          </div>
        ) : keys.length === 0 ? (
          <p className="text-sm text-muted-foreground">Chưa có khoá API nào.</p>
        ) : (
          <ul className="space-y-0">
            {keys.map((apiKey) => (
              <APIKeyRow
                key={apiKey.id}
                apiKey={apiKey}
                onRevoke={handleRevoke}
                revoking={revokingId === apiKey.id}
              />
            ))}
          </ul>
        )}
      </SettingsSection>
    </div>
  );
};
