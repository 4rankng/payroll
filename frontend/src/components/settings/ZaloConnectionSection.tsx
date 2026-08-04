import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';
import { MessageCircle, CheckCircle2, XCircle, AlertCircle, Loader2, Copy, RefreshCw, Power, Link2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
  useZaloStatus,
  useSaveZaloCredentials,
  useStartZaloOAuth,
  useCompleteZaloOAuth,
  useSetZaloEnabled,
  useRefreshZaloToken,
} from '@/hooks/api/useZaloConnection';

/**
 * ZaloConnectionSection — the admin control surface for the Zalo ZNS connection.
 *
 * Renders the live connection status badge, the credentials form (app_id /
 * secret_key / template_id), the "Kết nối Zalo" OAuth button, the runtime
 * enable/disable toggle, and a manual token-refresh button. All mutations
 * invalidate the status query on success so the badge updates immediately.
 *
 * Secrets are write-only: the server never returns secret_key, access_token, or
 * refresh_token in the status payload, so those fields are never populated from
 * the server (the secret_key input is empty by default — "leave blank to keep
 * existing").
 */
export const ZaloConnectionSection = () => {
  const { data: statusRes, isLoading, isError } = useZaloStatus();
  const status = statusRes?.data;
  const [searchParams, setSearchParams] = useSearchParams();

  const saveCreds = useSaveZaloCredentials();
  const startOAuth = useStartZaloOAuth();
  const completeOAuth = useCompleteZaloOAuth();
  const setEnabled = useSetZaloEnabled();
  const refreshTok = useRefreshZaloToken();

  // --- Handle Zalo OAuth redirect ---
  // Zalo redirects back to the SPA (e.g. /admin/settings?tab=zalo&code=...&state=...).
  // The SPA extracts code+state and POSTs to the backend (which validates the
  // single-use state and exchanges the code). This runs once on mount when the
  // params are present.
  useEffect(() => {
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    if (code && state) {
      completeOAuth.mutate(
        { code, state },
        {
          onSuccess: () => {
            toast.success('Đã kết nối Zalo thành công');
            // Clean the URL so a refresh doesn't re-trigger.
            const next = new URLSearchParams(searchParams);
            next.delete('code');
            next.delete('state');
            setSearchParams(next, { replace: true });
          },
          onError: (err: unknown) => {
            const msg = err instanceof Error ? err.message : 'Không rõ lỗi';
            toast.error(`Kết nối Zalo thất bại: ${msg}`);
            const next = new URLSearchParams(searchParams);
            next.delete('code');
            next.delete('state');
            setSearchParams(next, { replace: true });
          },
        },
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  // Form state — secret_key is write-only (never echoed from server).
  const [appID, setAppID] = useState('');
  const [secretKey, setSecretKey] = useState('');
  const [templateID, setTemplateID] = useState('617976');
  const [confirmDisable, setConfirmDisable] = useState(false);

  // Populate app_id + template_id from status once loaded.
  useEffect(() => {
    if (status) {
      setAppID((prev) => prev || status.configured ? appID : '');
      setTemplateID(status.template_id || '617976');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status?.template_id, status?.configured]);

  // Handle ?zalo_connected=1 / ?zalo_error= from the OAuth callback redirect.
  useEffect(() => {
    const connected = searchParams.get('zalo_connected');
    const error = searchParams.get('zalo_error');
    if (connected) {
      toast.success('Đã kết nối Zalo thành công');
      const next = new URLSearchParams(searchParams);
      next.delete('zalo_connected');
      setSearchParams(next, { replace: true });
    }
    if (error) {
      toast.error(`Kết nối Zalo thất bại: ${error}`);
      const next = new URLSearchParams(searchParams);
      next.delete('zalo_error');
      setSearchParams(next, { replace: true });
    }
  }, [searchParams, setSearchParams, toast]);

  const handleSaveCreds = async () => {
    try {
      await saveCreds.mutateAsync({ app_id: appID, secret_key: secretKey, template_id: templateID });
      toast.success('Đã lưu thông tin kết nối Zalo');
      setSecretKey(''); // clear the write-only field after save
    } catch (e) {
      toast.error('Không thể lưu thông tin kết nối');
    }
  };

  const handleConnect = async () => {
    try {
      const res = await startOAuth.mutateAsync();
      const url = res.data?.redirect_url;
      if (url) {
        window.location.href = url; // full redirect to Zalo permission page
      }
    } catch (e) {
      toast.error('Không thể bắt đầu kết nối Zalo');
    }
  };

  const handleToggle = async (enabled: boolean) => {
    if (enabled && !status?.connected) {
      toast.error('Kết nối Zalo trước khi bật tính năng');
      return;
    }
    if (!enabled && !confirmDisable) {
      setConfirmDisable(true);
      return;
    }
    setConfirmDisable(false);
    try {
      await setEnabled.mutateAsync(enabled);
      toast.success(enabled ? 'Đã bật Zalo OTP' : 'Đã tắt Zalo OTP');
    } catch (e) {
      toast.error('Không thể cập nhật trạng thái');
    }
  };

  const handleRefresh = async () => {
    try {
      await refreshTok.mutateAsync();
      toast.success('Đã làm mới token Zalo');
    } catch (e) {
      toast.error('Không thể làm mới token');
    }
  };

  const handleCopyCallback = () => {
    if (status?.callback_url) {
      navigator.clipboard.writeText(status.callback_url);
      toast.success('Đã sao chép URL callback');
    }
  };

  // --- loading / error ---
  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-10 text-center text-sm text-muted-foreground">
          <Loader2 className="mx-auto mb-3 h-6 w-6 animate-spin" />
          Đang tải trạng thái kết nối...
        </CardContent>
      </Card>
    );
  }
  if (isError) {
    return (
      <Card>
        <CardContent className="py-10 text-center text-sm text-destructive">
          <AlertCircle className="mx-auto mb-3 h-6 w-6" />
          Không thể tải trạng thái kết nối Zalo.
        </CardContent>
      </Card>
    );
  }

  // --- status badge ---
  const badge = (() => {
    if (!status?.configured) return { icon: XCircle, text: 'Chưa cấu hình', color: 'text-muted-foreground', bg: 'bg-muted' };
    if (status.last_error) return { icon: AlertCircle, text: 'Lỗi', color: 'text-destructive', bg: 'bg-destructive/10' };
    if (!status.connected) return { icon: AlertCircle, text: 'Chưa kết nối', color: 'text-yellow-600', bg: 'bg-yellow-500/10' };
    if (!status.enabled) return { icon: CheckCircle2, text: 'Đã kết nối (đang tắt)', color: 'text-yellow-600', bg: 'bg-yellow-500/10' };
    return { icon: CheckCircle2, text: 'Đang hoạt động', color: 'text-green-600', bg: 'bg-green-500/10' };
  })();
  const BadgeIcon = badge.icon;

  // Expiry countdown (rough).
  const expiryText = (() => {
    if (!status?.expires_at) return null;
    const ms = new Date(status.expires_at).getTime() - Date.now();
    if (ms <= 0) return 'Đã hết hạn';
    const h = Math.floor(ms / 3_600_000);
    const m = Math.floor((ms % 3_600_000) / 60_000);
    return h > 0 ? `còn ${h}h ${m}m` : `còn ${m}m`;
  })();

  return (
    <div className="space-y-5">
      {/* Status card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <MessageCircle className="h-4 w-4 text-blue-500" />
            Zalo ZNS — Đặt lại mật khẩu qua OTP
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <span className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-bold ${badge.bg} ${badge.color}`}>
              <BadgeIcon className="h-3.5 w-3.5" /> {badge.text}
            </span>
            {expiryText && (
              <span className="text-xs text-muted-foreground">Token: {expiryText}</span>
            )}
            <span className="text-xs text-muted-foreground">
              Template: <code className="font-mono">{status?.template_id || '617976'}</code>
            </span>
          </div>
          {status?.last_error && (
            <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-xs text-destructive">
              <strong>Lỗi gần nhất:</strong> {status.last_error}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Credentials form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Thông tin kết nối</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="zalo-app-id">App ID</Label>
            <Input
              id="zalo-app-id"
              value={appID}
              onChange={(e) => setAppID(e.target.value)}
              placeholder="App ID từ Zalo OA Console"
              disabled={saveCreds.isPending}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="zalo-secret">
              Secret Key
              <span className="ml-2 text-xs font-normal text-muted-foreground">(để trống để giữ nguyên)</span>
            </Label>
            <Input
              id="zalo-secret"
              type="password"
              value={secretKey}
              onChange={(e) => setSecretKey(e.target.value)}
              placeholder="••••••••••••••••"
              disabled={saveCreds.isPending}
              autoComplete="new-password"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="zalo-template">Template ID</Label>
            <Input
              id="zalo-template"
              value={templateID}
              onChange={(e) => setTemplateID(e.target.value)}
              placeholder="617976"
              disabled={saveCreds.isPending}
            />
          </div>
          <Button onClick={handleSaveCreds} disabled={saveCreds.isPending || !appID.trim()}>
            {saveCreds.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Lưu thông tin
          </Button>

          {/* Callback URL */}
          <div className="space-y-2 rounded-lg border bg-muted/30 p-3">
            <Label className="text-xs text-muted-foreground">OAuth Callback URL</Label>
            <p className="text-xs text-muted-foreground">
              Đăng ký URL này trong Zalo OA Console (OAuth redirect_uris).
            </p>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded bg-background px-2 py-1.5 text-xs">
                {status?.callback_url || '(chưa cấu hình)'}
              </code>
              <Button size="sm" variant="outline" onClick={handleCopyCallback} disabled={!status?.callback_url}>
                <Copy className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>

          {/* Connect + refresh actions */}
          <div className="flex flex-wrap gap-2">
            <Button onClick={handleConnect} disabled={startOAuth.isPending || !status?.configured}>
              {startOAuth.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Link2 className="h-4 w-4" />}
              Kết nối Zalo
            </Button>
            <Button onClick={handleRefresh} variant="outline" disabled={refreshTok.isPending || !status?.connected}>
              {refreshTok.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
              Làm mới token
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Enable toggle */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-sm">
            <Power className="h-4 w-4" />
            Bật tính năng đặt lại mật khẩu qua Zalo
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">
                {status?.enabled ? 'Đang bật' : 'Đang tắt'}
              </p>
              <p className="text-xs text-muted-foreground">
                {status?.connected
                  ? 'Cho phép nhân viên đặt lại mật khẩu qua Zalo ZNS.'
                  : 'Kết nối Zalo trước khi bật.'}
              </p>
            </div>
            <Switch
              checked={status?.enabled ?? false}
              disabled={!status?.connected || setEnabled.isPending}
              onCheckedChange={handleToggle}
            />
          </div>
          {confirmDisable && (
            <div className="mt-3 rounded-lg border border-yellow-500/30 bg-yellow-500/10 p-3 text-xs">
              <p className="font-bold text-yellow-700">Tắt tính năng này?</p>
              <p className="mt-1 text-yellow-700/80">
                Nhân viên sẽ không thể đặt lại mật khẩu qua Zalo cho đến khi bật lại.
              </p>
              <div className="mt-2 flex gap-2">
                <Button size="sm" variant="destructive" onClick={() => handleToggle(false)}>
                  Xác nhận tắt
                </Button>
                <Button size="sm" variant="outline" onClick={() => setConfirmDisable(false)}>
                  Hủy
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
