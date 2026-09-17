import { type LucideIcon, Building2, Landmark, Percent } from 'lucide-react';
import { useId, type ReactNode } from 'react';

import { SettingCard } from '@/components/settings/SettingCard';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import type { SettingsFormState } from '@/hooks/settings/useSettingsForm';

interface SettingsSectionProps {
  icon: LucideIcon;
  title: string;
  description: string;
  children: ReactNode;
}

interface SettingToggleCardProps {
  title: string;
  description: string;
  checked: boolean;
  originalChecked: boolean;
  onCheckedChange: (checked: boolean) => void;
  onSave: () => void;
  onReset: () => void;
  isSaving: boolean;
}

const SettingToggleCard = ({
  title,
  description,
  checked,
  originalChecked,
  onCheckedChange,
  onSave,
  onReset,
  isSaving,
}: SettingToggleCardProps) => {
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
            className="block break-words text-sm font-semibold leading-5 text-foreground"
          >
            {title}
          </Label>
          <p id={descriptionId} className="break-words text-sm leading-5 text-muted-foreground">
            {description}
          </p>
        </div>
        <div className="flex min-w-0 items-center gap-2 md:justify-end md:pt-2">
          <Switch
            id={switchId}
            checked={checked}
            onCheckedChange={onCheckedChange}
            aria-describedby={descriptionId}
            disabled={isSaving}
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

const SettingsSection = ({ icon: Icon, title, description, children }: SettingsSectionProps) => {
  const generatedId = useId().replace(/:/g, '');
  const headingId = `settings-section-${generatedId}`;

  return (
    <section
      aria-labelledby={headingId}
      className="grid gap-4 border-b py-6 first:pt-0 last:border-b-0 last:pb-0 lg:grid-cols-[minmax(12rem,0.65fr)_minmax(0,1.35fr)] lg:gap-8"
    >
      <div className="flex min-w-0 items-start gap-3 lg:pt-4">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-primary/10 bg-primary/5 text-primary">
          <Icon aria-hidden="true" className="h-4 w-4" />
        </div>
        <div className="min-w-0 pt-0.5">
          <h2 id={headingId} className="text-base font-semibold leading-6 text-foreground">
            {title}
          </h2>
          <p className="mt-0.5 text-sm leading-5 text-muted-foreground">{description}</p>
        </div>
      </div>

      <div
        data-slot="settings-group"
        className="min-w-0 divide-y overflow-hidden rounded-xl border bg-card"
      >
        {children}
      </div>
    </section>
  );
};

interface SettingsGeneralPanelProps {
  form: SettingsFormState;
}

export const SettingsGeneralPanel = ({ form }: SettingsGeneralPanelProps) => (
  <div className="space-y-0">
    <SettingsSection
      icon={Percent}
      title="Giới hạn thanh toán"
      description="Kiểm soát số tiền và tỷ lệ ngân sách được phép giải ngân."
    >
      <SettingCard
        title="Giới hạn tổng tiền mỗi file Chuyển lô"
        description="Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này."
        value={form.bulkTransferWorkbookLimitVnd}
        originalValue={form.originalBulkTransferWorkbookLimitVnd}
        onChange={form.setBulkTransferWorkbookLimitVnd}
        onSave={form.handleSaveBulkTransferWorkbookLimitVnd}
        onReset={() =>
          form.setBulkTransferWorkbookLimitVnd(form.originalBulkTransferWorkbookLimitVnd)
        }
        isDirty={
          form.bulkTransferWorkbookLimitVnd !== form.originalBulkTransferWorkbookLimitVnd
        }
        isSaving={form.isSaving}
        displayMode="currency-slider"
        min="100000000"
        max="500000000"
        errorMessage={form.bulkTransferWorkbookLimitSaveError}
        unavailableMessage={form.bulkTransferWorkbookLimitUnavailableMessage}
        onRetry={form.retryLoading}
      />
      <SettingCard
        title="Tỷ lệ trả lương tuần"
        description="Phần trăm ngân sách được phép trả theo chu kỳ tuần."
        value={form.weeklyPaymentPercentage}
        originalValue={form.originalWeeklyPayment}
        onChange={form.setWeeklyPaymentPercentage}
        onSave={form.handleSaveWeeklyPayment}
        onReset={() => form.setWeeklyPaymentPercentage(form.originalWeeklyPayment)}
        isDirty={form.weeklyPaymentPercentage !== form.originalWeeklyPayment}
        isSaving={form.isSaving}
        type="number"
        suffix="%"
      />
      <SettingCard
        title="Tỷ lệ trả lương tháng"
        description="Phần trăm ngân sách được phép trả theo chu kỳ tháng."
        value={form.monthlyPaymentPercentage}
        originalValue={form.originalMonthlyPayment}
        onChange={form.setMonthlyPaymentPercentage}
        onSave={form.handleSaveMonthlyPayment}
        onReset={() => form.setMonthlyPaymentPercentage(form.originalMonthlyPayment)}
        isDirty={form.monthlyPaymentPercentage !== form.originalMonthlyPayment}
        isSaving={form.isSaving}
        type="number"
        suffix="%"
      />
      <SettingCard
        title="Tỷ lệ ứng lương tự chấm công"
        description="Phần trăm tiền công đã ghi nhận được cộng vào hạn mức ứng lương."
        value={form.selfCheckInAdvancePercentage}
        originalValue={form.originalSelfCheckInAdvancePercentage}
        onChange={form.setSelfCheckInAdvancePercentage}
        onSave={form.handleSaveSelfCheckInAdvancePercentage}
        onReset={() =>
          form.setSelfCheckInAdvancePercentage(form.originalSelfCheckInAdvancePercentage)
        }
        isDirty={
          form.selfCheckInAdvancePercentage !== form.originalSelfCheckInAdvancePercentage
        }
        isSaving={form.isSaving}
        type="number"
        min={1}
        max={100}
        step={1}
        suffix="%"
      />
      <SettingCard
        title="Thời gian chờ ứng lương tự chấm công sau khi tan ca"
        description="Số giờ chờ trước khi tiền công của ca hoàn tất được cộng vào hạn mức ứng lương."
        value={form.selfCheckInAdvanceHoldHours}
        originalValue={form.originalSelfCheckInAdvanceHoldHours}
        onChange={form.setSelfCheckInAdvanceHoldHours}
        onSave={form.handleSaveSelfCheckInAdvanceHoldHours}
        onReset={() =>
          form.setSelfCheckInAdvanceHoldHours(form.originalSelfCheckInAdvanceHoldHours)
        }
        isDirty={
          form.selfCheckInAdvanceHoldHours !== form.originalSelfCheckInAdvanceHoldHours
        }
        isSaving={form.isSaving}
        type="number"
        min={0}
        max={720}
        step={1}
        wholeNumber
        suffix="giờ"
      />
    </SettingsSection>

    <SettingsSection
      icon={Building2}
      title="Thông tin đối tác"
      description="Tên doanh nghiệp hiển thị trong các nghiệp vụ tạm ứng."
    >
      <SettingCard
        title="Tên khách hàng"
        description="Tên công ty đối tác sử dụng dịch vụ tạm ứng."
        value={form.partnerCompany}
        originalValue={form.originalPartnerCompany}
        onChange={form.setPartnerCompany}
        onSave={form.handleSavePartnerCompany}
        onReset={() => form.setPartnerCompany(form.originalPartnerCompany)}
        isDirty={form.partnerCompany !== form.originalPartnerCompany}
        isSaving={form.isSaving}
      />
    </SettingsSection>

    <SettingsSection
      icon={Landmark}
      title="Tài khoản nhận chuyển khoản"
      description="Thông tin thụ hưởng in trên email sao kê và file Excel đính kèm."
    >
      <SettingToggleCard
        title="Hiện tài khoản nhận chuyển khoản"
        description="Khi tắt, sao kê (email và file Excel) chỉ là thông báo đối chiếu, không kèm thông tin chuyển khoản."
        checked={form.transferBankVisible}
        originalChecked={form.originalTransferBankVisible}
        onCheckedChange={form.setTransferBankVisible}
        onSave={form.handleSaveTransferBankVisible}
        onReset={() => form.setTransferBankVisible(form.originalTransferBankVisible)}
        isSaving={form.isSaving}
      />
      <SettingCard
        title="Chủ tài khoản"
        description="Tên chủ tài khoản thụ hưởng in trên sao kê."
        value={form.transferBankHolder}
        originalValue={form.originalTransferBankHolder}
        onChange={form.setTransferBankHolder}
        onSave={form.handleSaveTransferBank}
        onReset={() => form.setTransferBankHolder(form.originalTransferBankHolder)}
        isDirty={form.transferBankHolder !== form.originalTransferBankHolder}
        isSaving={form.isSaving}
      />
      <SettingCard
        title="Số tài khoản"
        description="Số tài khoản nhận chuyển khoản in trên sao kê."
        value={form.transferBankNumber}
        originalValue={form.originalTransferBankNumber}
        onChange={form.setTransferBankNumber}
        onSave={form.handleSaveTransferBank}
        onReset={() => form.setTransferBankNumber(form.originalTransferBankNumber)}
        isDirty={form.transferBankNumber !== form.originalTransferBankNumber}
        isSaving={form.isSaving}
        displayMode="account-number"
      />
      <SettingCard
        title="Ngân hàng"
        description="Tên ngân hàng nhận chuyển khoản in trên sao kê."
        value={form.transferBankName}
        originalValue={form.originalTransferBankName}
        onChange={form.setTransferBankName}
        onSave={form.handleSaveTransferBank}
        onReset={() => form.setTransferBankName(form.originalTransferBankName)}
        isDirty={form.transferBankName !== form.originalTransferBankName}
        isSaving={form.isSaving}
      />
    </SettingsSection>
  </div>
);
