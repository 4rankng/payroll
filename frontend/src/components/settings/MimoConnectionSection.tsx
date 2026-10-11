import { type FormEvent, type ReactNode, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import {
  AlertCircle,
  Bot,
  CheckCircle2,
  KeyRound,
  Loader2,
  PlugZap,
  RefreshCw,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import { Switch } from '@/components/ui/switch';
import {
  useCreateSetting,
  useSettings,
  useTestMimoConnection,
  useUpdateSetting,
} from '@/hooks/api/useSettings';
import {
  MIMO_MODELS,
  MIMO_SETTING_KEYS,
  type MimoTestConnectionCode,
} from '@/types/api/settings.types';

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
      className="grid gap-5 border-t bg-card px-4 py-6 sm:px-6 lg:grid-cols-[14rem_minmax(0,1fr)] lg:gap-8 lg:py-7 xl:grid-cols-[16rem_minmax(0,1fr)]"
    >
      <div className="min-w-0">
        <div className="mb-2 flex items-center gap-2">
          <h3 id={titleId} className="text-sm font-semibold text-foreground">
            {title}
          </h3>
        </div>
        <p className="max-w-xs text-sm leading-5 text-muted-foreground">{description}</p>
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
};

/**
 * Designed error-state copy for every test-connection outcome. `hint` tells
 * the admin what to do next; `tone` picks the panel color. Button behaviour
 * per state is implemented in the component: "missing_key" disables the test
 * button (nothing to test yet), every other state keeps both buttons enabled
 * so the admin can re-save or retry.
 */
const TEST_STATE_COPY: Record<
  Exclude<MimoTestConnectionCode, 'ok'>,
  { title: string; tone: 'warning' | 'error'; hint: string }
> = {
  missing_key: {
    title: 'Chưa có Khóa bí mật',
    tone: 'warning',
    hint: 'Nhập Khóa bí mật và bấm "Lưu cấu hình" để bật trợ lý AI.',
  },
  invalid_key: {
    title: 'Khóa API không hợp lệ',
    tone: 'error',
    hint: 'Khóa đã bị thu hồi hoặc sai — dán khóa mới từ MiMo Platform rồi lưu lại.',
  },
  timeout: {
    title: 'MiMo không phản hồi kịp',
    tone: 'warning',
    hint: 'Kết nối đã hết thời gian chờ — thử lại sau ít phút.',
  },
  rate_limited: {
    title: 'Đã vượt giới hạn tần suất',
    tone: 'warning',
    hint: 'Thử lại sau ít phút.',
  },
  unavailable: {
    title: 'Dịch vụ MiMo đang gián đoạn',
    tone: 'error',
    hint: 'Thử lại sau ít phút. Các nghiệp vụ khác không bị ảnh hưởng.',
  },
  busy: {
    title: 'Hệ thống đang xử lý nhiều yêu cầu AI',
    tone: 'warning',
    hint: 'Thử lại sau ít phút.',
  },
  error: {
    title: 'Không thể kiểm tra kết nối',
    tone: 'error',
    hint: 'Kiểm tra kết nối mạng và thử lại.',
  },
};

const MODEL_LABELS: Record<string, string> = {
  'mimo-v2.6-flash': 'Nhanh — mimo-v2.6-flash',
  'mimo-v2.6-pro': 'Chất lượng cao — mimo-v2.6-pro',
};

type TestState = {
  code: MimoTestConnectionCode;
  message: string;
  latencyMs?: number;
};

/** Admin control surface for the Xiaomi MiMo LLM connection and model routing. */
export const MimoConnectionSection = () => {
  // The settings list filtered by "mimo" (not useMultipleSettings): on a fresh
  // environment NONE of the rows exist yet, and an all-miss batch would read as
  // a load failure. An empty list is the designed "Chưa cấu hình" state.
  const {
    data: settings,
    isLoading,
    isError,
    refetch,
  } = useSettings({ search: 'mimo' });

  const updateSetting = useUpdateSetting();
  const createSetting = useCreateSetting();
  const testConnection = useTestMimoConnection();

  const [apiKeyInput, setApiKeyInput] = useState('');
  const [defaultModel, setDefaultModel] = useState<string>(MIMO_MODELS[0]);
  const [fallbackModel, setFallbackModel] = useState<string>(MIMO_MODELS[1]);
  const [chatModel, setChatModel] = useState<string>('');
  const [deliverableModel, setDeliverableModel] = useState<string>('');
  const [testState, setTestState] = useState<TestState | null>(null);
  const [chatAdmin, setChatAdmin] = useState(true);
  const [chatPartner, setChatPartner] = useState(true);
  const [togglingRole, setTogglingRole] = useState(false);

  const apiKeySetting = useMemo(
    () => (Array.isArray(settings) ? settings.find((s) => s?.key === MIMO_SETTING_KEYS.apiKey) : undefined),
    [settings],
  );
  const defaultModelSetting = useMemo(
    () => (Array.isArray(settings) ? settings.find((s) => s?.key === MIMO_SETTING_KEYS.defaultModel) : undefined),
    [settings],
  );
  const chatModelSetting = useMemo(
    () => (Array.isArray(settings) ? settings.find((s) => s?.key === MIMO_SETTING_KEYS.modelChat) : undefined),
    [settings],
  );
  const deliverableModelSetting = useMemo(
    () => (Array.isArray(settings) ? settings.find((s) => s?.key === MIMO_SETTING_KEYS.modelDeliverable) : undefined),
    [settings],
  );
  const fallbackModelSetting = useMemo(
    () => (Array.isArray(settings) ? settings.find((s) => s?.key === MIMO_SETTING_KEYS.fallbackModel) : undefined),
    [settings],
  );

  const findSetting = (key: string) =>
    Array.isArray(settings) ? settings.find((s) => s?.key === key) : undefined;
  const chatAdminSetting = findSetting(MIMO_SETTING_KEYS.chatEnabledAdmin);
  const chatPartnerSetting = findSetting(MIMO_SETTING_KEYS.chatEnabledPartner);

  const keyConfigured = Boolean(apiKeySetting?.masked ? apiKeySetting.configured : apiKeySetting?.value);

  useEffect(() => {
    setChatModel(chatModelSetting?.value ?? '');
    setDeliverableModel(deliverableModelSetting?.value ?? '');
  }, [chatModelSetting?.value, deliverableModelSetting?.value]);

  useEffect(() => {
    if (defaultModelSetting?.value) setDefaultModel(defaultModelSetting.value);
    if (fallbackModelSetting?.value) setFallbackModel(fallbackModelSetting.value);
  }, [defaultModelSetting?.value, fallbackModelSetting?.value]);
  useEffect(() => {
    setChatAdmin(chatAdminSetting?.value !== 'false');
    setChatPartner(chatPartnerSetting?.value !== 'false');
  }, [chatAdminSetting?.value, chatPartnerSetting?.value]);

  const isSaving = updateSetting.isPending || createSetting.isPending;
  const hasUnsavedKey = apiKeyInput.trim().length > 0;
  const hasUnsavedChanges =
    hasUnsavedKey ||
    defaultModel !== (defaultModelSetting?.value ?? MIMO_MODELS[0]) ||
    fallbackModel !== (fallbackModelSetting?.value ?? MIMO_MODELS[1]) ||
    chatModel !== (chatModelSetting?.value ?? '') ||
    deliverableModel !== (deliverableModelSetting?.value ?? '');

  // Upserts one string setting: the row may not exist yet on environments
  // where the key was never saved, so a miss creates it.
  const upsertSetting = async (key: string, value: string) => {
    const existing = findSetting(key);
    if (existing?.id) {
      await updateSetting.mutateAsync({ id: existing.id, data: { value } });
    } else {
      await createSetting.mutateAsync({ key, value, value_type: 'string' });
    }
  };

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    try {
      if (hasUnsavedKey) {
        await upsertSetting(MIMO_SETTING_KEYS.apiKey, apiKeyInput.trim());
      }
      await upsertSetting(MIMO_SETTING_KEYS.defaultModel, defaultModel);
      await upsertSetting(MIMO_SETTING_KEYS.fallbackModel, fallbackModel);
      // Route overrides: persist only when set or clearing a stored row —
      // an untouched "theo mặc định" must not create an empty row.
      if (chatModel || chatModelSetting?.id) {
        await upsertSetting(MIMO_SETTING_KEYS.modelChat, chatModel);
      }
      if (deliverableModel || deliverableModelSetting?.id) {
        await upsertSetting(MIMO_SETTING_KEYS.modelDeliverable, deliverableModel);
      }
      setApiKeyInput('');
      toast.success('Đã lưu cấu hình MiMo');
      // A fresh key invalidates the previous test verdict.
      setTestState(null);
    } catch {
      toast.error('Không thể lưu cấu hình MiMo — kiểm tra giá trị và thử lại.');
    }
  };

  const handleTestConnection = async () => {
    setTestState(null);
    try {
      const result = await testConnection.mutateAsync();
      if (result.success) {
        setTestState({ code: 'ok', message: result.message, latencyMs: result.latency_ms });
        toast.success('Kết nối MiMo hợp lệ');
      } else {
        setTestState({ code: result.code, message: result.message });
      }
    } catch {
      setTestState({
        code: 'error',
        message: 'Không thể kiểm tra kết nối — kiểm tra kết nối mạng và thử lại.',
      });
    }
  };

  // Per-role switches apply immediately (optimistic flip, revert on failure).
  const handleRoleToggle = async (role: 'admin' | 'partner', next: boolean) => {
    const key =
      role === 'admin' ? MIMO_SETTING_KEYS.chatEnabledAdmin : MIMO_SETTING_KEYS.chatEnabledPartner;
    const setter = role === 'admin' ? setChatAdmin : setChatPartner;
    const roleLabel = role === 'admin' ? 'quản trị viên' : 'đối tác';
    setter(next);
    setTogglingRole(true);
    try {
      await upsertSetting(key, next ? 'true' : 'false');
      toast.success(`${next ? 'Đã bật' : 'Đã tắt'} trợ lý AI cho ${roleLabel}`);
    } catch {
      setter(!next);
      toast.error('Không thể lưu thiết lập — thử lại.');
    } finally {
      setTogglingRole(false);
    }
  };

  if (isLoading) {
    return (
      <div
        role="status"
        aria-live="polite"
        className="rounded-xl border bg-card px-4 py-12 text-center text-sm text-muted-foreground"
      >
        <Loader2 aria-hidden="true" className="mx-auto mb-3 size-6 animate-spin" />
        Đang tải cấu hình MiMo…
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-10 text-center">
        <AlertCircle aria-hidden="true" className="mx-auto mb-3 size-6 text-destructive" />
        <p className="text-sm font-medium text-destructive">Không thể tải cấu hình MiMo.</p>
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

  // Designed state for a missing key — shown even before any test runs, so the
  // admin always knows why the assistant is inactive.
  const effectiveState: TestState | null =
    testState ??
    (!keyConfigured
      ? { code: 'missing_key', message: 'Trợ lý AI chưa hoạt động vì hệ thống chưa có Khóa bí mật MiMo.' }
      : null);
  const stateCopy =
    effectiveState && effectiveState.code !== 'ok' ? TEST_STATE_COPY[effectiveState.code] : null;

  return (
    <div className="overflow-hidden rounded-xl border bg-card text-card-foreground">
      <header className="bg-muted/20 px-4 py-5 sm:px-6 sm:py-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-violet-200 bg-violet-50 text-violet-700">
              <Bot aria-hidden="true" className="size-5" />
            </div>
            <div className="min-w-0">
              <h2
                id="mimo-connection-title"
                className="text-base font-semibold text-foreground sm:text-lg"
              >
                Trợ lý AI MiMo
              </h2>
              <p className="mt-1 max-w-2xl text-sm leading-5 text-muted-foreground">
                Kết nối Xiaomi MiMo cho các trợ lý AI của hệ thống. Khóa bí mật được mã hóa khi lưu
                và không bao giờ hiển thị lại.
              </p>
            </div>
          </div>
          <Badge
            variant="outline"
            className={cn(
              'h-7 w-fit gap-1.5 px-2.5',
              keyConfigured
                ? 'border-emerald-300 bg-emerald-50 text-emerald-800'
                : 'border-border bg-muted text-muted-foreground',
            )}
          >
            {keyConfigured ? (
              <CheckCircle2 aria-hidden="true" className="size-3.5" />
            ) : (
              <KeyRound aria-hidden="true" className="size-3.5" />
            )}
            {keyConfigured ? 'Đã cấu hình' : 'Chưa cấu hình'}
          </Badge>
        </div>
      </header>

      <SettingsSection
        id="mimo-credentials"
        title="Cấu hình kết nối"
        description="Nhập Khóa bí mật từ MiMo Platform và chọn mô hình sử dụng. Khóa chỉ hiển thị khi nhập, không hiển thị lại sau khi lưu."
      >
        <form onSubmit={handleSave} className="space-y-5">
          <div className="grid gap-5 md:grid-cols-2">
            <div className="min-w-0 space-y-2">
              <Label htmlFor="mimo-api-key" className="block text-sm font-medium">
                Khóa bí mật
              </Label>
              <Input
                id="mimo-api-key"
                name="mimo-api-key"
                type="password"
                value={apiKeyInput}
                onChange={(event) => setApiKeyInput(event.target.value)}
                placeholder={keyConfigured ? 'Đã lưu — chỉ nhập khi thay đổi' : 'Nhập Khóa bí mật MiMo'}
                disabled={isSaving}
                autoComplete="new-password"
                className="h-11"
                aria-describedby="mimo-api-key-hint"
              />
              <p id="mimo-api-key-hint" className="text-xs leading-5 text-muted-foreground">
                {keyConfigured
                  ? 'Để trống nếu bạn muốn giữ nguyên khóa hiện tại.'
                  : 'Khóa được mã hóa khi lưu và không hiển thị lại sau khi lưu.'}
              </p>
            </div>

            <div className="min-w-0 space-y-2">
              <Label htmlFor="mimo-default-model" className="block text-sm font-medium">
                Mô hình mặc định
              </Label>
              <select
                id="mimo-default-model"
                name="mimo-default-model"
                value={defaultModel}
                onChange={(event) => setDefaultModel(event.target.value)}
                disabled={isSaving}
                className="flex h-11 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                {MIMO_MODELS.map((model) => (
                  <option key={model} value={model}>
                    {MODEL_LABELS[model] ?? model}
                  </option>
                ))}
              </select>
              <p className="text-xs leading-5 text-muted-foreground">
                Dùng cho các tác vụ hằng ngày (soạn thảo, hội thoại).
              </p>
            </div>

            <div className="min-w-0 space-y-2">
              <Label htmlFor="mimo-fallback-model" className="block text-sm font-medium">
                Mô hình dự phòng
              </Label>
              <select
                id="mimo-fallback-model"
                name="mimo-fallback-model"
                value={fallbackModel}
                onChange={(event) => setFallbackModel(event.target.value)}
                disabled={isSaving}
                className="flex h-11 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                {MIMO_MODELS.map((model) => (
                  <option key={model} value={model}>
                    {MODEL_LABELS[model] ?? model}
                  </option>
                ))}
              </select>
              <p className="text-xs leading-5 text-muted-foreground">
                Tự động dùng thay thế nếu mô hình mặc định không khả dụng.
              </p>
            </div>

            <div className="min-w-0 space-y-2">
              <Label htmlFor="mimo-chat-model" className="block text-sm font-medium">
                Mô hình hội thoại (trợ lý AI)
              </Label>
              <select
                id="mimo-chat-model"
                value={chatModel}
                onChange={(event) => setChatModel(event.target.value)}
                disabled={isSaving}
                className="flex h-11 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option value="">Theo mô hình mặc định</option>
                {MIMO_MODELS.map((model) => (
                  <option key={model} value={model}>
                    {MODEL_LABELS[model] ?? model}
                  </option>
                ))}
              </select>
              <p className="text-xs leading-5 text-muted-foreground">
                Dùng cho các lượt hội thoại trong trợ lý AI.
              </p>
            </div>

            <div className="min-w-0 space-y-2">
              <Label htmlFor="mimo-deliverable-model" className="block text-sm font-medium">
                Mô hình soạn thảo (email, tóm tắt)
              </Label>
              <select
                id="mimo-deliverable-model"
                value={deliverableModel}
                onChange={(event) => setDeliverableModel(event.target.value)}
                disabled={isSaving}
                className="flex h-11 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              >
                <option value="">Theo mô hình mặc định</option>
                {MIMO_MODELS.map((model) => (
                  <option key={model} value={model}>
                    {MODEL_LABELS[model] ?? model}
                  </option>
                ))}
              </select>
              <p className="text-xs leading-5 text-muted-foreground">
                Dùng cho bản thảo email sao kê và tóm tắt hàng ngày.
              </p>
            </div>
          </div>

          <div className="flex flex-col gap-2 border-t pt-4 sm:flex-row sm:items-center">
            <Button type="submit" className="h-11 gap-2 sm:w-auto" disabled={isSaving}>
              {isSaving ? <Loader2 aria-hidden="true" className="size-4 animate-spin" /> : null}
              Lưu cấu hình
            </Button>
            <Button
              type="button"
              variant="outline"
              className="h-11 gap-2 sm:w-auto"
              onClick={() => void handleTestConnection()}
              disabled={
                testConnection.isPending ||
                // Designed state "missing key": nothing to test until a key is
                // saved. Every other state keeps this button enabled (retry).
                !keyConfigured ||
                hasUnsavedKey
              }
            >
              {testConnection.isPending ? (
                <Loader2 aria-hidden="true" className="size-4 animate-spin" />
              ) : (
                <PlugZap aria-hidden="true" className="size-4" />
              )}
              Kiểm tra kết nối
            </Button>
            <p className="text-xs leading-5 text-muted-foreground sm:ml-2">
              {!keyConfigured
                ? 'Nhập và lưu Khóa bí mật trước khi kiểm tra.'
                : hasUnsavedKey
                  ? 'Lưu thay đổi trước khi kiểm tra.'
                  : 'Kiểm tra khóa đã lưu với MiMo mà không phát sinh chi phí tạo nội dung.'}
            </p>
          </div>
        </form>
      </SettingsSection>

      <SettingsSection
        id="mimo-chat-roles"
        title="Quyền sử dụng trợ lý AI"
        description="Bật hoặc tắt riêng cho từng vai trò. Tắt sẽ ẩn trợ lý AI với vai trò đó kể từ yêu cầu tiếp theo."
      >
        <div className="space-y-3">
          <div className="flex items-center justify-between gap-4 rounded-lg border bg-background px-4 py-3.5">
            <div className="min-w-0">
              <Label htmlFor="mimo-chat-admin" className="text-sm font-medium">
                Quản trị viên
              </Label>
              <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
                Cho phép quản trị viên dùng trợ lý AI trong giao diện quản trị.
              </p>
            </div>
            <Switch
              id="mimo-chat-admin"
              checked={chatAdmin}
              onCheckedChange={(checked) => void handleRoleToggle('admin', checked)}
              disabled={togglingRole}
              aria-label="Bật trợ lý AI cho quản trị viên"
            />
          </div>
          <div className="flex items-center justify-between gap-4 rounded-lg border bg-background px-4 py-3.5">
            <div className="min-w-0">
              <Label htmlFor="mimo-chat-partner" className="text-sm font-medium">
                Đối tác
              </Label>
              <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
                Cho phép đối tác dùng trợ lý AI để tra cứu và gửi yêu cầu phê duyệt.
              </p>
            </div>
            <Switch
              id="mimo-chat-partner"
              checked={chatPartner}
              onCheckedChange={(checked) => void handleRoleToggle('partner', checked)}
              disabled={togglingRole}
              aria-label="Bật trợ lý AI cho đối tác"
            />
          </div>
        </div>
      </SettingsSection>

      {effectiveState ? (
        <SettingsSection
          id="mimo-test-result"
          title="Trạng thái kết nối"
          description="Kết quả lần kiểm tra kết nối gần nhất với MiMo."
        >
          <div
            role={effectiveState.code === 'ok' ? 'status' : 'alert'}
            aria-live="polite"
            className={cn(
              'rounded-lg border p-4 text-sm',
              effectiveState.code === 'ok'
                ? 'border-emerald-300 bg-emerald-50 text-emerald-800'
                : stateCopy?.tone === 'warning'
                  ? 'border-amber-300 bg-amber-50 text-amber-900'
                  : 'border-destructive/30 bg-destructive/5 text-destructive',
            )}
          >
            <div className="flex items-start gap-2">
              {effectiveState.code === 'ok' ? (
                <CheckCircle2 aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
              ) : (
                <AlertCircle aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
              )}
              <div className="min-w-0">
                <p className="font-semibold">
                  {effectiveState.code === 'ok'
                    ? `Kết nối MiMo thành công${effectiveState.latencyMs !== undefined ? ` — độ trễ ${effectiveState.latencyMs} ms` : ''}`
                    : (stateCopy?.title ?? 'Không thể kiểm tra kết nối')}
                </p>
                <p className="mt-1 break-words leading-5">{effectiveState.message}</p>
                {effectiveState.code === 'ok' || !stateCopy ? null : (
                  <p className="mt-1 leading-5 opacity-80">{stateCopy.hint}</p>
                )}
              </div>
            </div>
          </div>
        </SettingsSection>
      ) : null}
    </div>
  );
};
