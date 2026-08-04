import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { MessageCircle, CheckCircle2, XCircle, AlertCircle, Loader2, Power, Send, PlugZap } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import {
  useZaloStatus,
  useSaveZaloCredentials,
  useSetZaloEnabled,
  useTestZaloSend,
  useRefreshZaloToken,
} from '@/hooks/api/useZaloConnection';

/**
 * ZaloConnectionSection — the admin control surface for the Zalo ZNS connection.
 *
 * Renders the live connection status badge, the credentials form (app_id /
 * secret_key / access_token / refresh_token / template_id — tokens are pasted
 * manually), the runtime enable/disable toggle, and a manual token-refresh
 * button. All mutations invalidate the status query on success so the badge
 * updates immediately.
 *
 * Secrets are write-only: the server never returns secret_key, access_token, or
 * refresh_token in the status payload, so those fields are never populated from
 * the server (the secret_key input is empty by default — "leave blank to keep
 * existing").
 */
export const ZaloConnectionSection = () => {
  const { data: statusRes, isLoading, isError } = useZaloStatus();
  const status = statusRes?.data;

  const saveCreds = useSaveZaloCredentials();
  const setEnabled = useSetZaloEnabled();
  const testSend = useTestZaloSend();
  const refreshTok = useRefreshZaloToken();

  // Form state — secret_key/access_token/refresh_token are write-only (never
  // echoed from server; empty on submit means "keep existing").
  const [appID, setAppID] = useState('');
  const [secretKey, setSecretKey] = useState('');
  const [accessToken, setAccessToken] = useState('');
  const [refreshToken, setRefreshToken] = useState('');
  const [templateID, setTemplateID] = useState('617976');
  const [confirmDisable, setConfirmDisable] = useState(false);
  const [testPhone, setTestPhone] = useState('');
  const [testResult, setTestResult] = useState<{ error_code: number; error_msg: string; msg_id?: string } | null>(null);

  // Populate template_id from status once loaded. App ID is write-only
  // (never echoed from server) — leaving it empty means "keep existing" on
  // save, matching the password fields. The status badge already tells the
  // admin whether the connection is configured.
  useEffect(() => {
    if (!status) return;
    setTemplateID(status.template_id || '617976');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status?.template_id, status?.configured]);

  const handleSaveCreds = async () => {
    try {
      await saveCreds.mutateAsync({
        app_id: appID,
        secret_key: secretKey,
        template_id: templateID,
        access_token: accessToken,
        refresh_token: refreshToken,
      });
      toast.success('Đã lưu thông tin kết nối Zalo');
      // Clear write-only password fields after save (server keeps existing
      // values for fields left blank).
      setSecretKey('');
      setAccessToken('');
      setRefreshToken('');
    } catch (e) {
      toast.error('Không thể lưu thông tin kết nối');
    }
  };

  const handleTestConnection = async () => {
    try {
      // RefreshNow calls Zalo's OAuth /v4/oa/access_token with
      // grant_type=refresh_token — this validates app_id + secret_key +
      // refresh_token against Zalo's servers WITHOUT sending any message.
      // Template approval is not required. Returns fresh access+refresh tokens.
      await refreshTok.mutateAsync();
      toast.success('Kết nối Zalo hợp lệ — App ID, Secret, Refresh Token đều chính xác');
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Không rõ lỗi';
      toast.error(`Kết nối thất bại: ${msg}`);
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

  const handleTestSend = async () => {
    setTestResult(null);
    if (!testPhone.trim()) {
      toast.error('Nhập SĐT nhận thử');
      return;
    }
    try {
      const res = await testSend.mutateAsync({ phone: testPhone.trim() });
      const r = res.data;
      setTestResult({
        error_code: r.error_code,
        error_msg: r.error_msg,
        msg_id: r.msg_id,
      });
      if (r.error_code === 0) {
        toast.success(`Đã gửi thử (msg_id: ${r.msg_id || '—'})`);
      } else {
        toast.error(`Lỗi ZNS: ${r.error_msg}`);
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Không rõ lỗi';
      toast.error(`Gửi thử thất bại: ${msg}`);
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
            <Label htmlFor="zalo-app-id">
              App ID
              {status?.configured && (
                <span className="ml-2 text-xs font-normal text-muted-foreground">
                  (để trống để giữ nguyên)
                </span>
              )}
            </Label>
            <Input
              id="zalo-app-id"
              value={appID}
              onChange={(e) => setAppID(e.target.value)}
              placeholder={status?.configured ? '(đã lưu)' : 'App ID từ Zalo OA Console'}
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
            <Label htmlFor="zalo-access-token">
              Access Token
              <span className="ml-2 text-xs font-normal text-muted-foreground">
                (dán thủ công — để trống để giữ nguyên)
              </span>
            </Label>
            <Input
              id="zalo-access-token"
              type="password"
              value={accessToken}
              onChange={(e) => setAccessToken(e.target.value)}
              placeholder="••••••••••••••••"
              disabled={saveCreds.isPending}
              autoComplete="new-password"
            />
            <p className="text-xs text-muted-foreground">
              Hết hạn ~24h. Dán token mới → đồng hồ hết hạn tự đặt lại +24h. Thay
              thế OAuth khi chạy localhost.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="zalo-refresh-token">
              Refresh Token
              <span className="ml-2 text-xs font-normal text-muted-foreground">
                (dán thủ công — để trống để giữ nguyên)
              </span>
            </Label>
            <Input
              id="zalo-refresh-token"
              type="password"
              value={refreshToken}
              onChange={(e) => setRefreshToken(e.target.value)}
              placeholder="••••••••••••••••"
              disabled={saveCreds.isPending}
              autoComplete="new-password"
            />
            <p className="text-xs text-muted-foreground">
              Cần kèm App ID + Secret Key để hệ thống tự làm mới token khi hết hạn.
            </p>
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
          <div className="flex flex-wrap gap-2">
            <Button
              onClick={handleSaveCreds}
              disabled={saveCreds.isPending || (!appID.trim() && !status?.configured)}
            >
              {saveCreds.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              Lưu thông tin
            </Button>
            <Button
              onClick={handleTestConnection}
              variant="outline"
              disabled={refreshTok.isPending || !status?.configured}
            >
              {refreshTok.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <PlugZap className="h-4 w-4" />}
              Kiểm tra kết nối
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">
            &quot;Kiểm tra kết nối&quot; xác thực App ID + Secret + Refresh Token với Zalo
            mà <strong>không gửi tin nhắn</strong> — dùng được khi template chưa được duyệt.
          </p>
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

      {/* Test send */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Gửi thử ZNS</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs text-muted-foreground">
            Gửi 1 tin ZNS thử (template OTP {status?.template_id || '617976'}) đến SĐT
            bạn nhập để kiểm tra tokens đang hoạt động. Không ảnh hưởng luồng đặt
            lại mật khẩu — không lưu OTP vào Redis.
          </p>
          <div className="flex flex-wrap items-end gap-2">
            <div className="min-w-[200px] flex-1 space-y-2">
              <Label htmlFor="zalo-test-phone">SĐT nhận thử</Label>
              <Input
                id="zalo-test-phone"
                value={testPhone}
                onChange={(e) => setTestPhone(e.target.value)}
                placeholder="84987654321 (84 + 9 số, không dấu cách)"
                disabled={testSend.isPending}
              />
            </div>
            <Button
              onClick={handleTestSend}
              disabled={testSend.isPending || !status?.configured}
            >
              {testSend.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
              Gửi thử
            </Button>
          </div>
          {testResult && (
            <div
              className={`rounded-lg border p-3 text-xs ${
                testResult.error_code === 0
                  ? 'border-green-500/30 bg-green-500/5 text-green-700'
                  : 'border-destructive/30 bg-destructive/5 text-destructive'
              }`}
            >
              <strong>{testResult.error_code === 0 ? 'Thành công' : `Lỗi ${testResult.error_code}`}</strong>
              {' — '}
              {testResult.error_msg}
              {testResult.msg_id && (
                <span className="ml-2 font-mono text-muted-foreground">msg_id: {testResult.msg_id}</span>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};
