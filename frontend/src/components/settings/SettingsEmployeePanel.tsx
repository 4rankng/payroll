import { useId, useEffect, useState } from 'react';

import { SettingsSection } from '@/components/settings/SettingsSection';

import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { useCreateSetting, useMultipleSettings, useUpdateSetting } from '@/hooks/api/useSettings';
import type { Setting } from '@/types/api/settings.types';

const MOBILE_REQUIRED_KEY = 'employee.mobile_required';

interface SettingToggleRowProps {
  title: string;
  description: string;
  checked: boolean;
  originalChecked: boolean;
  onCheckedChange: (checked: boolean) => void;
  onSave: () => void;
  onReset: () => void;
  isSaving: boolean;
  isDisabled?: boolean;
}

// Mirrors SettingsGeneralPanel's SettingToggleCard (Switch + Lưu/Hoàn tác
// shown only while dirty) so boolean settings share one save/reset UX.
const SettingToggleRow = ({
  title,
  description,
  checked,
  originalChecked,
  onCheckedChange,
  onSave,
  onReset,
  isSaving,
  isDisabled = false,
}: SettingToggleRowProps) => {
  const generatedId = useId().replace(/:/g, '');
  const switchId = `setting-toggle-${generatedId}`;
  const descriptionId = `${switchId}-description`;
  const isDirty = checked !== originalChecked;

  return (
    <div
      data-slot="setting-row"
      className={`group min-w-0 px-4 py-4 transition-colors sm:px-5 ${isDirty ? 'bg-warning/5' : ''}`}
    >
      <div className="grid min-w-0 gap-3 md:grid-cols-[minmax(0,1fr)_minmax(15rem,20rem)] md:items-start md:gap-6">
        <div className="min-w-0 space-y-1 md:py-2">
          <Label
            htmlFor={switchId}
            className="block break-words text-[13px] font-medium leading-5 text-foreground"
          >
            {title}
          </Label>
          <p id={descriptionId} className="break-words text-xs leading-4 text-muted-foreground">
            {description}
          </p>
        </div>
        <div className="flex min-w-0 items-center gap-2 md:justify-end md:pt-2">
          <Switch
            id={switchId}
            checked={checked}
            onCheckedChange={onCheckedChange}
            aria-describedby={descriptionId}
            disabled={isSaving || isDisabled}
          />
          {isDirty && (
            <>
              <Button size="sm" onClick={onSave} disabled={isSaving}>
                Lưu
              </Button>
              <Button size="sm" variant="outline" onClick={onReset} disabled={isSaving}>
                Hoàn tác
              </Button>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

/**
 * SettingsEmployeePanel — employee-creation policy settings.
 *
 * `employee.mobile_required` is a boolean row: 'true'/'false' strings, and a
 * missing row means OFF. The row may not exist yet, so saving upserts (the
 * same update-else-create pattern as useSettingsForm's upsertStringSetting).
 */
export const SettingsEmployeePanel = () => {
  const { data: settings, isLoading, isError, refetch } = useMultipleSettings([MOBILE_REQUIRED_KEY]);
  const updateMutation = useUpdateSetting();
  const createMutation = useCreateSetting();

  const [checked, setChecked] = useState(false);
  const [originalChecked, setOriginalChecked] = useState(false);

  useEffect(() => {
    if (!settings) return;
    const setting = settings.find((s?: Setting) => s?.key === MOBILE_REQUIRED_KEY);
    const enabled = setting?.value === 'true';
    setChecked(enabled);
    setOriginalChecked(enabled);
  }, [settings]);

  const isSaving = updateMutation.isPending || createMutation.isPending;

  const handleSave = () => {
    const setting = settings?.find((s?: Setting) => s?.key === MOBILE_REQUIRED_KEY);
    const value = checked ? 'true' : 'false';
    const onSaved = () => setOriginalChecked(checked);

    if (setting?.id) {
      updateMutation.mutate(
        { id: setting.id, data: { value, value_type: 'boolean' } },
        { onSuccess: onSaved },
      );
    } else {
      createMutation.mutate(
        { key: MOBILE_REQUIRED_KEY, value, value_type: 'boolean' },
        { onSuccess: onSaved },
      );
    }
  };

  return (
    <div className="space-y-3">
      <SettingsSection kicker="Chính sách" title="Nhân viên">
        {isError ? (
          <div role="alert" className="flex min-w-0 flex-wrap items-center justify-between gap-3 px-4 py-4 sm:px-5">
            <p className="min-w-0 break-words text-sm leading-relaxed text-destructive">
              Không thể tải cài đặt nhân viên. Vui lòng thử lại.
            </p>
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              Thử lại
            </Button>
          </div>
        ) : (
          <SettingToggleRow
            title="Bắt buộc số điện thoại khi thêm nhân viên mới"
            description="Khi bật, mọi luồng tạo nhân viên mới (thêm tay, import Excel, import bảng công) đều phải có số điện thoại. Lưu ý: số điện thoại trùng luôn bị chặn ở mọi luồng."
            checked={checked}
            originalChecked={originalChecked}
            onCheckedChange={setChecked}
            onSave={handleSave}
            onReset={() => setChecked(originalChecked)}
            isSaving={isSaving}
            isDisabled={isLoading}
          />
        )}
      </SettingsSection>
    </div>
  );
};

export default SettingsEmployeePanel;
