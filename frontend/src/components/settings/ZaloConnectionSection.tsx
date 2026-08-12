import { type FormEvent, type ReactNode, useEffect, useState } from "react";
import { toast } from "sonner";
import {
  AlertCircle,
  Check,
  CheckCircle2,
  ChevronDown,
  Loader2,
  MessageCircle,
  PlugZap,
  RefreshCw,
  Send,
  XCircle,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { buildSalaryZnsTestPayload } from "./zalo-test-message";
import {
  useRefreshZaloToken,
  useSaveZaloCredentials,
  useSetFlexPayZNSEnabled,
  useSetZaloEnabled,
  useTestZaloSend,
  useZaloStatus,
} from "@/hooks/api/useZaloConnection";

type SettingsSectionProps = {
  id: string;
  title: string;
  description: string;
  children: ReactNode;
};

const SettingsSection = ({
  id,
  title,
  description,
  children,
}: SettingsSectionProps) => {
  const titleId = `${id}-title`;

  return (
    <section
      id={id}
      aria-labelledby={titleId}
      className="grid gap-5 border-t px-4 py-6 sm:px-6 lg:grid-cols-[14rem_minmax(0,1fr)] lg:gap-8 lg:py-7 xl:grid-cols-[16rem_minmax(0,1fr)]"
    >
      <div className="min-w-0">
        <div className="mb-2 flex items-center gap-2">
          <h3 id={titleId} className="text-sm font-semibold text-foreground">
            {title}
          </h3>
        </div>
        <p className="max-w-xs text-sm leading-5 text-muted-foreground">
          {description}
        </p>
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
};

type CredentialFieldProps = {
  id: string;
  label: string;
  hint?: string;
  children: ReactNode;
};

const CredentialField = ({
  id,
  label,
  hint,
  children,
}: CredentialFieldProps) => {
  const hintId = hint ? `${id}-hint` : undefined;

  return (
    <div className="min-w-0 space-y-2">
      <Label htmlFor={id} className="block text-sm font-medium">
        {label}
      </Label>
      {children}
      {hint ? (
        <p id={hintId} className="text-xs leading-5 text-muted-foreground">
          {hint}
        </p>
      ) : null}
    </div>
  );
};

type SetupStepProps = {
  step: string;
  label: string;
  complete: boolean;
  active: boolean;
};

type TestMessageKind = "otp" | "salary";

const SetupStep = ({ step, label, complete, active }: SetupStepProps) => (
  <li
    className="flex min-w-0 flex-col items-center gap-1.5 text-center sm:flex-row sm:gap-2 sm:text-left"
    aria-current={active ? "step" : undefined}
  >
    <span
      className={cn(
        "flex size-6 shrink-0 items-center justify-center rounded-full border text-[11px] font-semibold",
        complete && "border-emerald-600 bg-emerald-600 text-white",
        !complete && active && "border-primary bg-primary/10 text-primary",
        !complete &&
          !active &&
          "border-border bg-background text-muted-foreground",
      )}
      aria-hidden="true"
    >
      {complete ? <Check className="size-3.5" /> : step}
    </span>
    <span
      className={cn(
        "text-xs font-medium leading-4",
        complete || active ? "text-foreground" : "text-muted-foreground",
      )}
    >
      {label}
      <span className="sr-only">
        {complete
          ? ", đã hoàn tất"
          : active
            ? ", bước hiện tại"
            : ", chưa thực hiện"}
      </span>
    </span>
  </li>
);

/** Admin control surface for the Zalo ZNS password-reset connection. */
export const ZaloConnectionSection = () => {
  const { data: statusRes, isLoading, isError, refetch } = useZaloStatus();
  const status = statusRes?.data;

  const saveCreds = useSaveZaloCredentials();
  const setEnabled = useSetZaloEnabled();
  const setFlexPayZNSEnabledMutation = useSetFlexPayZNSEnabled();
  const testSend = useTestZaloSend();
  const refreshTok = useRefreshZaloToken();

  const [appID, setAppID] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [accessToken, setAccessToken] = useState("");
  const [refreshToken, setRefreshToken] = useState("");
  const [templateID, setTemplateID] = useState("619684");
  const [confirmDisable, setConfirmDisable] = useState(false);
  const [flexPayZNSEnabled, setFlexPayZNSEnabled] = useState(false);
  const [showCredentials, setShowCredentials] = useState(false);
  const [testPhone, setTestPhone] = useState("");
  const [testSendingKind, setTestSendingKind] =
    useState<TestMessageKind | null>(null);
  const [testResult, setTestResult] = useState<{
    kind: TestMessageKind;
    error_code: number;
    error_msg: string;
    msg_id?: string;
  } | null>(null);

  useEffect(() => {
    if (!status) return;
    setAppID(status.app_id || "");
    setTemplateID(status.template_id || "619684");
    setFlexPayZNSEnabled(status.flexpay_zns_enabled || false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    status?.app_id,
    status?.template_id,
    status?.configured,
    status?.flexpay_zns_enabled,
  ]);

  useEffect(() => {
    if (status?.configured === false) setShowCredentials(true);
  }, [status?.configured]);

  const handleFlexPayZNSToggle = async (enabled: boolean) => {
    if (!status?.connected) {
      toast.error("Cần kết nối Zalo trước khi bật thông báo lương linh hoạt");
      return;
    }
    try {
      await setFlexPayZNSEnabledMutation.mutateAsync(enabled);
      setFlexPayZNSEnabled(enabled);
    } catch {
      toast.error("Không thể cập nhật trạng thái");
    }
  };

  const handleSaveCreds = async () => {
    try {
      await saveCreds.mutateAsync({
        app_id: appID,
        secret_key: secretKey,
        template_id: templateID,
        access_token: accessToken,
        refresh_token: refreshToken,
      });
      toast.success("Đã lưu thông tin kết nối Zalo");
      setSecretKey("");
      setAccessToken("");
      setRefreshToken("");
      if (status?.configured) setShowCredentials(false);
    } catch {
      toast.error("Không thể lưu thông tin kết nối");
    }
  };

  const handleCredentialsSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void handleSaveCreds();
  };

  const handleTestConnection = async () => {
    try {
      await refreshTok.mutateAsync();
      toast.success(
        "Kết nối Zalo hợp lệ — App ID, Secret Key và Refresh Token đều chính xác",
      );
    } catch (error) {
      const message = error instanceof Error ? error.message : "Không rõ lỗi";
      toast.error(`Kết nối thất bại: ${message}`);
    }
  };

  const handleToggle = async (enabled: boolean) => {
    if (enabled && !status?.connected) {
      toast.error("Kết nối Zalo trước khi bật tính năng");
      return;
    }
    if (!enabled && !confirmDisable) {
      setConfirmDisable(true);
      return;
    }
    setConfirmDisable(false);
    try {
      await setEnabled.mutateAsync(enabled);
      toast.success(enabled ? "Đã bật Zalo OTP" : "Đã tắt Zalo OTP");
    } catch {
      toast.error("Không thể cập nhật trạng thái");
    }
  };

  const handleTestSend = async (kind: TestMessageKind) => {
    setTestResult(null);
    if (!testPhone.trim()) {
      toast.error("Nhập SĐT nhận thử");
      return;
    }
    setTestSendingKind(kind);
    try {
      const phone = testPhone.trim();
      const response = await testSend.mutateAsync(
        kind === "otp"
          ? {
              phone,
              template_id: status?.template_id || "619684",
              template_data: { otp: "000000" },
            }
          : {
              phone,
              ...buildSalaryZnsTestPayload(),
            },
      );
      const result = response.data;
      setTestResult({
        kind,
        error_code: result.error_code,
        error_msg: result.error_msg,
        msg_id: result.msg_id,
      });
      if (result.error_code === 0) {
        toast.success(
          `${kind === "otp" ? "Đã gửi OTP mẫu" : "Đã gửi mẫu thông báo lương"} (msg_id: ${result.msg_id || "—"})`,
        );
      } else {
        toast.error(`Lỗi ZNS: ${result.error_msg}`);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "Không rõ lỗi";
      toast.error(`Gửi thử thất bại: ${message}`);
    } finally {
      setTestSendingKind(null);
    }
  };

  if (isLoading) {
    return (
      <div
        role="status"
        aria-live="polite"
        className="rounded-xl border bg-card px-4 py-12 text-center text-sm text-muted-foreground"
      >
        <Loader2
          aria-hidden="true"
          className="mx-auto mb-3 size-6 animate-spin"
        />
        Đang tải trạng thái kết nối…
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-10 text-center">
        <AlertCircle
          aria-hidden="true"
          className="mx-auto mb-3 size-6 text-destructive"
        />
        <p className="text-sm font-medium text-destructive">
          Không thể tải trạng thái kết nối Zalo.
        </p>
        <Button
          type="button"
          variant="outline"
          className="mt-4 h-11 gap-2"
          onClick={() => void refetch()}
        >
          <RefreshCw aria-hidden="true" className="size-4" />
          Tải lại
        </Button>
      </div>
    );
  }

  const badge = (() => {
    if (!status?.configured) {
      return {
        icon: XCircle,
        text: "Chưa cấu hình",
        className: "border-border bg-muted text-muted-foreground",
      };
    }
    if (status.last_error) {
      return {
        icon: AlertCircle,
        text: "Có lỗi kết nối",
        className: "border-destructive/30 bg-destructive/5 text-destructive",
      };
    }
    if (!status.connected) {
      return {
        icon: AlertCircle,
        text: "Chưa kết nối",
        className: "border-amber-300 bg-amber-50 text-amber-800",
      };
    }
    return {
      icon: CheckCircle2,
      text: "Đã kết nối",
      className: "border-emerald-300 bg-emerald-50 text-emerald-800",
    };
  })();
  const BadgeIcon = badge.icon;
  const hasUnsavedCredentials = Boolean(
    appID !== (status?.app_id || "") ||
      templateID !== (status?.template_id || "619684") ||
      secretKey ||
      accessToken ||
      refreshToken,
  );

  const expiryText = (() => {
    if (!status?.expires_at) return "Chưa có thời hạn";
    const remainingMs = new Date(status.expires_at).getTime() - Date.now();
    if (remainingMs <= 0) return "Đã hết hạn";
    const hours = Math.floor(remainingMs / 3_600_000);
    const minutes = Math.floor((remainingMs % 3_600_000) / 60_000);
    return hours > 0
      ? `Còn ${hours} giờ ${minutes} phút`
      : `Còn ${minutes} phút`;
  })();

  return (
    <div className="overflow-hidden rounded-xl border bg-card text-card-foreground">
      <header className="bg-muted/20 px-4 py-5 sm:px-6 sm:py-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-sky-200 bg-sky-50 text-sky-700">
              <MessageCircle aria-hidden="true" className="size-5" />
            </div>
            <div className="min-w-0">
              <h2
                id="zalo-connection-title"
                className="text-base font-semibold text-foreground sm:text-lg"
              >
                Kênh thông báo Zalo ZNS
              </h2>
              <p className="mt-1 max-w-2xl text-sm leading-5 text-muted-foreground">
                Một kết nối dùng chung cho OTP đặt lại mật khẩu và thông báo
                lương linh hoạt.
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              variant="outline"
              className={cn("h-7 w-fit gap-1.5 px-2.5", badge.className)}
            >
              <BadgeIcon aria-hidden="true" className="size-3.5" />
              {badge.text}
            </Badge>
            {status?.configured ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-11 gap-1.5 px-2.5 text-sm"
                aria-expanded={showCredentials}
                aria-controls="zalo-credentials"
                onClick={() => setShowCredentials((current) => !current)}
              >
                {showCredentials ? "Đóng cấu hình" : "Thay đổi cấu hình"}
                <ChevronDown
                  aria-hidden="true"
                  className={cn(
                    "size-4 transition-transform",
                    showCredentials && "rotate-180",
                  )}
                />
              </Button>
            ) : null}
          </div>
        </div>

        <div className="mt-5 grid gap-4 border-t pt-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
          <ol
            aria-label="Tiến độ thiết lập Zalo ZNS"
            className="grid grid-cols-3 gap-3"
          >
            <SetupStep
              step="1"
              label="Cấu hình"
              complete={Boolean(status?.configured)}
              active={!status?.configured}
            />
            <SetupStep
              step="2"
              label="Kết nối"
              complete={Boolean(status?.connected)}
              active={Boolean(status?.configured && !status.connected)}
            />
            <SetupStep
              step="3"
              label="Dịch vụ"
              complete={Boolean(status?.enabled || flexPayZNSEnabled)}
              active={Boolean(
                status?.connected && !status.enabled && !flexPayZNSEnabled,
              )}
            />
          </ol>
          <dl className="flex flex-wrap gap-x-5 gap-y-2 text-xs">
            <div className="flex gap-1.5">
              <dt className="text-muted-foreground">Mã mẫu ZNS</dt>
              <dd className="font-mono font-medium text-foreground">
                {status?.template_id || "619684"}
              </dd>
            </div>
            <div className="flex gap-1.5">
              <dt className="text-muted-foreground">Hiệu lực mã truy cập</dt>
              <dd className="font-medium text-foreground">{expiryText}</dd>
            </div>
          </dl>
        </div>

        {status?.last_error ? (
          <div
            role="alert"
            className="mt-4 flex items-start gap-2 border-t border-destructive/20 pt-4 text-sm text-destructive"
          >
            <AlertCircle
              aria-hidden="true"
              className="mt-0.5 size-4 shrink-0"
            />
            <p className="min-w-0 break-words">
              <span className="font-semibold">Lỗi gần nhất:</span>{" "}
              {status.last_error}
            </p>
          </div>
        ) : null}
      </header>

      {showCredentials ? (
      <SettingsSection
        id="zalo-credentials"
        title="Cấu hình kết nối"
        description="Nhập thông tin ứng dụng, mã truy cập và mã mẫu từ Zalo OA Console. Khóa bảo mật không hiển thị lại sau khi lưu."
      >
        <form onSubmit={handleCredentialsSubmit} className="space-y-5">
          <div className="grid gap-5 md:grid-cols-2">
            <CredentialField id="zalo-app-id" label="App ID">
              <Input
                id="zalo-app-id"
                name="zalo-app-id"
                value={appID}
                onChange={(event) => setAppID(event.target.value)}
                placeholder="App ID từ Zalo OA Console"
                disabled={saveCreds.isPending}
                className="h-11"
              />
            </CredentialField>
            <CredentialField
              id="zalo-secret"
              label="Khóa bí mật (Secret Key)"
              hint="Để trống nếu bạn muốn giữ nguyên khóa hiện tại."
            >
              <Input
                id="zalo-secret"
                name="zalo-secret"
                type="password"
                value={secretKey}
                onChange={(event) => setSecretKey(event.target.value)}
                placeholder="Chỉ nhập khi cần thay đổi"
                disabled={saveCreds.isPending}
                autoComplete="new-password"
                className="h-11"
                aria-describedby="zalo-secret-hint"
              />
            </CredentialField>
            <CredentialField
              id="zalo-access-token"
              label="Mã truy cập (Access Token)"
              hint="Lấy từ Zalo OA Console → Token quản trị trang. Có hiệu lực khoảng 24 giờ; hệ thống tự làm mới qua Refresh Token."
            >
              <Input
                id="zalo-access-token"
                name="zalo-access-token"
                type="password"
                value={accessToken}
                onChange={(event) => setAccessToken(event.target.value)}
                placeholder="Chỉ nhập khi cần thay đổi"
                disabled={saveCreds.isPending}
                autoComplete="new-password"
                className="h-11"
                aria-describedby="zalo-access-token-hint"
              />
            </CredentialField>
            <CredentialField
              id="zalo-refresh-token"
              label="Mã làm mới (Refresh Token)"
              hint="Cần App ID và khóa bí mật để hệ thống tự làm mới mã truy cập."
            >
              <Input
                id="zalo-refresh-token"
                name="zalo-refresh-token"
                type="password"
                value={refreshToken}
                onChange={(event) => setRefreshToken(event.target.value)}
                placeholder="Chỉ nhập khi cần thay đổi"
                disabled={saveCreds.isPending}
                autoComplete="new-password"
                className="h-11"
                aria-describedby="zalo-refresh-token-hint"
              />
            </CredentialField>
            <div className="md:max-w-xs">
              <CredentialField id="zalo-template" label="Mã mẫu ZNS">
                <Input
                  id="zalo-template"
                  name="zalo-template"
                  value={templateID}
                  onChange={(event) => setTemplateID(event.target.value)}
                  placeholder="619684"
                  disabled={saveCreds.isPending}
                  className="h-11"
                />
              </CredentialField>
            </div>
          </div>

          <div className="flex flex-col gap-2 border-t pt-4 sm:flex-row sm:items-center">
            <Button
              type="submit"
              className="h-11 gap-2 sm:w-auto"
              disabled={saveCreds.isPending}
            >
              {saveCreds.isPending ? (
                <Loader2 aria-hidden="true" className="size-4 animate-spin" />
              ) : null}
              Lưu cấu hình
            </Button>
            <Button
              type="button"
              variant="outline"
              className="h-11 gap-2 sm:w-auto"
              onClick={handleTestConnection}
              disabled={
                refreshTok.isPending ||
                !status?.configured ||
                hasUnsavedCredentials
              }
            >
              {refreshTok.isPending ? (
                <Loader2 aria-hidden="true" className="size-4 animate-spin" />
              ) : (
                <PlugZap aria-hidden="true" className="size-4" />
              )}
              Kiểm tra cấu hình đã lưu
            </Button>
            <p className="text-xs leading-5 text-muted-foreground sm:ml-2">
              {hasUnsavedCredentials
                ? "Lưu thay đổi trước khi kiểm tra."
                : "Kiểm tra khóa đã lưu với Zalo mà không gửi tin nhắn."}
            </p>
          </div>
        </form>
      </SettingsSection>
      ) : null}

      <SettingsSection
        id="zalo-services"
        title="Dịch vụ sử dụng kết nối"
        description="Chọn nghiệp vụ được phép gửi qua kênh ZNS. Hai dịch vụ hoạt động độc lập trên cùng một kết nối."
      >
        <div className="overflow-hidden rounded-lg border bg-background">
          <label
            htmlFor="zalo-enabled"
            className={cn(
              "flex min-h-[4.5rem] cursor-pointer items-center justify-between gap-4 px-4 py-3.5",
              !status?.connected && "cursor-not-allowed bg-muted/40",
            )}
          >
            <span className="min-w-0">
              <span className="block text-sm font-semibold text-foreground">
                OTP đặt lại mật khẩu
              </span>
              <span className="mt-0.5 block text-xs leading-5 text-muted-foreground">
                {status?.connected
                  ? status.enabled
                    ? "Đang gửi OTP qua Zalo cho nhân viên."
                    : "Cho phép nhân viên nhận OTP đặt lại mật khẩu qua Zalo."
                  : "Hoàn tất kết nối trước khi bật dịch vụ."}
              </span>
            </span>
            <Switch
              id="zalo-enabled"
              checked={status?.enabled ?? false}
              disabled={!status?.connected || setEnabled.isPending}
              onCheckedChange={handleToggle}
              aria-label="Bật đặt lại mật khẩu qua Zalo"
            />
          </label>

          <label
            htmlFor="flexpay-zns-enabled"
            className={cn(
              "flex min-h-[4.5rem] cursor-pointer items-center justify-between gap-4 border-t px-4 py-3.5",
              !status?.connected && "cursor-not-allowed bg-muted/40",
            )}
          >
            <span className="min-w-0">
              <span className="block text-sm font-semibold text-foreground">
                Thông báo lương linh hoạt
              </span>
              <span className="mt-0.5 block text-xs leading-5 text-muted-foreground">
                {status?.connected
                  ? "Gửi hạn mức và hạn yêu cầu sau khi nhập bảng lương linh hoạt · 400 ₫/tin."
                  : "Hoàn tất kết nối trước khi bật dịch vụ."}
              </span>
            </span>
            <Switch
              id="flexpay-zns-enabled"
              checked={flexPayZNSEnabled}
              disabled={
                !status?.connected || setFlexPayZNSEnabledMutation.isPending
              }
              onCheckedChange={handleFlexPayZNSToggle}
              aria-label="Bật thông báo ZNS lương linh hoạt"
            />
          </label>
        </div>

        {confirmDisable ? (
          <div
            role="alert"
            className="mt-3 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900"
          >
            <p className="font-semibold">Tắt đặt lại mật khẩu qua Zalo?</p>
            <p className="mt-1 leading-5 text-amber-800">
              Nhân viên sẽ không thể nhận OTP qua Zalo cho đến khi tính năng
              được bật lại.
            </p>
            <div className="mt-3 flex flex-col gap-2 sm:flex-row">
              <Button
                type="button"
                className="h-11"
                variant="destructive"
                onClick={() => void handleToggle(false)}
              >
                Xác nhận tắt
              </Button>
              <Button
                type="button"
                className="h-11"
                variant="outline"
                onClick={() => setConfirmDisable(false)}
              >
                Hủy
              </Button>
            </div>
          </div>
        ) : null}
      </SettingsSection>

      <SettingsSection
        id="zalo-test-send"
        title="Kiểm tra gửi thực tế"
        description="Gửi một trong hai mẫu đến số điện thoại kiểm tra trước khi đưa kênh vào sử dụng."
      >
        <div className="space-y-2">
          <Label htmlFor="zalo-test-phone">Số điện thoại nhận thử</Label>
          <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-start">
            <div className="min-w-0">
              <Input
                id="zalo-test-phone"
                name="zalo-test-phone"
                type="tel"
                inputMode="tel"
                autoComplete="tel"
                value={testPhone}
                onChange={(event) => setTestPhone(event.target.value)}
                placeholder="84987654321"
                disabled={testSend.isPending}
                className="h-11"
                aria-describedby="zalo-test-phone-hint"
              />
              <p
                id="zalo-test-phone-hint"
                className="mt-2 text-xs leading-5 text-muted-foreground"
              >
                Nhập số điện thoại dạng 0xxxxxxxxx hoặc 84xxxxxxxxx, không có
                dấu cách.
              </p>
            </div>
            <div
              role="group"
              aria-label="Chọn loại tin nhắn gửi thử"
              className="grid gap-2 sm:grid-cols-2"
            >
              <Button
                type="button"
                variant="outline"
                className="h-11 gap-2"
                onClick={() => void handleTestSend("otp")}
                disabled={testSend.isPending || !status?.configured}
              >
                {testSend.isPending && testSendingKind === "otp" ? (
                  <Loader2 aria-hidden="true" className="size-4 animate-spin" />
                ) : (
                  <Send aria-hidden="true" className="size-4" />
                )}
                {testSend.isPending && testSendingKind === "otp"
                  ? "Đang gửi OTP…"
                  : "Gửi OTP mẫu"}
              </Button>
              <Button
                type="button"
                variant="outline"
                className="h-11 gap-2"
                onClick={() => void handleTestSend("salary")}
                disabled={testSend.isPending || !status?.configured}
              >
                {testSend.isPending && testSendingKind === "salary" ? (
                  <Loader2 aria-hidden="true" className="size-4 animate-spin" />
                ) : (
                  <Send aria-hidden="true" className="size-4" />
                )}
                {testSend.isPending && testSendingKind === "salary"
                  ? "Đang gửi mẫu lương…"
                  : "Gửi mẫu lương"}
              </Button>
            </div>
          </div>
        </div>

        {testResult ? (
          <div
            role="status"
            aria-live="polite"
            className={cn(
              "mt-4 rounded-lg border p-3 text-sm",
              testResult.error_code === 0
                ? "border-emerald-300 bg-emerald-50 text-emerald-800"
                : "border-destructive/30 bg-destructive/5 text-destructive",
            )}
          >
            <span className="font-semibold">
              {testResult.error_code === 0
                ? testResult.kind === "otp"
                  ? "OTP mẫu đã gửi thành công"
                  : "Mẫu thông báo lương đã gửi thành công"
                : `${testResult.kind === "otp" ? "OTP mẫu" : "Mẫu thông báo lương"} lỗi ${testResult.error_code}`}
            </span>
            {" — "}
            {testResult.error_msg}
            {testResult.msg_id ? (
              <span className="ml-2 break-all font-mono text-xs text-muted-foreground">
                msg_id: {testResult.msg_id}
              </span>
            ) : null}
          </div>
        ) : null}
      </SettingsSection>
    </div>
  );
};
